package govaluateplus

import (
	"bytes"
)

/*
	Holds a series of "transactions" which represent each token as it is output by an outputter (such as ToSQLQuery()).
	Some outputs (such as SQL) require a function call or non-c-like syntax to represent an expression.
	To accomplish this, this struct keeps track of each translated token as it is output, and can return and rollback those transactions.
*/
type expressionOutputStream struct {
	transactions []string
}

func (eos *expressionOutputStream) add(transaction string) {
	eos.transactions = append(eos.transactions, transaction)
}

func (eos *expressionOutputStream) rollback() string {

	index := len(eos.transactions) - 1
	ret := eos.transactions[index]

	eos.transactions = eos.transactions[:index]
	return ret
}

func (eos *expressionOutputStream) createString(delimiter string) string {

	var retBuffer bytes.Buffer
	var transaction string

	penultimate := len(eos.transactions) - 1

	for i := 0; i < penultimate; i++ {

		transaction = eos.transactions[i]

		retBuffer.WriteString(transaction)
		retBuffer.WriteString(delimiter)
	}
	retBuffer.WriteString(eos.transactions[penultimate])

	return retBuffer.String()
}
