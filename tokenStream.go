package govaluateplus

type tokenStream struct {
	tokens      []ExpressionToken
	index       int
	tokenLength int
}

func newTokenStream(tokens []ExpressionToken) *tokenStream {

	var ret *tokenStream

	ret = new(tokenStream)
	ret.tokens = tokens
	ret.tokenLength = len(tokens)
	return ret
}

func (ts *tokenStream) rewind() {
	ts.index -= 1
}

func (ts *tokenStream) next() ExpressionToken {

	var token ExpressionToken

	token = ts.tokens[ts.index]

	ts.index += 1
	return token
}

func (ts tokenStream) hasNext() bool {

	return ts.index < ts.tokenLength
}
