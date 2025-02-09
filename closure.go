package main

var zzclose = 0 // count closures

/* closure computes LALR(1) closure for state n. used in stategen*/
func closure1(n int) {
	zzclose++
	cwp = 0 // reset current pointer for working set
	// copy kernel items of new state
	for p := kernlp[n]; p < kernlp[n+1]; p++ {
		wSet[cwp].item = kernls[p].clone()
		wSet[cwp].closed, wSet[cwp].done = false, false
		cwp++
	}
	/* changed reflects that
	- an item is added to closure
	- lookahead set of an item is changed (by merged)
	these can:
		- add more items into closure
		- change lookahead sets of items in closure
	*/
	changed := true
	for changed {
		changed = false
		for i := 0; i < cwp; i++ {
			fill(clkset, lksize, 0)
			if wSet[i].closed {
				continue
			}
			wSet[i].closed = true // mark processed, skip next time

			first := wSet[i].item.first
			if first < NTBASE { // terminal symbol or action, skip
				continue
			}
			// non-terminal symbol, get lookahead set for items derived from this item
			itemI := wSet[i].item
			prd := itemI.prd
			j := itemI.off + 1
			sym := prd[j] // symbol right after first
			for sym > 0 {
				if sym < NTBASE {
					clkset.set(sym) // terminal symbol, set it and stop
					break
				}
				// non-terminal symbol, merge its first set
				clkset.union(firstSets[sym-NTBASE])
				if !empty[sym-NTBASE] { // not nullable, stop
					break
				}
				j++
				sym = prd[j]
			}
			if sym <= 0 { // reach the end
				clkset.union(itemI.lkset)
			}
			// the productions that has first as LHS
			prds := yields[first-NTBASE]
		nextPrd:
			for _, prd = range prds {
				/* if an item is already derived with prdI,
				for example, an item derived itself! t -> .t*f
				merge its lookahead set with clkset*/
				for i := 0; i < cwp; i++ {
					itemI := wSet[i].item
					// derived item always has oft = 1
					if itemI.off == 1 && id(itemI.prd) == id(prd) {
						if itemI.lkset.union(clkset) {
							changed = true
							wSet[i].closed = false // mark to process later. may change closure of derived items
						}
						continue nextPrd

					}
				}

				if cwp >= len(wSet) {
					extend(&wSet, WSETINC)
				}
				wSet[cwp].item = item{1, prd, prd[1], clkset.clone()}
				wSet[cwp].closed, wSet[cwp].done = false, false
				changed = true
				cwp++
			}
		}
	}
}

/* closure0 computes LR0 closure for state n. used in writing output*/
func closure0(n int) {
	zzclose++
	cwp = 0 // reset current pointer for working set
	// copy kernel items of new state
	for p := kernlp[n]; p < kernlp[n+1]; p++ {
		wSet[cwp].item = kernls[p].clone()
		wSet[cwp].closed, wSet[cwp].done = false, false
		cwp++
	}
	/* changed reflects that
	- an item is added to closure
	*/
	changed := true
	for changed {
		changed = false
		for i := 0; i < cwp; i++ {
			if wSet[i].closed {
				continue
			}
			wSet[i].closed = true // mark processed, skip next time

			first := wSet[i].item.first
			if first < NTBASE { // terminal symbol, skip
				continue
			}

			prds := yields[first-NTBASE]

		nextPrd:
			for _, prd := range prds {
				for i := 0; i < cwp; i++ { // check if an item already derived by this prd. avoid infinite loop!
					itemI := wSet[i].item
					// derived item always has oft = 1
					if itemI.off == 1 && id(itemI.prd) == id(prd) {
						continue nextPrd

					}
				}

				if cwp >= len(wSet) {
					extend(&wSet, WSETINC)
				}
				wSet[cwp].item = item{1, prd, prd[1], emptyLkset()}
				wSet[cwp].closed, wSet[cwp].done = false, false
				changed = true
				cwp++
			}
		}
	}
}
