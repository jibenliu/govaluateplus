package govaluateplus

type lexerStream struct {
	source   []rune
	position int
	length   int
}

func newLexerStream(source string) *lexerStream {

	var ret *lexerStream
	var runes []rune

	for _, character := range source {
		runes = append(runes, character)
	}

	ret = new(lexerStream)
	ret.source = runes
	ret.length = len(runes)
	return ret
}

func (ls *lexerStream) readCharacter() rune {

	var character rune

	character = ls.source[ls.position]
	ls.position += 1
	return character
}

func (ls *lexerStream) rewind(amount int) {
	ls.position -= amount
}

func (ls lexerStream) canRead() bool {
	return ls.position < ls.length
}
