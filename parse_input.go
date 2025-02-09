package main

import (
	"bufio"
	"bytes"
	"fmt"
	"unicode"
)

const prefix = "yy"    // makes the output code mysterious and the user's jaw drop!😲😲😲
const PRIVATE = 0xE000 // only $end symbol is private. serves to make the code more mysterious! ¯\_(ツ)_/¯
const (
	IDENTIFIER = PRIVATE + iota // terminal with values above PRIVATE is used in parsing grammar
	MARK
	TERM
	LEFT
	RIGHT
	BINARY
	PREC
	LCURLY
	IDENTCOLON
	NUMBER
	START
	TYPEDEF
	TYPENAME
	UNION
	ERROR
)
const SYMINC = 50
const ENDFILE = 0
const OK = 1
const NOMORE = -1000
const EOF = -1
const PRODINC = 100
const RULEINC = 50

const (
	ACTFLAG = 1 << (iota + 2) // rule has imbedded action
	REDFLAG                   // rule is reduced (in a state)
)

var tokname string // store the current identifier name, word, typename
var lineno = 1     // current input line number

var peekline = 0                   // line read but not yet consumed
var peekrune rune                  // temporarily store the rune that was read, but not consumed
var ntypes int                     // number of types defined
var typeset = make(map[int]string) // pointers to type tags
var numbval int                    // temporarily store value of an input number
var extval int                     // value of terminal symbol, except for single-character literals
var start int                      // start symbol value
var ty int                         // the numeric value of type
var fmtImported bool               // helps to import fmt just once
var fcode = &bytes.Buffer{}        // store imbedded actions. write to ftable later
var rlines []int                   // line number of a rule
var nprod = 1                      // number of productions
var initialstacksize = 16          // initial stack size
const TOKSTART = 4                 // the first terminal symbol defined grammar
var termN = 0                      // last idx of terminal symbol in terms. starting from 1 because 0 is to mark the end of production 0
var nontermN = -1                  // last idx of non-terminal symbol in non-terms including $accept. starting from 0

type Error struct {
	lineno int
	tokens []string
	msg    string
}

var errors []Error

type Resrv struct {
	name  string
	value int
}

var resrv = []Resrv{
	{"binary", BINARY},
	{"left", LEFT},
	{"nonassoc", BINARY},
	{"prec", PREC},
	{"right", RIGHT},
	{"start", START},
	{"term", TERM},  // terminal symbol
	{"token", TERM}, // terminal symbol. same as term
	{"type", TYPEDEF},
	{"union", UNION},  // union type
	{"struct", UNION}, // union type. same as union
	{"error", ERROR},
}

var nerrors = 0 // number of errors
var stderr *bufio.Writer
var fatfl = 1 // if on, error is fatal

// write out error with line
func lineErrorf(lineno int, s string, v ...interface{}) {
	s += "\n"
	nerrors++
	fmt.Fprintf(stderr, s, v...)
	fmt.Fprintf(stderr, ": %v:%v\n", infile, lineno)
	if fatfl != 0 {
		summary()
		exit(1)
	}
}

// write error. exit when fatfl is on
func errorf(s string, v ...interface{}) {
	lineErrorf(lineno, s, v...)
}

// read the next rune in input stream
func getRune(f *bufio.Reader) rune {
	var r rune

	if peekrune != 0 {
		if peekrune == EOF {
			return EOF
		}
		r = peekrune
		peekrune = 0
		return r
	}

	c, n, err := f.ReadRune()
	if n == 0 {
		return EOF
	}
	if err != nil {
		errorf("read error: %v", err)
	}
	return c
}

// put the rune back to be consumed later
func ungetRune(f *bufio.Reader, c rune) {
	if f != finput { // seems unnecessary?
		panic("ungetc - not finput")
	}
	if peekrune != 0 {
		panic("ungetc - 2nd unget")
	}
	peekrune = c
}

// skip over comments, is called after reading a '/'
func skipComment() int {
	c := getRune(finput)
	if c == '/' { // line comment, skip the whole line
		for c != EOF {
			if c == '\n' {
				return 1
			}
			c = getRune(finput)
		}
		errorf("EOF inside line comment")
		return 0
	}
	// must be block comment
	if c != '*' {
		errorf("illegal comment")
	}

	nl := 0             // lines skipped
	c = getRune(finput) // must be *

	for c != EOF {
		switch c {
		case '*':
			c = getRune(finput)
			if c == '/' { // end of comment
				return nl
			}
		case '\n':
			nl++
			fallthrough
		default:
			c = getRune(finput)
		}
	}
	errorf("EOF inside block comment")
	return 0
}

// read the word and store it into tokname
func getWord(c rune) {
	tokname = ""
	for isWord(c) || isDigit(c) || c == '.' || c == '$' {
		tokname += string(c)
		c = getRune(finput)
	}
	ungetRune(finput, c) // put last character back
}

// consume token, its name is stored in tokname, numeric value is in numbval
// returns the token types: ENDFILE, TYPENAME, IDENTIFIER, NUMBER, IDENTCOLON
// MARK, LCURLY, PREC, "=" (for "{"" of action),
func getToken() int {
	var i int
	var match, c rune

	tokname = ""
	// skip white spaces and comments
	for {
		lineno += peekline
		peekline = 0
		c = getRune(finput)
		// ship white spaces
		for c == ' ' || c == '\n' || c == '\t' || c == '\v' || c == '\r' {
			if c == '\n' {
				lineno++
			}
			c = getRune(finput)
		}
		// not-comment, break
		if c != '/' {
			break
		}
		// skip comment
		lineno += skipComment()
	}

	switch c {
	case EOF:
		return ENDFILE

	case '{': // start of imbedded action, not LCURLY %{
		ungetRune(finput, c)
		return '='

	case '<': // type definition <type_name>
		c = getRune(finput)
		for c != '>' && c != EOF && c != '\n' {
			tokname += string(c) // store the name of type
			c = getRune(finput)
		}

		if c != '>' {
			errorf("unterminated < ... > clause")
		}
		// search existing types
		for i = 1; i <= ntypes; i++ {
			if typeset[i] == tokname {
				numbval = i // store the numeric value of type
				return TYPENAME
			}
		}
		// not found, new type
		ntypes++
		numbval = ntypes // store numeric value of new type
		typeset[numbval] = tokname
		return TYPENAME

	case '"', '\'': // literal tokens: 'abc', "abc"
		match = c
		tokname = string(c)
		for {
			c = getRune(finput)
			if c == '\n' || c == EOF {
				errorf("illegal or missing ' or \"")
			}
			if c == match {
				tokname += string(c)
				return IDENTIFIER
			}
			tokname += string(c)
		}

	case '%': // MARK, PREC, LCURLY, UNION, TYPE, TOKEN,
		c = getRune(finput)
		switch c {
		case '%': // MARK token, %%
			return MARK
		case '=': // PREC %=, same as %prec
			return PREC
		case '{': // block of go code %{, don't use
			return LCURLY
		}

		getWord(c) // reserved word: %union, %token, %type ... and %error!
		// find a reserved word
		for i := range resrv {
			if tokname == resrv[i].name {
				return resrv[i].value
			}
		}
		errorf("invalid escape, or illegal reserved word: %v", tokname)

	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9': // number
		numbval = int(c - '0') // numeric value for the token
		for {
			c = getRune(finput)
			if !isDigit(c) {
				break
			}
			numbval = numbval*10 + int(c-'0')
		}
		ungetRune(finput, c) // put the last rune back

		return NUMBER

	default: // abc or .abc or $abc
		if isWord(c) || c == '.' || c == '$' {
			getWord(c)
			break
		}
		// ============= why return this???
		return int(c)
	}

	// IDENTCOLON: abc; IDENTIFIER: abc
	c = getRune(finput)
	// skip white spaces and comments
	for c == ' ' || c == '\t' || c == '\n' || c == '\v' || c == '\r' || c == '/' {
		if c == '\n' {
			peekline++
		}
		// look for comments
		if c == '/' {
			peekline += skipComment()
		}
		c = getRune(finput)
	}
	if c == ':' {
		return IDENTCOLON
	}

	ungetRune(finput, c) // put last rune back
	return IDENTIFIER
}

// parse .y input file
func parseInput() {
	t := parseDeclaration()
	if t == ENDFILE {
		errorf("unexpected EOF before %%")
	}
	t = parseRules()
	writeSymbols()
	if t == MARK {
		copyProgram()
	}
}

// parse declaration part of input file, including code section
func parseDeclaration() int {
	var j int
	var i int
	t := getToken()

	// read code (%{), %union, %token, %token <type> ... until %% (productions) or EOF
	for {
		switch t {
		default:
			errorf("syntax error tok=%v", t-PRIVATE)

		case MARK, ENDFILE:
			return t

		case ERROR: // %error
			lno := lineno
			var tokens []string
			for {
				t := getToken()
				if t == ':' {
					break
				}
				if t != IDENTIFIER && t != IDENTCOLON {
					errorf("bad syntax in %%error")
				}
				tokens = append(tokens, tokname)
				if t == IDENTCOLON {
					break
				}
			}
			if getToken() != IDENTIFIER {
				errorf("bad syntax in %%error")
			}
			errors = append(errors, Error{lno, tokens, tokname})

		case TYPEDEF: // TYPEDEF TYPENAME (ty store numeric value of type)
			t = getToken()
			if t != TYPENAME {
				errorf("bad syntax in %%type")
			}
			ty = numbval
			for {
				t = getToken()
				switch t {
				case IDENTIFIER:
					if t = findTerminal(tokname); t > 0 {
						j = typ(aptTerm[t])
						if j == 0 { // terminal symbol defined but type is not set. ok
							aptTerm[t] = addTyp(aptTerm[t], ty)
						} else if j != ty { // terminal symbol type is set but different from current type. error
							errorf("type redeclaration of token %s", terms[t].name)
						}
					} else if t = findNonterm(tokname); t > 0 {
						j = nonterms[t-NTBASE].typ
						if j == 0 { // non-terminal symbol found but type is not set. set it
							nonterms[t-NTBASE].typ = ty
						} else if j != ty { // non-term type is set and different from current type. error
							errorf("type redeclaration of nonterminal %v", nonterms[t-NTBASE].name)
						}
					} else { // not defined. define nonterm
						t = defineNonTerminal(tokname)
						nonterms[t-NTBASE].typ = ty
					}
					continue
				case ',':
					continue
				}
				break
			}
			continue // avoid getToken after switch!

		// %token ASSIGN OR %token <str> ID
		case LEFT, BINARY, RIGHT, TERM:
			as := t - TERM // associativity. LEFT:1, RIGHT: 2, BINARY: 3
			if as != 0 {
				i++ // precedence
			}
			ty = 0
			t = getToken()

			// type name: %token <str> ID
			if t == TYPENAME {
				ty = numbval // ty is the idx of type in typeset
				t = getToken()
			}
			for {
				switch t {
				case ',':
					t = getToken()
					continue

				case ';':
					// Do nothing.

				case IDENTIFIER:
					if findNonterm(tokname) > 0 {
						errorf("%v defined earlier as nonterminal", tokname)
					}
					j = findInsertTerminal(tokname)
					if as != 0 {
						if asc(aptTerm[j]) != 0 {
							errorf("redeclaration of precedence of %v", tokname)
						}
						aptTerm[j] = addAsc(aptTerm[j], as)
						aptTerm[j] = addPrec(aptTerm[j], i)
					}
					if ty != 0 {
						if typ(aptTerm[j]) != 0 {
							errorf("redeclaration of type of %v", tokname)
						}
						aptTerm[j] = addTyp(aptTerm[j], ty)
					}
					t = getToken()
					if t == NUMBER { // terminal j has numeric value
						terms[j].value = numbval
						t = getToken()
					}
					continue
				}
				break
			}
			continue // avoid getToken after switch
		case ';':
			// Do nothing.

		case START:
			t = getToken()
			if t != IDENTIFIER {
				errorf("bad %%start construction")
			}
			start = findInsertNonTerminal(tokname)
		case UNION:
			copyUnion() // getToken after switch
		case LCURLY:
			copyCode() // getToken after switch
		}
		t = getToken()
	}
}

// parse rules part of input file
func parseRules() int {
	fmt.Fprintf(fcode, "switch %snt {\n", prefix)
	extendProds()
	// production 0 [$accept program $end 0]
	prods[0] = []int{NTBASE, start, 1, 0}
	t := getToken()
	if t != IDENTCOLON {
		errorf("LHS must be IDENTCOLON")
	}
	// start symbol is not defined
	// use LHS of rule 1, no matter it is terminal or nonterminal!
	if start == 0 {
		if n := findTerminal(tokname); n > 0 {
			prods[0][1] = n
		} else {
			prods[0][1] = findInsertNonTerminal(tokname)
		}
	}
	nprod = 1
	prd := make([]int, RULEINC) // temporary array to store rule
	for t != MARK && t != ENDFILE {
		rlines[nprod] = lineno
		ruleline := lineno
		i := 0 // idx of symbol in rule

		// LHS of rule. must be an IDENTCOLON or |
		if t == '|' {
			prd[0] = prods[nprod-1][0]
			i++
		} else if t == IDENTCOLON {
			if findTerminal(tokname) > 0 { // do not accept terminal symbol as LHS
				lineErrorf(ruleline, "token illegal on LHS of grammar rule")
			}
			prd[i] = findInsertNonTerminal(tokname)
			i++
		} else {
			lineErrorf(ruleline, "illegal rule: missing <identifier>: or | ?")
		}

		// RHS of rule
		t = getToken()
		for {
			for t == IDENTIFIER { // get all idenfier until action {
				if n := findTerminal(tokname); n > 0 {
					prd[i] = n
					aptPrd[nprod] = aptTerm[n]
				} else {
					prd[i] = findInsertNonTerminal(tokname)
				}
				i++
				if i >= len(prd) {
					extend(&prd, RULEINC)
				}
				t = getToken()
			}
			if t == PREC {
				if getToken() != IDENTIFIER {
					lineErrorf(ruleline, "illegal %%prec syntax")
				}
				j := findTerminal(tokname)
				if j < 0 {
					lineErrorf(ruleline, "terminal required after %%prec")
				}
				aptPrd[nprod] = aptTerm[j]
				t = getToken()
			}
			// expect to have a '{'. it was put back to be processed later so '=' was returned
			if t != '=' { // not having action. stop
				break
			}
			// have action
			aptPrd[nprod] |= ACTFLAG // prd has action
			fmt.Fprintf(fcode, "\n\tcase %v:", nprod)
			// yyDollar = yyS[yypt-1 : yypt+1]
			fmt.Fprintf(fcode, "\n\t\t%sDollar = %sS[%spt-%v:%spt+1]", prefix, prefix, prefix, i-1, prefix)
			// write action to fcode
			copyAction(prd, i)
			t = getToken()
			if t == IDENTIFIER { // action in the middle of rule
				// define epsilon rule for the action
				j := defineNonTerminal(fmt.Sprintf("$$%d", nprod))
				prods[nprod] = []int{j, -nprod}
				// the current production move 1 notch forward
				nprod++
				extendProds()
				aptPrd[nprod] = aptPrd[nprod-1] & ^ACTFLAG // unset action flag in original production
				aptPrd[nprod-1] = ACTFLAG
				rlines[nprod] = lineno // assign rule line to original rule
				// add the new rule to original rule
				prd[i] = j
				i++
				if i >= len(prd) {
					extend(&prd, RULEINC)
				}
			}
		}

		// end of rule
		for t == ';' { // skip all the semi
			t = getToken()
		}
		prd[i] = -nprod // set the end of production
		i++

		// if no imbedded action specified, check if default action feasible
		// required first symbol of RHS has the same type with LHS
		if aptPrd[nprod]&ACTFLAG == 0 {
			if lType := nonterms[prd[0]-NTBASE].typ; lType != 0 {
				j := prd[1]
				var rType int
				if j < 0 {
					lineErrorf(ruleline, "must return a value, since LHS has a type")
				}
				if j >= NTBASE {
					rType = nonterms[j-NTBASE].typ
				} else {
					rType = typ(aptTerm[j])
				}
				if lType != rType {
					lineErrorf(ruleline, "default action causes potential type clash")
				}
			}
		}
		extendProds()
		prods[nprod] = make([]int, i) // skip all trailing 0s in prd
		copy(prods[nprod], prd)
		// preparing for new rule
		nprod++
		extendProds()
		aptPrd[nprod] = 0
	}
	if TRANSIZE < termN+nontermN+1 {
		errorf("too many tokens (%d) or non-terminals (%d)", termN, nontermN+1)
	}
	fmt.Fprintf(fcode, "\n\t}") // end of imbedded actions code
	return t
}

// write out terminal symbols to ftable
func writeSymbols() {
	// write constant
	for _, t := range terms[TOKSTART : termN+1] {
		if t.isConst {
			fmt.Fprintf(ftable, "const %s = %d\n", t.name, t.value)
		}
	}

	// write token names
	ftable.WriteRune('\n')
	fmt.Fprintf(ftable, "var %sToknames = [...]string{\n", prefix)
	for _, t := range terms[1 : termN+1] {
		fmt.Fprintf(ftable, "\t%q,\n", t.name)
	}
	fmt.Fprintf(ftable, "}\n")
	ftable.WriteRune('\n')

	ftable.WriteRune('\n')
	fmt.Fprintf(ftable, "const %sEofCode = 1\n", prefix)
	fmt.Fprintf(ftable, "const %sErrCode = 2\n", prefix)
	fmt.Fprintf(ftable, "const %sInitialStackSize = %v\n", prefix, initialstacksize)
	fmt.Fprintf(ftable, "const %sPrivate = %v\n", prefix, PRIVATE)
}

// copy program part of input file
func copyProgram() {
	fmt.Fprintf(ftable, "\n//line %v:%v\n", infile, lineno)
	for {
		c := getRune(finput)
		if c == EOF {
			break
		}
		ftable.WriteRune(c)
	}
}

// copy the union declaration to the output, and the define file if present
func copyUnion() {

	fmt.Fprintf(ftable, "\n//line %v:%v\n", infile, lineno)
	fmt.Fprintf(ftable, "type %sSymType struct", prefix)

	level := 0

out:
	for {
		c := getRune(finput)
		if c == EOF {
			errorf("EOF encountered while processing %%union")
		}
		ftable.WriteRune(c)
		switch c {
		case '\n':
			lineno++
		case '{':
			if level == 0 {
				fmt.Fprintf(ftable, "\n\tyys int")
			}
			level++
		case '}':
			level--
			if level == 0 {
				break out
			}
		}
	}
	fmt.Fprintf(ftable, "\n\n")
}

// saves code between %{ and %}
// adds an import for __fmt__
func copyCode() {
	lno := lineno

	c := getRune(finput)
	if c == '\n' { // %{\n
		c = getRune(finput)
		lineno++
	}
	fmt.Fprintf(ftable, "\n//line %v:%v\n", infile, lineno)
	// accumulate until %}
	code := make([]rune, 0, 1024) // capacity will be 1024, 2048, 4096
	for c != EOF {
		if c == '%' {
			c = getRune(finput)
			if c == '}' { // %} end of code section
				writeCode(code, lno+1) // lno + 1 is the line right after %{
				return
			}
			code = append(code, '%')
		}
		code = append(code, c)
		if c == '\n' {
			lineno++
		}
		c = getRune(finput)
	}
	lineno = lno // error, restore lineno
	errorf("eof before %%}")
}

// writes code between %{ and %} to ftable
// called by copyCode
// adds an import for __yyfmt__ after the package clause
func writeCode(code []rune, lineno int) {
	for i, line := range lines(code) {
		writeLine(line)
		if !fmtImported && isPackageClause(line) {
			fmtImported = true
			fmt.Fprintln(ftable, `import __yyfmt__ "fmt"`)
			fmt.Fprintf(ftable, "//line %v:%v\n\t\t", infile, lineno+i)
		}
	}
}

// break code into lines
func lines(code []rune) [][]rune {
	l := make([][]rune, 0, 100)
	for len(code) > 0 {
		// one line per loop
		var i int
		for i = range code { // i gets value from range
			if code[i] == '\n' {
				break
			}
		}
		l = append(l, code[:i+1]) // including '\n' in line
		code = code[i+1:]
	}
	return l
}

// writes line of code to ftable
func writeLine(line []rune) {
	for _, r := range line {
		ftable.WriteRune(r)
	}
}

// check format: package abc //comment
func isPackageClause(line []rune) bool {
	line = skipSpace(line)

	// must be big enough.
	if len(line) < len("package X\n") {
		return false
	}

	// must start with "package"
	for i, r := range []rune("package") {
		if line[i] != r {
			return false
		}
	}
	line = skipSpace(line[len("package"):])

	// must have another identifier.
	if len(line) == 0 || (!unicode.IsLetter(line[0]) && line[0] != '_') {
		return false
	}
	for len(line) > 0 {
		if !unicode.IsLetter(line[0]) && !unicode.IsDigit(line[0]) && line[0] != '_' {
			break
		}
		line = line[1:]
	}
	line = skipSpace(line)

	// eol, newline, or comment must follow
	if len(line) == 0 {
		return true
	}
	if line[0] == '\r' || line[0] == '\n' {
		return true
	}
	if len(line) >= 2 {
		return line[0] == '/' && (line[1] == '/' || line[1] == '*')
	}
	return false
}

// skip leading spaces
func skipSpace(line []rune) []rune {
	for len(line) > 0 {
		if line[0] != ' ' && line[0] != '\t' {
			break
		}
		line = line[1:]
	}
	return line
}

// extend allPrds, aptPrd, rlines if they are full
func extendProds() {
	if nprod >= len(prods) {
		extend(&prods, PRODINC)
		extend(&aptPrd, PRODINC)
		extend(&rlines, PRODINC)
	}
}

// copy imbedded action until ";" or "}"
func copyAction(prod []int, max int) {

	fmt.Fprintf(fcode, "\n//line %v:%v", infile, lineno)
	fmt.Fprint(fcode, "\n\t\t")

	lno := lineno
	brac := 0

loop:
	for {
		c := getRune(finput)

	swt:
		switch c {
		case ';':
			if brac == 0 {
				fcode.WriteRune(c)
				return
			}

		case '{':
			brac++

		case '$': // most logic happens here!
			s := 1
			tok := -1
			c = getRune(finput)

			// type description
			if c == '<' { // $<int>$1 ???????
				ungetRune(finput, c)
				if getToken() != TYPENAME {
					errorf("bad syntax on $<ident> clause")
				}
				tok = numbval // type number
				c = getRune(finput)
			}
			if c == '$' {
				fmt.Fprintf(fcode, "%sVAL", prefix)

				// write type, i.e. yyVAL.tnode
				if ntypes != 0 {
					if tok < 0 {
						tok = findSymType(prod[0])
					}
					fmt.Fprintf(fcode, ".%v", typeset[tok])
				}
				continue loop
			}
			if c == '-' {
				s = -s
				c = getRune(finput)
			}
			j := 0
			// $1 => yyDollar[1].tnode; $statement_list@1 => yyDollar[2].tnode
			if isDigit(c) {
				for isDigit(c) {
					j = j*10 + int(c-'0')
					c = getRune(finput)
				}
				ungetRune(finput, c)
				j = j * s
				if j >= max {
					errorf("Illegal use of $%v", j)
				}
			} else if isWord(c) || c == '.' { // $statement_list@1 - rare syntax: first occurence of statement_List
				// look for $name
				ungetRune(finput, c)
				if getToken() != IDENTIFIER {
					errorf("$ must be followed by an identifier")
				}

				tokn := findSym(tokname)
				fnd := -1
				c = getRune(finput)
				if c != '@' {
					ungetRune(finput, c)
				} else if getToken() != NUMBER {
					errorf("@ must be followed by number")
				} else {
					fnd = numbval // $statement_list@1
				}
				for j = 1; j < max; j++ {
					if tokn == prod[j] { // search RHS for tokn @fnd times
						fnd--
						if fnd <= 0 {
							break
						}
					}
				}
				if j >= max {
					errorf("$name or $name@number not found")
				}
			} else { // ===== not a digit? not a word??? should be error!!!!!
				fcode.WriteRune('$')
				if s < 0 {
					fcode.WriteRune('-')
				}
				ungetRune(finput, c)
				continue loop
			}
			fmt.Fprintf(fcode, "%sDollar[%v]", prefix, j)

			// put out the proper tag
			if ntypes != 0 {
				if j <= 0 && tok < 0 { // this can occur normally. we better not supporting negative number ?
					errorf("must specify type of $%v", j)
				}
				if tok < 0 {
					tok = findSymType(prod[j])
				}
				fmt.Fprintf(fcode, ".%v", typeset[tok])
			}
			continue loop

		case '}':
			brac--
			if brac != 0 { // write c and continue loop
				break
			}
			fcode.WriteRune(c)
			return

		case '/': // comment in action. copy
		case '\'', '"': // "abc" or 'abc'
			// character string or constant
			match := c
			fcode.WriteRune(c)
			c = getRune(finput)
			for c != EOF {
				if c == '\\' {
					fcode.WriteRune(c)
					c = getRune(finput)
					if c == '\n' {
						lineno++
					}
				} else if c == match {
					break swt
				}
				if c == '\n' {
					errorf("newline in string")
				}
				fcode.WriteRune(c)
				c = getRune(finput)
			}
			errorf("EOF in string or character constant")

		case EOF:
			lineno = lno
			errorf("action does not terminate")

		case '\n':
			fmt.Fprint(fcode, "\n\t")
			lineno++
			continue loop
		}

		fcode.WriteRune(c)
	}
}

// find type of a symbol
func findSymType(t int) int {
	var ttype int
	var tname string
	if t >= NTBASE {
		ttype = nonterms[t-NTBASE].typ
		tname = nonterms[t-NTBASE].name
	} else {
		ttype = typ(aptTerm[t])
		tname = terms[t].name
	}
	if ttype <= 0 {
		errorf("must specify type for %s", tname)
	}
	return ttype
}
