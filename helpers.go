package main

import (
	"math"
)

// set elements 0 through n-1 to c
func fill(v []int, n, c int) {
	for i := 0; i < n; i++ {
		v[i] = c
	}
}

func extend[T any](s *[]T, INC int) {
	new := make([]T, len(*s)+INC)
	copy(new, *s)
	*s = new
}

func isDigit(c rune) bool { return c >= '0' && c <= '9' }

func isWord(c rune) bool {
	return c >= 0xa0 || c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

var counts = map[int]int{} // used for counting occurrence of reductions or gotos

// reset counts before counting
func clearCounts() {
	for k := range counts {
		counts[k] = 0
	}
}

// i compacts info of associativity, precedence, type of a production or terminal symbol
func asc(i int) int { return i & 3 } // last 2 bits

func prec(i int) int { return (i >> 4) & 077 } // skip 4 bits, use 6 bits

func typ(i int) int { return (i >> 10) & 077 } // skip 10 bits, use 6 bits

func addAsc(i, j int) int { return i | j } // j <=3

func addPrec(i, j int) int { return i | (j << 4) } // j<= 077

func addTyp(i, j int) int { return i | (j << 10) } // j <= 077

// find min and max values of an array
func minMax(s []int) (int, int) {
	min, max := s[0], s[0]
	for _, n := range s {
		if n < min {
			min = n
		}
		if n > max {
			max = n
		}
	}
	return min, max
}

// types' name, min value and max value
var typValues = []struct {
	name string
	min  int
	max  int
}{
	{"int32", math.MinInt32, math.MaxInt32},
	{"int16", math.MinInt16, math.MaxInt16},
	{"int8", math.MinInt8, math.MaxInt8},
}

// the type with minimum size that can store all values of s. prefer unsigned types
func minType(s []int) string {
	typ := "int"
	min, max := minMax(s)
	for _, tv := range typValues {
		if min >= tv.min && max <= tv.max {
			typ = tv.name
		}
	}
	return typ
}
