package govaluateplus

import (
	"errors"
	"fmt"
	"strings"
)

const isoDateFormat string = "2006-01-02T15:04:05.999999999Z0700"
const shortCircuitHolder int = -1

var DUMMY_PARAMETERS = MapParameters(map[string]interface{}{})

/*
	EvaluableExpression represents a set of ExpressionTokens which, taken together,
	are an expression that can be evaluated down into a single value.
*/
type EvaluableExpression struct {

	/*
		Represents the query format used to output dates. Typically only used when creating SQL or Mongo queries from an expression.
		Defaults to the complete ISO8601 format, including nanoseconds.
	*/
	QueryDateFormat string

	/*
		Whether or not to safely check types when evaluating.
		If true, this library will return error messages when invalid types are used.
		If false, the library will panic when operators encounter types they can't use.

		This is exclusively for users who need to squeeze every ounce of speed out of the library as they can,
		and you should only set this to false if you know exactly what you're doing.
	*/
	ChecksTypes bool

	tokens           []ExpressionToken
	evaluationStages *evaluationStage
	inputExpression  string
}

/*
	Parses a new EvaluableExpression from the given [expression] string.
	Returns an error if the given expression has invalid syntax.
*/
func NewEvaluableExpression(expression string) (*EvaluableExpression, error) {

	functions := make(map[string]ExpressionFunction)
	return NewEvaluableExpressionWithFunctions(expression, functions)
}

/*
	Similar to [NewEvaluableExpression], except that instead of a string, an already-tokenized expression is given.
	This is useful in cases where you may be generating an expression automatically, or using some other parser (e.g., to parse from a query language)
*/
func NewEvaluableExpressionFromTokens(tokens []ExpressionToken) (*EvaluableExpression, error) {

	var ret *EvaluableExpression
	var err error

	ret = new(EvaluableExpression)
	ret.QueryDateFormat = isoDateFormat

	err = checkBalance(tokens)
	if err != nil {
		return nil, err
	}

	err = checkExpressionSyntax(tokens)
	if err != nil {
		return nil, err
	}

	ret.tokens, err = optimizeTokens(tokens)
	if err != nil {
		return nil, err
	}

	ret.evaluationStages, err = planStages(ret.tokens)
	if err != nil {
		return nil, err
	}

	ret.ChecksTypes = true
	return ret, nil
}

/*
	Similar to [NewEvaluableExpression], except enables the use of user-defined functions.
	Functions passed into this will be available to the expression.
*/
func NewEvaluableExpressionWithFunctions(expression string, functions map[string]ExpressionFunction) (*EvaluableExpression, error) {

	var ret *EvaluableExpression
	var err error

	ret = new(EvaluableExpression)
	ret.QueryDateFormat = isoDateFormat
	ret.inputExpression = expression

	ret.tokens, err = parseTokens(expression, functions)
	if err != nil {
		return nil, err
	}

	err = checkBalance(ret.tokens)
	if err != nil {
		return nil, err
	}

	err = checkExpressionSyntax(ret.tokens)
	if err != nil {
		return nil, err
	}

	ret.tokens, err = optimizeTokens(ret.tokens)
	if err != nil {
		return nil, err
	}

	ret.evaluationStages, err = planStages(ret.tokens)
	if err != nil {
		return nil, err
	}

	ret.ChecksTypes = true
	return ret, nil
}

/*
	Same as `Eval`, but automatically wraps a map of parameters into a `govalute.Parameters` structure.
*/
func (ee EvaluableExpression) Evaluate(parameters map[string]interface{}) (interface{}, error) {

	if parameters == nil {
		return ee.Eval(nil)
	}
	param := escapeBackslash(parameters)
	return ee.Eval(MapParameters(param))
}

// escapeBackslash 如果param中带有'\',在遍历ast求值阶段干掉'\'，这么做是为了兼容生成token阶段，每个token都会去掉'\'的做法
func escapeBackslash(param map[string]interface{}) map[string]interface{} {
	res := make(map[string]interface{}, len(param))

	for k, v := range param {
		switch v.(type) {
		case string:
			res[k] = removeBackSlash(v.(string))
		default:
			res[k] = v
		}
	}

	return res
}

func removeBackSlash(source string) string {
	return strings.ReplaceAll(source, "\\", "")
}

/*
	Runs the entire expression using the given [parameters].
	e.g., If the expression contains a reference to the variable "foo", it will be taken from `parameters.Get("foo")`.

	This function returns errors if the combination of expression and parameters cannot be run,
	such as if a variable in the expression is not present in [parameters].

	In all non-error circumstances, this returns the single value result of the expression and parameters given.
	e.g., if the expression is "1 + 1", this will return 2.0.
	e.g., if the expression is "foo + 1" and parameters contains "foo" = 2, this will return 3.0
*/
func (ee EvaluableExpression) Eval(parameters Parameters) (interface{}, error) {

	if ee.evaluationStages == nil {
		return nil, nil
	}

	if parameters != nil {
		parameters = &sanitizedParameters{parameters}
	} else {
		parameters = DUMMY_PARAMETERS
	}

	return ee.evaluateStage(ee.evaluationStages, parameters)
}

func (ee EvaluableExpression) evaluateStage(stage *evaluationStage, parameters Parameters) (interface{}, error) {

	var left, right interface{}
	var err error

	if stage.leftStage != nil {
		left, err = ee.evaluateStage(stage.leftStage, parameters)
		if err != nil {
			return nil, err
		}
	}

	if stage.isShortCircuitable() {
		switch stage.symbol {
		case AND:
			if left == false {
				return false, nil
			}
		case OR:
			if left == true {
				return true, nil
			}
		case COALESCE:
			if left != nil {
				return left, nil
			}

		case TERNARY_TRUE:
			if left == false {
				right = shortCircuitHolder
			}
		case TERNARY_FALSE:
			if left != nil {
				right = shortCircuitHolder
			}
		}
	}

	if right != shortCircuitHolder && stage.rightStage != nil {
		right, err = ee.evaluateStage(stage.rightStage, parameters)
		if err != nil {
			return nil, err
		}
	}

	if ee.ChecksTypes {
		if stage.typeCheck == nil {

			err = typeCheck(stage.leftTypeCheck, left, stage.symbol, stage.typeErrorFormat)
			if err != nil {
				return nil, err
			}

			err = typeCheck(stage.rightTypeCheck, right, stage.symbol, stage.typeErrorFormat)
			if err != nil {
				return nil, err
			}
		} else {
			// special case where the type check needs to know both sides to determine if the operator can handle it
			if !stage.typeCheck(left, right) {
				errorMsg := fmt.Sprintf(stage.typeErrorFormat, left, stage.symbol.String())
				return nil, errors.New(errorMsg)
			}
		}
	}

	return stage.operator(left, right, parameters)
}

func typeCheck(check stageTypeCheck, value interface{}, symbol OperatorSymbol, format string) error {

	if check == nil {
		return nil
	}

	if check(value) {
		return nil
	}

	errorMsg := fmt.Sprintf(format, value, symbol.String())
	return errors.New(errorMsg)
}

/*
	Returns an array representing the ExpressionTokens that make up this expression.
*/
func (ee EvaluableExpression) Tokens() []ExpressionToken {

	return ee.tokens
}

/*
	Returns the original expression used to create this EvaluableExpression.
*/
func (ee EvaluableExpression) String() string {

	return ee.inputExpression
}

/*
	Returns an array representing the variables contained in this EvaluableExpression.
*/
func (ee EvaluableExpression) Vars() []string {
	var varlist []string
	for _, val := range ee.Tokens() {
		if val.Kind == VARIABLE {
			varlist = append(varlist, val.Value.(string))
		}
	}
	return varlist
}
