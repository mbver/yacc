package main

import (
	"fmt"
	"strings"
)

var YYLEXUNK = 3

// write something that make the readers scratch their head. 🤔
func summary() {
	fmt.Fprintf(foutput, "\n%v terminals, %v nonterminals\n", termN, nontermN+1)
	fmt.Fprintf(foutput, "%v grammar rules, %v/%v states\n", nprod, nstate, NSTATES)
	fmt.Fprintf(foutput, "%v shift/reduce, %v reduce/reduce conflicts reported\n", zzsrconf, zzrrconf)
	fmt.Fprintf(foutput, "%v working sets used\n", len(wSet))
	fmt.Fprintf(foutput, "memory: parser %v/%v\n", maxGotoIdx, ACTSIZE)
	fmt.Fprintf(foutput, "%v extra closures\n", zzclose-2*nstate)
	fmt.Fprintf(foutput, "%v shift entries, %v exceptions\n", zzshift, zzexcp)
	fmt.Fprintf(foutput, "%v goto entries\n", zzgoent)
	fmt.Fprintf(foutput, "%v entries saved by goto default\n", zzgobest)
	i := 0 // count 0 actions
	for _, a := range actionStore[:lastActIdx+1] {
		if a == 0 {
			i++
		}
	}
	fmt.Fprintf(foutput, "Optimizer space used: output %v/%v\n", lastActIdx+1, ACTSIZE)
	fmt.Fprintf(foutput, "%v table entries, %v zero\n", lastActIdx+1, i)
	fmt.Fprintf(foutput, "maximum spread: %v, maximum offset: %v\n", maxspr, maxoff)

	if zzsrconf != 0 || zzrrconf != 0 {
		fmt.Printf("\nconflicts: ")
		if zzsrconf != 0 {
			fmt.Printf("%v shift/reduce", zzsrconf)
		}
		if zzsrconf != 0 && zzrrconf != 0 {
			fmt.Printf(", ")
		}
		if zzrrconf != 0 {
			fmt.Printf("%v reduce/reduce", zzrrconf)
		}
		fmt.Printf("\n")
	}
}

// write an array to ftable
func writeArray(name string, s []int) {
	name = prefix + name
	ftable.WriteRune('\n')
	typ := minType(s)
	fmt.Fprintf(ftable, "var %s = [...]%s{", name, typ)
	for i, v := range s {
		if i%10 == 0 { // write 10 numbers on a line
			fmt.Fprintf(ftable, "\n\t")
		} else {
			ftable.WriteRune(' ')
		}
		fmt.Fprintf(ftable, "%d,", v)
	}
	fmt.Fprintf(ftable, "\n}\n")
}

// write out shifts, reduces and gotos to ftable
func writeActions() {
	ftable.WriteRune('\n')
	fmt.Fprintf(ftable, "const %sLast = %d\n", prefix, lastActIdx+1)
	writeArray("Act", actionStore[:lastActIdx+1])
	writeArray("Pshift", shiftIdx)
	writeArray("Pgo", pgoIdx)
	writeArray("Def", defReds)
	writeArray("Exca", exca)
	tmp := trans // reuse trans
	computeCheck(tmp)
	writeArray("Chk", tmp[:nstate])
	computeLHS(tmp)
	writeArray("R1", tmp[:nprod])
	computeRHS(tmp)
	writeArray("R2", tmp[:nprod])
}

// write LHS syms, num of RHS syms, terminal symbols
func writeTokens() {
	tmp := trans // reuse trans
	// terminal symbols with values from 0 to 256
	fill(tmp, 256, 0)
	maxVal := 0
	var val int
	for i := 1; i <= termN; i++ {
		val = terms[i].value
		if val >= 0 && val < 256 {
			if tmp[val] != 0 {
				fmt.Printf("yacc bug -- 2 different tokens with the same value: %s and %s\n", terms[i].name, terms[tmp[val]].name)
				nerrors++
			}
			tmp[val] = i
			if val > maxVal {
				maxVal = val
			}
		}
	}
	for i := 0; i <= maxVal; i++ {
		if tmp[i] == 0 {
			tmp[i] = YYLEXUNK
		}
	}
	writeArray("Tok1", tmp[:maxVal+1])

	// terminal symbols with values from PRIVATE to PRIVATE + 256
	fill(tmp, 256, 0)
	maxVal = 0
	val = 0
	for i := 1; i <= termN; i++ {
		val = terms[i].value - PRIVATE
		if val >= 0 && val < 256 {
			if tmp[val] != 0 {
				fmt.Printf("yacc bug -- 2 different tokens with the same value: %s and %s\n", terms[i].name, terms[tmp[val]].name)
				nerrors++
			}
			tmp[val] = i
			if val > maxVal {
				maxVal = val
			}
		}
	}
	writeArray("Tok2", tmp[:maxVal+1])
	ftable.WriteRune('\n')

	// terminal symbols with values between (256, PRIVATE) or above PRIVATE+256
	tmp = []int{}
	for i := 1; i <= termN; i++ {
		val = terms[i].value
		if val >= 0 && val < 256 {
			continue
		}
		if val >= PRIVATE && val < PRIVATE+256 {
			continue
		}
		tmp = append(tmp, val, i)
	}
	tmp = append(tmp, 0)
	writeArray("Tok3", tmp)
	fmt.Fprintf(ftable, "\n")
}

func writeErrMsgs() {
	fmt.Fprintf(ftable, "\n")
	fmt.Fprintf(ftable, "var %sErrorMessages = [...]struct {\n", prefix)
	fmt.Fprintf(ftable, "\tstate int\n")
	fmt.Fprintf(ftable, "\ttoken int\n")
	fmt.Fprintf(ftable, "\tmsg   string\n")
	fmt.Fprintf(ftable, "}{\n")
	for _, err := range errors {
		lineno = err.lineno
		state, token := runUntilErr(err.tokens) // oh no. it's horrible. praying...⛩️
		fmt.Fprintf(ftable, "\t{%v, %v, %s},\n", state, token, err.msg)
	}
	fmt.Fprintf(ftable, "}\n")
}

// start from state 0, simulate the run of state machine on err.tokens until we got an ERRCODE or ACCEPTCODE
func runUntilErr(tokens []string) (state, token int) {
	var stack []int
	i := 0
	token = -1
Loop:
	if token < 0 {
		token = findSym(tokens[i])
		if token < 0 {
			errorf("symbol not defined: %s", tokens[i])
			return
		}
		i++
	}

	row := stateTable[state] // possible transitions for a state

	c := token
	if token >= NTBASE {
		c = token - NTBASE + termN
	}
	action := row.actions[c] // transition with token as lookahead
	if action == 0 {
		action = row.defAct
	}

	switch {
	case action == ACCEPTCODE:
		errorf("tokens are accepted")
		return
	case action == ERRCODE:
		if token >= NTBASE {
			errorf("error at non-terminal token %s", terms[token].name)
		}
		return
	case action > 0: // shift
		stack = append(stack, state)
		state = action
		token = -1
		goto Loop
	default: // reduce
		prd := prods[-action]
		if rhsLen := len(prd) - 2; rhsLen > 0 {
			n := len(stack) - rhsLen
			state = stack[n-1] // state of the uncovered TOS. make more sense than stack[n]!!!
			stack = stack[:n]
		}
		if token >= 0 {
			i-- // put the token back for later use
		}
		token = prd[0] // the LHS
		goto Loop
	}
}

// write generated parser code from template
func writeParseCode() {
	parts := strings.SplitN(template, "yyImbedAct()", 2)
	fmt.Fprintf(ftable, "%v", parts[0])
	ftable.Write(fcode.Bytes()) // write imbedded actions
	fmt.Fprintf(ftable, "%v", parts[1])
}

// write the generated parser to ftable
func writeParser() {
	writeActions()
	writeTokens()
	writeErrMsgs()
	writeParseCode()
}
