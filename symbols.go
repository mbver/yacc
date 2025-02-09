package main

import (
	"strconv"
)

type SymTerm struct {
	name    string
	isConst bool
	value   int
}

type SymNonterm struct {
	name string
	typ  int // hold type value of non-term
}

var terms []SymTerm
var nonterms []SymNonterm

// store non-term into the nonterms
func defineNonTerminal(s string) int {
	nontermN++
	if nontermN >= len(nonterms) {
		extend(&nonterms, SYMINC)
	}
	nonterms[nontermN] = SymNonterm{name: s}
	return NTBASE + nontermN
}

// store terminal symbol into the terms
// value of single literal terminal is extracted from its character
// value of other terminal is assigned from extval. extval is increased after each used
func defineTerminal(s string) int {
	termN++
	if termN >= len(terms) {
		extend(&terms, SYMINC)
		extend(&aptTerm, SYMINC)
	}
	terms[termN].name = s
	terms[termN].isConst = true
	aptTerm[termN] = 0
	val := 0
	// single character literal term-symbol
	if s[0] == '\'' || s[0] == '"' {
		q, err := strconv.Unquote(s)
		if err != nil {
			errorf("invalid token: %s", err)
		}
		r := []rune(q)
		if len(r) != 1 {
			errorf("expect single character literal token, but got: %s", s)
		}
		val = int(r[0])
		if val == 0 {
			errorf("token value 0 is illegal")
		}
		terms[termN].isConst = false
	} else {
		val = extval
		extval++
		if s[0] == '$' {
			terms[termN].isConst = false
		}
	}
	terms[termN].value = val
	return termN
}

// find terminal symbol
func findTerminal(s string) int {
	for i, t := range terms[:termN+1] {
		if t.name == s {
			return i
		}
	}
	return -1
}

// find terminal s, insert if not found
func findInsertTerminal(s string) int {
	if i := findTerminal(s); i > 0 {
		return i
	}
	return defineTerminal(s)
}

func findNonterm(s string) int {
	for i, nt := range nonterms[:nontermN+1] {
		if nt.name == s {
			return i + NTBASE
		}
	}
	return -1
}

// find nonterm s, insert if not found
func findInsertNonTerminal(s string) int {
	if i := findNonterm(s); i >= 0 {
		return i
	}
	return defineNonTerminal(s)
}

func findSym(s string) int {
	if i := findTerminal(s); i > 0 {
		return i
	}
	return findNonterm(s)
}

// define initial symbols
func defineInitialSymbols() {
	defineTerminal("$end")
	extval = PRIVATE // all value defined later will be in PRIVATE range
	defineTerminal("error")
	defineTerminal("$unk")
	defineNonTerminal("$accept")
}

// name of a symbol
func symbolName(i int) string {
	if i >= NTBASE {
		return nonterms[i-NTBASE].name
	} else {
		return terms[i].name
	}
}
