package main

import "fmt"

// statistics variables
var zzgoent = 0  // count goto entries
var zzgobest = 0 // count entries saved by default gotos
var zzshift = 0  // count shifts
var zzexcp = 0   // count exceptions
var zzrrconf = 0 // count educe-reduce conflicts
var zzsrconf = 0 // count shift-reduce conflict

var maxRed int       // default reduction of a state
var defReds []int    // default reductions of states
var exca []int       // exception action table
var stateTable []row // for checking errors
type row struct {
	actions []int
	defAct  int
}

var shifts [][]int // first dimension is state, second dimension is pairs of (terminal symbol, next state)

// handle shift/reduce conflict in state s, between rule r and terminal symbol t
func handleShiftReduceConflict(r, t, s int) {
	lp := aptPrd[r]
	lt := aptTerm[t]
	if prec(lp) == 0 || prec(lt) == 0 { // shift
		fmt.Fprintf(foutput, "\n%d: shift/reduce conflict (shift %d(%d), red'n %d(%d)) on %s",
			s, trans[t], prec(lt), r, prec(lp), terms[t].name)
		zzsrconf++
		return
	}
	if prec(lt) > prec(lp) { // shift
		return
	}
	as := LASC // check associativity
	if prec(lt) == prec(lp) {
		as = asc(lt)
	}
	switch as {
	case BASC:
		trans[t] = ERRCODE
	case LASC: // reduce
		trans[t] = -r
	}
	// shift in other cases
}

// compute reductions after closing a state
func computeReductions(i int) {
	red := 0
	// check if an item induces reduction
	// handle reduce/reduce conflict or shift/reduce conflict
	computeReductionItem := func(itemI item) {
		first := itemI.first
		if first > 0 {
			return
		}
		red = -first
		lk := itemI.lkset
		for j, s := range trans[:termN+1] {
			if !lk.check(j) {
				continue
			}
			if s == 0 {
				trans[j] = first
			} else if s < 0 { // reduce/reduce conflict. don't need to check the case s == red!
				fmt.Fprintf(foutput, "\n %v: reduce/reduce conflict  (red'ns %v and %v) on %v",
					i, -s, red, terms[j].name)
				if -s > red { // favor rule higher in grammar
					trans[j] = red
				}
				zzrrconf++
			} else { // shift/reduce conflict
				handleShiftReduceConflict(red, j, i)
			}
		}
	}
	// only consider kernel and epsilon items for the reduction of state i
	for _, itemI := range kernls[kernlp[i]:kernlp[i+1]] {
		computeReductionItem(itemI)
	}
	for _, itemI := range epsilons[i] {
		computeReductionItem(itemI)
	}
	// compute reduction with maximum count
	maxRed = 0
	maxCount := 0
	clearCounts()
	for _, act := range trans[:termN+1] {
		if act < 0 {
			r := -act
			aptPrd[r] |= REDFLAG // mark that prd r can reduce
			counts[r]++
			if counts[r] > maxCount {
				maxCount = counts[r]
				maxRed = r
			}
		}
	}
	for i, act := range trans[:termN+1] { // clear default reduction cells.
		if act == -maxRed {
			trans[i] = 0
		}
	}
	defReds[i] = maxRed
}

// compute shifts after closing state i
func computeShifts(i int) {
	n := 0
	// counting the shifts
	for _, act := range trans[:termN+1] {
		if act > 0 && act != ACCEPTCODE && act != ERRCODE {
			n++
		}
	}
	row := make([]int, n*2) // pairs of (terminal symbol, next state)
	n = 0
	// this is repetitive, but more efficient for memory allocation
	for t, act := range trans[:termN+1] {
		if act > 0 && act != ACCEPTCODE && act != ERRCODE {
			row[n] = t
			n++
			row[n] = act
			n++
			zzshift++
		}
	}
	shifts[i] = row
}

// compute exception actions for state i
func computeExceptionActions(i int, exca *[]int) {
	hasExcp := false
	for j, s := range trans[:termN+1] {
		if s != 0 {
			if s < 0 {
				s = -s
			} else if s == ACCEPTCODE {
				s = -1
			} else if s == ERRCODE {
				s = 0
			} else {
				continue
			}
			if !hasExcp {
				hasExcp = true
				*exca = append(*exca, -1, i) // mark the beginning of exceptions for state i
			}
			*exca = append(*exca, j, s) // store the cause of exception
			zzexcp++
		}
	}
	if hasExcp {
		*exca = append(*exca, -2, defReds[i]) // store default reduction
		defReds[i] = -2                       // mark exception in default reductions
	}
}

// compute which symbol induces transition to a state.  use to check valid shift or goto
func computeCheck(tmp []int) {
	fill(tmp, nstate, -1000)
	for i := 0; i <= termN; i++ {
		for j := tstates[i]; j != 0; j = stateChain[j] {
			tmp[j] = i // shift
		}
	}
	for i := 0; i <= nontermN; i++ {
		for j := ntstates[i]; j != 0; j = stateChain[j] {
			tmp[j] = -i // goto
		}
	}
}

// write the state to foutput
func writeState(i int) {
	fmt.Fprintf(foutput, "\nstate %v\n", i)
	// print kernel items
	for _, a := range kernls[kernlp[i]:kernlp[i+1]] {
		fmt.Fprintf(foutput, "\t%s\n", a.string())
	}
	// print epsilons
	for _, itemI := range epsilons[i] {
		fmt.Fprintf(foutput, "\t%s\n", itemI.string())
	}
	// print accept, error, shift, reduce
	for t, s := range trans[:termN+1] {
		if s != 0 {
			fmt.Fprintf(foutput, "\n\t%s  ", terms[t].name)
			switch {
			case s < 0: // reduction that is not default
				fmt.Fprintf(foutput, "reduce %d (src line %d)", -s, rlines[-s])
			case s == ACCEPTCODE:
				fmt.Fprintf(foutput, "accept")
			case s == ERRCODE:
				fmt.Fprintf(foutput, "error")
			default: // shift
				fmt.Fprintf(foutput, "shift %v", s)
			}
		}
	}
	// print default reduction or error
	if maxRed != 0 {
		fmt.Fprintf(foutput, "\n\t.  reduce %d (src line %d)\n\n", maxRed, rlines[maxRed])
	} else {
		fmt.Fprintf(foutput, "\n\t.  error\n\n")
	}
	// print gotos
	for n, s := range trans[termN+1 : termN+1+nontermN] {
		if s != 0 {
			fmt.Fprintf(foutput, "\t%s  goto %d\n", nonterms[n+1].name, s)
		}
	}
	foutput.Flush()
}

// fill the row i of state table
func writeStateTable(i int) {
	actions := append([]int{}, trans[:termN+1+nontermN]...)
	defAct := ERRCODE
	if maxRed != 0 {
		defAct = -maxRed
	}
	stateTable[i] = row{actions, defAct}
}

// process generated states. compute shifts, reductions, gotos and exception action table
func procStates() {
	defReds = make([]int, nstate)
	shifts = make([][]int, nstate)
	hasError := len(errors) > 0
	if hasError {
		stateTable = make([]row, nstate)
	}
	for i := 0; i < nstate; i++ {
		closure0(i)
		fill(trans, termN+1+nontermN, 0)
		kernlp[nstate+1] = kernlp[nstate] // temporary initally holds 0 items
		for j, w := range wSet[:cwp] {
			first := w.item.first
			if first > 1 && first < NTBASE && trans[first] == 0 {
				for _, v := range wSet[j:cwp] {
					if first == v.item.first {
						addKernItem(v.item)
					}
				}
				trans[first] = retrieveState(first) // a shift
			} else if first > NTBASE {
				first -= NTBASE
				trans[first+termN] = actionStore[gotoIdx[i]+first] // a goto
			}
		}
		if i == 1 {
			trans[1] = ACCEPTCODE
		}
		computeReductions(i)
		computeShifts(i)
		computeExceptionActions(i, &exca)
		writeState(i)
		if hasError {
			writeStateTable(i)
		}
	}
	checkRules()
}

// check if any rule never reduced
func checkRules() {
	nred := 0
	for i := 1; i < nprod; i++ {
		f := aptPrd[i]
		if f&REDFLAG == 0 {
			fmt.Fprintf(foutput, "rule not reduced: %s\n", prod2str(prods[i]))
			fmt.Printf("rule %s not reduced:\n", prod2str(prods[i]))
			nred++
		}
	}

	if nred != 0 {
		fmt.Printf("%d rules never reduced\n", nred)
	}
}
