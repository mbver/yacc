package main

import (
	"fmt"
	"strings"
)

var prods [][]int     // holds all production rules declared in input
var yields [][][]int  // 1st dimension is the non-terminal symbol, 2nd dimension is prds started with it. are we talking about corn? 🍿
var empty []bool      // to check whether a symbol is nullable
var firstSets []lkset // the first sets for each symbol

// get id of a production
func id(prd []int) int {
	return -prd[len(prd)-1]
}

// compute production-yield for each non-terminal symbol
func copyFields() {
	yields = make([][][]int, nontermN+1)

	yld := make([][]int, nprod)

	if false {
		for j, t := range nonterms[:nontermN+1] {
			fmt.Printf("nonterms[%d] = %s\n", j, t.name)
		}
		for j, prd := range prods[:nprod] {
			fmt.Printf("allPrds[%d][0] = %d+NTBASE\n", j, prd[0]-NTBASE)
		}
	}

	fatfl = 0 // catch all problematic non-terminal symbols
	for i, s := range nonterms[:nontermN+1] {
		n := 0
		c := i + NTBASE
		for _, prd := range prods[:nprod] {
			if prd[0] == c {
				yld[n] = prd
				n++
			}
		}
		if n == 0 {
			errorf("nonterminal %s doesn't yield any production", s.name)
			continue
		}
		yields[i] = make([][]int, n)
		copy(yields[i], yld)
	}
	fatfl = 1
	if nerrors != 0 {
		summary()
		exit(1)
	}
}

// to check if a non-term symbol is nullable. use in computing firstsets
func computeEmpty() {
	through := make([]bool, nontermN+1) // a symbol can derive through the end if all symbols on its RHS can derive through the end

thru: // check if any symbol can not derive through the end
	for {
		for _, prd := range prods[:nprod] {
			if through[prd[0]-NTBASE] { // already checked
				continue
			}
			i := 1
			for _, s := range prd[1 : len(prd)-1] {
				if s >= NTBASE && !through[s-NTBASE] {
					break
				}
				i++
			}
			if i == len(prd)-1 {
				through[prd[0]-NTBASE] = true
				continue thru // discover new symbol. search all over again
			}
		}
		break
	}
	fatfl = 0 // catch all problematic non-terminal symbols

	var s SymNonterm
	for i := 1; i <= nontermN; i++ {
		s = nonterms[i]
		if !through[i] {
			errorf("nonterminal " + s.name + " can not derive through the end")
		}
	}
	fatfl = 1

	if nerrors != 0 {
		summary()
		exit(1)
	}
	for i := range through {
		through[i] = false
	}
	// reuse array
	empty = through // a LHS symbol is empty if there's a rule of its yield that RHS is ɛ or all the RHS symbol is empty

emp:
	for {
	nextPrd:
		for _, prd := range prods[1:nprod] {
			if empty[prd[0]-NTBASE] { // already checked
				continue
			}

			for _, s := range prd[1 : len(prd)-1] {
				if s < NTBASE || !empty[s-NTBASE] {
					continue nextPrd
				}
			}
			empty[prd[0]-NTBASE] = true
			// discover a new empty non-term. search all over again from first prd
			continue emp
		}
		return // done searching
	}
}

// compute firstset for each non-terminal symbol
func computeFirstsets() {
	firstSets = make([]lkset, nontermN+1)
	var yield [][]int
	// fill firstset with terminal appears first on RHS
	for i := range nonterms[:nontermN+1] {
		firstSets[i] = newLkset()
		yield = yields[i] // prds started with s

		for _, prd := range yield {
			for _, ch := range prd[1 : len(prd)-1] {
				if ch < NTBASE { // terminal symbol, set it in firstset, move on to next prd
					firstSets[i].set(ch)
					break
				}
				if !empty[ch-NTBASE] { // non-nullable non-terminal symbol. move on to next prd
					break
				}
			}
		}
	}

	// transitivity: if a non-term on RHS can appear first, its firstset will be unioned with LHS's
	changed := true // changed reflects if firstset of any non-term changed through transitivity. if so, search the whole prd list again
	for changed {
		changed = false
		for i := range nonterms[:nontermN+1] {
			yield = yields[i]
			for _, prd := range yield {
				for _, ch := range prd[1 : len(prd)-1] {
					if ch < NTBASE { // terminal symbol appears first. break
						break
					}
					ch -= NTBASE
					changed = firstSets[i].union(firstSets[ch]) || changed
					if !empty[ch] {
						break
					}
				}
			}
		}
	}
}

// LHS for a production
func computeLHS(tmp []int) {
	for i := 0; i < nprod; i++ {
		tmp[i] = prods[i][0] - NTBASE
	}
}

// number of RHS symbols for a production
func computeRHS(tmp []int) {
	for i := 0; i < nprod; i++ {
		tmp[i] = len(prods[i]) - 2
	}
}

// string representation of a production
func prod2str(p []int) string {
	buf := strings.Builder{}
	for i, s := range p[:len(p)-1] {
		buf.WriteString(symbolName(s))
		if i == 0 {
			buf.WriteString(": ")
		} else {
			buf.WriteString(" ")
		}
	}
	return buf.String()
}

// compute production yields, nullable non-terms and first_sets
func procProds() {
	copyFields()
	computeEmpty()
	computeFirstsets()
}
