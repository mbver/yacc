package main

import (
	"reflect"
	"testing"
)

func TestStateGen(t *testing.T) {
	termN = 6
	nontermN = 8 // include $accept
	prods = [][]int{
		{4096, 4097, 1, 0},
		{4097, 4098, -1},
		{4098, 4098, 4099, -2},
		{4098, -3},
		{4099, 4100, -4},
		{4100, 4101, -5},
		{4101, 5, 4, 4101, -6},
		{4101, 4102, -7},
		{4102, 4103, -8},
		{4103, 5, -9},
	}

	yields = [][][]int{
		{prods[0]},
		{prods[1]},
		{prods[2], prods[3]},
		{prods[4]},
		{prods[5]},
		{prods[6], prods[7]},
		{prods[8]},
		{prods[9]},
	}
	firstSets = []lkset{{32}, {32}, {32}, {32}, {32}, {32}, {32}}
	empty = []bool{false, true, true, false, false, false, false, false}
	wSet = make([]wItem, 57)
	lksize = 1
	clkset = newLkset()
	expKern := []item{
		{1, []int{4096, 4097, 1, 0}, 4097, lkset{0}},      // 0
		{2, []int{4096, 4097, 1, 0}, 1, lkset{0}},         // 1
		{2, []int{4097, 4098, -1}, -1, lkset{2}},          // 2
		{2, []int{4098, 4098, 4099, -2}, 4099, lkset{34}}, // 2
		{3, []int{4098, 4098, 4099, -2}, -2, lkset{34}},   // 3
		{2, []int{4099, 4100, -4}, -4, lkset{34}},         // 4
		{2, []int{4100, 4101, -5}, -5, lkset{34}},         // 5
		{2, []int{4101, 5, 4, 4101, -6}, 4, lkset{34}},    // 6
		{2, []int{4103, 5, -9}, -9, lkset{34}},            // 6
		{2, []int{4101, 4102, -7}, -7, lkset{34}},         // 7
		{2, []int{4102, 4103, -8}, -8, lkset{34}},         // 8
		{3, []int{4101, 5, 4, 4101, -6}, 4101, lkset{34}}, // 9
		{4, []int{4101, 5, 4, 4101, -6}, -6, lkset{34}},   // 10
		{2, []int{4102, 4103, -8}, -8, lkset{34}},         // ???????
		{2, []int{4103, 5, -9}, -9, lkset{34}},            // ????????
	}
	expGotos := []int{1, 2, 3, 4, 5, 7, 8, 10, 7, 8}
	expGotoIdx := []int{-1, -1, -1, -1, -1, -1, -1, -1, -1, 2, -1}
	expLastGoto := 9
	stategen()
	resKern := []item{}
	for _, a := range kernls {
		if a.prd != nil {
			resKern = append(resKern, a)
		}
	}
	if !reflect.DeepEqual(expKern, resKern) {
		t.Errorf("kernels not correct: expected: %+v, got: %+v\n", expKern, resKern)
	}
	resGotos := []int{}
	for _, a := range actionStore {
		if a == 0 {
			break
		}
		resGotos = append(resGotos, a)
	}
	if !reflect.DeepEqual(resGotos, expGotos) {
		t.Errorf("actions not correct: expected: %v, got: %v\n", expGotos, resGotos)
	}
	resGotoIdx := []int{}
	for _, g := range gotoIdx {
		if g == 0 {
			break
		}
		resGotoIdx = append(resGotoIdx, g)
	}
	if !reflect.DeepEqual(resGotoIdx, expGotoIdx) {
		t.Errorf("gotos not correct: expected: %v, got: %v\n", expGotos, resGotos)
	}
	if expLastGoto != lastActIdx {
		t.Errorf("last action not correct: expected: %v, got: %v", expLastGoto, lastActIdx)
	}
}
