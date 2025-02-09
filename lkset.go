package main

import (
	"strings"
)

/*
lkset implements Lookahead set for LR(1) parsing.
Each integer in the set represents 32 bits.
In the set, the order of integers are from left to right
but in each integer, the order of bits are from right to left.
For example, 1st bit is in the 1st integer, but it is the last bit of that integer!
*/
type lkset []int

/*
set sets a bit at position `bit` to 1
bit>>5 retrieves the integer for `bit`
bit&31 takes the last 5 bits of `bit`, essentially the same as `bit` if it is less than 5 bits
a |= (1<<b) set the bit bth from the END of a
*/
func (s lkset) set(bit int) {
	s[bit>>5] |= (1 << uint(bit&31))
}

/*
checks if bit `bit` is ON in the set
*/
func (s lkset) check(bit int) bool {
	return s[bit>>5]&(1<<uint(bit&31)) > 0
}

/*
union does the union of set s with r.
the size of union is size of s.
if set s changed, return true
*/
func (s lkset) union(r lkset) bool {
	changed := false
	for i := 0; i < lksize; i++ {
		tmp := s[i]
		s[i] |= r[i]
		if s[i] != tmp {
			changed = true
		}
	}
	return changed
}

/*
String prints the list of tokens in lkset
*/
func (s lkset) String() string {
	if s == nil {
		return "NULL"

	}
	buf := strings.Builder{}
	buf.WriteString("{ ")
	for i := 0; i <= termN; i++ {
		if s.check(i) {
			buf.WriteString(terms[i].name)
			buf.WriteString(" ")
		}
	}
	buf.WriteString("}")
	return buf.String()
}

func newLkset() lkset {
	return make([]int, lksize)
}

func emptyLkset() lkset {
	return []int{}
}

func (l lkset) clone() lkset {
	newSet := newLkset()
	copy(newSet, l)
	return newSet
}
