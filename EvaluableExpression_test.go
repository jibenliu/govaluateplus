package govaluateplus

import (
	"fmt"
	"github.com/awalterschulze/gographviz"
	"testing"
)

func traverseNode(root *EvaluationStage, graph *gographviz.Graph) *gographviz.Graph {
	if root == nil {
		return nil
	}

	graph.AddNode("G", fmt.Sprintf("%+v", root.symbol), nil)
	traverseNode(root.leftStage, graph)
	traverseNode(root.rightStage, graph)

	return graph
}

func traverseEdge(root *EvaluationStage, graph *gographviz.Graph) *gographviz.Graph {
	if root == nil {
		return nil
	}
	if root.leftStage != nil {
		graph.AddEdge(fmt.Sprintf("%+v", root.symbol), fmt.Sprintf("%+v", root.leftStage.symbol), true, nil)
		traverseEdge(root.leftStage, graph)
	}
	if root.rightStage != nil {
		graph.AddEdge(fmt.Sprintf("%+v", root.symbol), fmt.Sprintf("%+v", root.rightStage.symbol), true, nil)
		traverseEdge(root.rightStage, graph)
	}
	return graph
}

func TestNewEvaluableExpression(t *testing.T) {
	a := "foo + 5 * boo"

	expression, err := NewEvaluableExpression(a)
	if err != nil {
		fmt.Println(err)
	}
	param := map[string]interface{}{
		"foo": "3",
		"boo": "2",
	}

	graphAst, _ := gographviz.ParseString(`digraph G {}`)
	graph := gographviz.NewGraph()
	if err := gographviz.Analyse(graphAst, graph); err != nil {
		panic(err)
	}

	// 有点问题...
	traverseNode(expression.evaluationStages, graph)
	traverseEdge(expression.evaluationStages, graph)
	output := graph.String()
	fmt.Println(output)

	res, err := expression.Evaluate(param)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(res)

}
