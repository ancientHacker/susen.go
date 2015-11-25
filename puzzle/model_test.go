package puzzle

/*

Tests for the puzzle representation.

*/

// [TODO] Add history testing!

import (
	"fmt"
	"reflect"
	"testing"
)

/*

helpers

*/

func helperDupSquare(sq *square) *square {
	return &square{
		sq.index,
		sq.aval,
		newIntsetCopy(sq.pvals),
		sq.bval,
		append([]GroupID(nil), sq.bsrc...),
		sq.logger,
	}
}

// depends on newEmptySquare and (*square).subtract, test those first
func helperRestrictedSquare(index, sidelen int, excepts ...int) *square {
	sp := newEmptySquare(index, sidelen, nil)
	errs := sp.subtract(excepts)
	if len(errs) > 0 {
		panic(errs[0])
	}
	return sp
}

// depends on (*square).bind, test that first
func helperBindSquare(s *square, bval int, bsrc GroupID) *square {
	errs := s.bind(bval, bsrc)
	if len(errs) > 0 {
		panic(errs[0])
	}
	return s
}

// map from group index to group ID
func helperGID(gi int) GroupID {
	// if the group index is eligible for a 4x4 puzzle, assume
	// it's from a 4x4 puzzle, otherwise it's a fake
	if gi > 0 && gi < len(square4Map.gdescs) {
		return square4Map.gdescs[gi].id
	}
	return GroupID{"test", gi}
}

// map from group indices to group ids
func helperBsrc(gis ...int) []GroupID {
	gids := make([]GroupID, len(gis))
	for i, gi := range gis {
		gids[i] = helperGID(gi)
	}
	return gids
}

func helperSquareGroupDescriptor(slen int, gtype string, idx int) *groupDescriptor {
	mapping, err := squarePuzzleMapping(slen * slen)
	if err != nil {
		panic(err)
	}
	for _, g := range mapping.gdescs {
		if g.id.Gtype == gtype && g.id.Index == idx {
			return &g
		}
	}
	panic(fmt.Errorf("No such group: \"%s %d\"", gtype, idx))
}

func TestHelperSquareGroupDescriptor(t *testing.T) {
	pgd := helperSquareGroupDescriptor(4, GtypeRow, 1)
	egd := groupDescriptor{1, GroupID{GtypeRow, 1}, []int{1, 2, 3, 4}}
	if !reflect.DeepEqual(*pgd, egd) {
		t.Errorf("Row 1 descriptor in 4-puzzle: got %v, expected %v", *pgd, egd)
	}

	pgd = helperSquareGroupDescriptor(9, GtypeTile, 8)
	egd = groupDescriptor{26, GroupID{GtypeTile, 8}, []int{58, 59, 60, 67, 68, 69, 76, 77, 78}}
	if !reflect.DeepEqual(*pgd, egd) {
		t.Errorf("Tile 8 descriptor in 9-puzzle: got %v, expected %v", *pgd, egd)
	}
}

// vals > 0 are assignments, vals = 0 are empty, vals < 0 are additional removals
func helperMakeGroupSquares(gd *groupDescriptor, vals ...int) []*square {
	sidelen := len(gd.indices)
	if sidelen != len(vals) {
		panic("Mismatch between group size and number of values.")
	}
	maxidx := gd.indices[len(gd.indices)-1]
	squares := make([]*square, maxidx+1) // 1-based
	// first make the assigned squares, tracking the values
	var have []int
	for i, idx := range gd.indices {
		if val := vals[i]; val > 0 {
			if val < 1 || val > sidelen {
				panic(fmt.Errorf("Bad value in helperMakeGroupSquares: %d", val))
			}
			squares[idx] = newFilledSquare(idx, sidelen, val, nil)
			have = append(have, val)
		}
	}
	// now make the unassigned squares, removing the assigned possible values
	for i, idx := range gd.indices {
		if val := vals[i]; val <= 0 {
			squares[idx] = helperRestrictedSquare(idx, sidelen, have...)
			if val < 0 {
				errs := squares[idx].remove(-val)
				if len(errs) > 0 {
					panic(errs[0])
				}
			}
		}
	}
	return squares
}

type helperMakeGroupSquaresTestcase struct {
	sidelen int
	gtype   string
	gindex  int
	vals    []int
	es      []*square
}

func TestHelperMakeGroupSquares(t *testing.T) {
	tcs := []helperMakeGroupSquaresTestcase{
		helperMakeGroupSquaresTestcase{
			4, GtypeRow, 1,
			[]int{1, 2, 0, 0},
			[]*square{
				nil,
				newFilledSquare(1, 4, 1, nil),
				newFilledSquare(2, 4, 2, nil),
				helperRestrictedSquare(3, 4, 1, 2),
				helperRestrictedSquare(4, 4, 1, 2),
			},
		},
		helperMakeGroupSquaresTestcase{
			4, GtypeRow, 1,
			[]int{1, -4, -2, -3},
			[]*square{
				nil,
				newFilledSquare(1, 4, 1, nil),
				helperRestrictedSquare(2, 4, 1, 4),
				helperRestrictedSquare(3, 4, 1, 2),
				helperRestrictedSquare(4, 4, 1, 3),
			},
		},
		helperMakeGroupSquaresTestcase{
			4, GtypeRow, 1,
			[]int{1, -4, 3, -2},
			[]*square{
				nil,
				newFilledSquare(1, 4, 1, nil),
				helperRestrictedSquare(2, 4, 1, 3, 4),
				newFilledSquare(3, 4, 3, nil),
				helperRestrictedSquare(4, 4, 1, 2, 3),
			},
		},
		helperMakeGroupSquaresTestcase{
			4, GtypeTile, 1,
			[]int{1, -4, 3, -2},
			[]*square{
				nil,
				newFilledSquare(1, 4, 1, nil),
				helperRestrictedSquare(2, 4, 1, 3, 4),
				nil,
				nil,
				newFilledSquare(5, 4, 3, nil),
				helperRestrictedSquare(6, 4, 1, 2, 3),
			},
		},
	}
	for _, tc := range tcs {
		gd := helperSquareGroupDescriptor(tc.sidelen, tc.gtype, tc.gindex)
		ss := helperMakeGroupSquares(gd, tc.vals...)
		es := tc.es
		if len(ss) != len(es) {
			t.Fatalf("In group %v, square shape %v doesn't match expected %v.", gd, ss, es)
		}
		if !reflect.DeepEqual(ss, es) {
			t.Errorf("In group %v, got unexpected square(s):", gd)
			for _, i := range gd.indices {
				if s, e := ss[i], es[i]; !reflect.DeepEqual(s, e) {
					t.Errorf("%v (expected %v)", *s, *e)
				}
			}
		}
	}
}

// compare one square with another, excluding loggers
func helperSquareEqual(s1, s2 *square) bool {
	return (s1 == nil && s2 == nil) ||
		(s1.index == s2.index &&
			s1.aval == s2.aval &&
			reflect.DeepEqual(s1.pvals, s2.pvals) &&
			s1.bval == s2.bval &&
			reflect.DeepEqual(s1.bsrc, s2.bsrc))
}

// compare two slices of squares, excluding loggers
func helperSquaresEqual(ss1, ss2 []*square) bool {
	if len(ss1) != len(ss2) {
		return false
	}
	for i := range ss1 {
		if !helperSquareEqual(ss1[i], ss2[i]) {
			return false
		}
	}
	return true
}

// check for an error condition in a list of errors
func helperCheckCondition(cond ErrorCondition, errs []Error) bool {
	for _, err := range errs {
		if err.Condition == cond {
			return true
		}
	}
	return false
}

// diff two slices of squares, excluding loggers, returning (in
// increasing order) the indices of the ones that differ.  Only
// compares them as far as the shorter one goes, but meant to be
// applied to slices of the same length.
func helperDiffSquares(ss1, ss2 []*square) intset {
	max := len(ss1)
	if len(ss2) < len(ss1) {
		max = len(ss2)
	}
	var diff []int
	for i := 0; i < max; i++ {
		if !helperSquareEqual(ss1[i], ss2[i]) {
			diff = append(diff, ss1[i].index)
		}
	}
	return intset(diff)
}

// create a new sudoku *puzzle by casting from the Puzzle
func helperNewSudokuPuzzle(vals []int) (*puzzle, error) {
	p, e := newSudokuPuzzle(vals)
	if e != nil {
		return nil, e
	}
	return p.(*puzzle), nil
}

// create a new empty sudoku puzzle with the given sidelength
func helperNewEmptySudokuPuzzle(sidelen int) (*puzzle, error) {
	vals := make([]int, sidelen*sidelen)
	p, e := helperNewSudokuPuzzle(vals)
	return p, e
}

// create a new dudoku *puzzle by casting from the Puzzle
func helperNewDudokuPuzzle(vals []int) (*puzzle, error) {
	p, e := newDudokuPuzzle(vals)
	if e != nil {
		return nil, e
	}
	return p.(*puzzle), nil
}

/*

test values

*/

var (
	square4Map             = computeSquarePuzzleMapping(4, 2)
	miniPuzzle1StartValues = []int{
		1, 0, 4, 5, 3, 2,
		0, 0, 3, 6, 0, 0,
		4, 0, 1, 3, 2, 0,
		0, 3, 6, 1, 0, 5,
		0, 0, 2, 4, 0, 0,
		3, 4, 5, 2, 0, 1,
	}
	miniPuzzle1CompleteValues = []int{
		1, 6, 4, 5, 3, 2,
		5, 2, 3, 6, 1, 4,
		4, 5, 1, 3, 2, 6,
		2, 3, 6, 1, 4, 5,
		6, 1, 2, 4, 5, 3,
		3, 4, 5, 2, 6, 1,
	}
	rotation4Puzzle1PartialValues = []int{
		1, 0, 3, 0,
		0, 3, 0, 1,
		3, 0, 1, 0,
		0, 1, 0, 3,
	}
	rotation4Puzzle1PartialPossibles = [][]int{
		[]int{1}, []int{2, 4}, []int{3}, []int{2, 4},
		[]int{2, 4}, []int{3}, []int{2, 4}, []int{1},
		[]int{3}, []int{2, 4}, []int{1}, []int{2, 4},
		[]int{2, 4}, []int{1}, []int{2, 4}, []int{3},
	}
	rotation4Puzzle1PartialSquares = []*square{
		nil,
		&square{index: 1, aval: 1},
		&square{index: 2, pvals: intset{2, 4}},
		&square{index: 3, aval: 3},
		&square{index: 4, pvals: intset{2, 4}},
		&square{index: 5, pvals: intset{2, 4}},
		&square{index: 6, aval: 3},
		&square{index: 7, pvals: intset{2, 4}},
		&square{index: 8, aval: 1},
		&square{index: 9, aval: 3},
		&square{index: 10, pvals: intset{2, 4}},
		&square{index: 11, aval: 1},
		&square{index: 12, pvals: intset{2, 4}},
		&square{index: 13, pvals: intset{2, 4}},
		&square{index: 14, aval: 1},
		&square{index: 15, pvals: intset{2, 4}},
		&square{index: 16, aval: 3},
	}
	rotation4Puzzle1PartialGroups = []*group{
		nil,
		&group{ // row 1
			&square4Map.gdescs[1], []int{0, 1, 0, 3, 0}, intset{2, 4}, intset{2, 4},
		},
		&group{ // row 2
			&square4Map.gdescs[2], []int{0, 8, 0, 6, 0}, intset{2, 4}, intset{5, 7},
		},
		&group{ // row 3
			&square4Map.gdescs[3], []int{0, 11, 0, 9, 0}, intset{2, 4}, intset{10, 12},
		},
		&group{ // row 4
			&square4Map.gdescs[4], []int{0, 14, 0, 16, 0}, intset{2, 4}, intset{13, 15},
		},
		&group{ // column 1
			&square4Map.gdescs[5], []int{0, 1, 0, 9, 0}, intset{2, 4}, intset{5, 13},
		},
		&group{ // column 2
			&square4Map.gdescs[6], []int{0, 14, 0, 6, 0}, intset{2, 4}, intset{2, 10},
		},
		&group{ // column 3
			&square4Map.gdescs[7], []int{0, 11, 0, 3, 0}, intset{2, 4}, intset{7, 15},
		},
		&group{ // column 4
			&square4Map.gdescs[8], []int{0, 8, 0, 16, 0}, intset{2, 4}, intset{4, 12},
		},
		&group{ // tile 1
			&square4Map.gdescs[9], []int{0, 1, 0, 6, 0}, intset{2, 4}, intset{2, 5},
		},
		&group{ // tile 2
			&square4Map.gdescs[10], []int{0, 8, 0, 3, 0}, intset{2, 4}, intset{4, 7},
		},
		&group{ // tile 3
			&square4Map.gdescs[11], []int{0, 14, 0, 9, 0}, intset{2, 4}, intset{10, 13},
		},
		&group{ // tile 4
			&square4Map.gdescs[12], []int{0, 11, 0, 16, 0}, intset{2, 4}, intset{12, 15},
		},
	}
	rotation4Puzzle1PartialAssign1Values = []int{ // assign(13, 2)
		1, 0, 3, 0,
		0, 3, 0, 1,
		3, 0, 1, 0,
		2, 1, 0, 3,
	}
	rotation4Puzzle1PartialAssign1Possibles = [][]int{
		[]int{1}, []int{2, 4}, []int{3}, []int{2, 4},
		[]int{4}, []int{3}, []int{2, 4}, []int{1},
		[]int{3}, []int{2, 4}, []int{1}, []int{2, 4},
		[]int{2}, []int{1}, []int{4}, []int{3},
	}
	rotation4Puzzle1PartialAssign1Squares = []*square{
		nil,
		&square{index: 1, aval: 1},
		&square{index: 2, pvals: intset{2, 4}, bval: 2, bsrc: helperBsrc(4+2, 8+1)},
		&square{index: 3, aval: 3},
		&square{index: 4, pvals: intset{2, 4}},
		&square{index: 5, pvals: intset{4}},
		&square{index: 6, aval: 3},
		&square{index: 7, pvals: intset{2, 4}, bval: 2, bsrc: helperBsrc(0+2, 4+3)},
		&square{index: 8, aval: 1},
		&square{index: 9, aval: 3},
		&square{index: 10, pvals: intset{4}},
		&square{index: 11, aval: 1},
		&square{index: 12, pvals: intset{2, 4}, bval: 2, bsrc: helperBsrc(0+3, 8+4)},
		&square{index: 13, aval: 2},
		&square{index: 14, aval: 1},
		&square{index: 15, pvals: intset{4}},
		&square{index: 16, aval: 3},
	}
	rotation4Puzzle1PartialAssign1Groups = []*group{
		nil,
		&group{ // row 1
			&square4Map.gdescs[1], []int{0, 1, 0, 3, 0}, intset{2, 4}, intset{2, 4},
		},
		&group{ // row 2
			&square4Map.gdescs[2], []int{0, 8, 0, 6, 0}, intset{}, intset{},
		},
		&group{ // row 3
			&square4Map.gdescs[3], []int{0, 11, 0, 9, 0}, intset{}, intset{},
		},
		&group{ // row 4
			&square4Map.gdescs[4], []int{0, 14, 13, 16, 0}, intset{}, intset{},
		},
		&group{ // column 1
			&square4Map.gdescs[5], []int{0, 1, 13, 9, 0}, intset{}, intset{},
		},
		&group{ // column 2
			&square4Map.gdescs[6], []int{0, 14, 0, 6, 0}, intset{}, intset{},
		},
		&group{ // column 3
			&square4Map.gdescs[7], []int{0, 11, 0, 3, 0}, intset{}, intset{},
		},
		&group{ // column 4
			&square4Map.gdescs[8], []int{0, 8, 0, 16, 0}, intset{2, 4}, intset{4, 12},
		},
		&group{ // tile 1
			&square4Map.gdescs[9], []int{0, 1, 0, 6, 0}, intset{}, intset{},
		},
		&group{ // tile 2
			&square4Map.gdescs[10], []int{0, 8, 0, 3, 0}, intset{2, 4}, intset{4, 7},
		},
		&group{ // tile 3
			&square4Map.gdescs[11], []int{0, 14, 13, 9, 0}, intset{}, intset{},
		},
		&group{ // tile 4
			&square4Map.gdescs[12], []int{0, 11, 0, 16, 0}, intset{}, intset{},
		},
	}
	rotation4Puzzle1PartialAssign1CapitalSquares = []Square{
		Square{Index: 1, Aval: 1},
		Square{Index: 2, Pvals: intset{2, 4},
			Bval: 2, Bsrc: []GroupID{GroupID{GtypeCol, 2}, GroupID{GtypeTile, 1}}},
		Square{Index: 3, Aval: 3},
		Square{Index: 4, Pvals: intset{2, 4}},
		Square{Index: 5, Pvals: intset{4}},
		Square{Index: 6, Aval: 3},
		Square{Index: 7, Pvals: intset{2, 4},
			Bval: 2, Bsrc: []GroupID{GroupID{GtypeRow, 2}, GroupID{GtypeCol, 3}}},
		Square{Index: 8, Aval: 1},
		Square{Index: 9, Aval: 3},
		Square{Index: 10, Pvals: intset{4}},
		Square{Index: 11, Aval: 1},
		Square{Index: 12, Pvals: intset{2, 4},
			Bval: 2, Bsrc: []GroupID{GroupID{GtypeRow, 3}, GroupID{GtypeTile, 4}}},
		Square{Index: 13, Aval: 2},
		Square{Index: 14, Aval: 1},
		Square{Index: 15, Pvals: intset{4}},
		Square{Index: 16, Aval: 3},
	}
	rotation4Puzzle1PartialAssign2Values = []int{ // assign(10, 4)
		1, 0, 3, 0,
		0, 3, 0, 1,
		3, 4, 1, 0,
		2, 1, 0, 3,
	}
	rotation4Puzzle1PartialAssign2Possibles = [][]int{
		[]int{1}, []int{2}, []int{3}, []int{2, 4},
		[]int{4}, []int{3}, []int{2, 4}, []int{1},
		[]int{3}, []int{4}, []int{1}, []int{2},
		[]int{2}, []int{1}, []int{4}, []int{3},
	}
	rotation4Puzzle1PartialAssign2Squares = []*square{
		nil,
		&square{index: 1, aval: 1},
		&square{index: 2, pvals: intset{2}, bval: 2, bsrc: helperBsrc(4+2, 8+1)},
		&square{index: 3, aval: 3},
		&square{index: 4, pvals: intset{2, 4}, bval: 4, bsrc: helperBsrc(0+1, 4+4)},
		&square{index: 5, pvals: intset{4}},
		&square{index: 6, aval: 3},
		&square{index: 7, pvals: intset{2, 4}, bval: 2, bsrc: helperBsrc(0+2, 4+3)},
		&square{index: 8, aval: 1},
		&square{index: 9, aval: 3},
		&square{index: 10, aval: 4},
		&square{index: 11, aval: 1},
		&square{index: 12, pvals: intset{2}, bval: 2, bsrc: helperBsrc(0+3, 8+4)},
		&square{index: 13, aval: 2},
		&square{index: 14, aval: 1},
		&square{index: 15, pvals: intset{4}},
		&square{index: 16, aval: 3},
	}
	rotation4Puzzle1PartialAssign2Groups = []*group{
		nil,
		&group{ // row 1
			&square4Map.gdescs[1], []int{0, 1, 0, 3, 0}, intset{}, intset{},
		},
		&group{ // row 2
			&square4Map.gdescs[2], []int{0, 8, 0, 6, 0}, intset{}, intset{},
		},
		&group{ // row 3
			&square4Map.gdescs[3], []int{0, 11, 0, 9, 10}, intset{}, intset{},
		},
		&group{ // row 4
			&square4Map.gdescs[4], []int{0, 14, 13, 16, 0}, intset{}, intset{},
		},
		&group{ // column 1
			&square4Map.gdescs[5], []int{0, 1, 13, 9, 0}, intset{}, intset{},
		},
		&group{ // column 2
			&square4Map.gdescs[6], []int{0, 14, 0, 6, 10}, intset{}, intset{},
		},
		&group{ // column 3
			&square4Map.gdescs[7], []int{0, 11, 0, 3, 0}, intset{}, intset{},
		},
		&group{ // column 4
			&square4Map.gdescs[8], []int{0, 8, 0, 16, 0}, intset{}, intset{},
		},
		&group{ // tile 1
			&square4Map.gdescs[9], []int{0, 1, 0, 6, 0}, intset{}, intset{},
		},
		&group{ // tile 2
			&square4Map.gdescs[10], []int{0, 8, 0, 3, 0}, intset{2, 4}, intset{4, 7},
		},
		&group{ // tile 3
			&square4Map.gdescs[11], []int{0, 14, 13, 9, 10}, intset{}, intset{},
		},
		&group{ // tile 4
			&square4Map.gdescs[12], []int{0, 11, 0, 16, 0}, intset{}, intset{},
		},
	}
	rotation4Puzzle1PartialAssign2CapitalSquares = []Square{
		Square{Index: 1, Aval: 1},
		Square{Index: 2, Pvals: intset{2}},
		Square{Index: 3, Aval: 3},
		Square{Index: 4, Pvals: intset{2, 4},
			Bval: 4, Bsrc: []GroupID{GroupID{GtypeRow, 1}, GroupID{GtypeCol, 4}}},
		Square{Index: 5, Pvals: intset{4}},
		Square{Index: 6, Aval: 3},
		Square{Index: 7, Pvals: intset{2, 4},
			Bval: 2, Bsrc: []GroupID{GroupID{GtypeRow, 2}, GroupID{GtypeCol, 3}}},
		Square{Index: 8, Aval: 1},
		Square{Index: 9, Aval: 3},
		Square{Index: 10, Aval: 4},
		Square{Index: 11, Aval: 1},
		Square{Index: 12, Pvals: intset{2}},
		Square{Index: 13, Aval: 2},
		Square{Index: 14, Aval: 1},
		Square{Index: 15, Pvals: intset{4}},
		Square{Index: 16, Aval: 3},
	}
	rotation4Puzzle1PartialAssign3Values = []int{ // assign(15, 4)
		1, 0, 3, 0,
		0, 3, 0, 1,
		3, 4, 1, 0,
		2, 1, 4, 3,
	}
	rotation4Puzzle1PartialAssign3Possibles = [][]int{
		[]int{1}, []int{2}, []int{3}, []int{2, 4},
		[]int{4}, []int{3}, []int{2}, []int{1},
		[]int{3}, []int{4}, []int{1}, []int{2},
		[]int{2}, []int{1}, []int{4}, []int{3},
	}
	rotation4Puzzle1PartialAssign3Squares = []*square{
		nil,
		&square{index: 1, aval: 1},
		&square{index: 2, pvals: intset{2}, bval: 2, bsrc: helperBsrc(4+2, 8+1)},
		&square{index: 3, aval: 3},
		&square{index: 4, pvals: intset{2, 4}, bval: 4, bsrc: helperBsrc(0+1, 4+4, 8+2)},
		&square{index: 5, pvals: intset{4}},
		&square{index: 6, aval: 3},
		&square{index: 7, pvals: intset{2}, bval: 2, bsrc: helperBsrc(0+2, 4+3)},
		&square{index: 8, aval: 1},
		&square{index: 9, aval: 3},
		&square{index: 10, aval: 4},
		&square{index: 11, aval: 1},
		&square{index: 12, pvals: intset{2}, bval: 2, bsrc: helperBsrc(0+3, 8+4)},
		&square{index: 13, aval: 2},
		&square{index: 14, aval: 1},
		&square{index: 15, aval: 4},
		&square{index: 16, aval: 3},
	}
	rotation4Puzzle1PartialAssign3Groups = []*group{
		nil,
		&group{ // row 1
			&square4Map.gdescs[1], []int{0, 1, 0, 3, 0}, intset{}, intset{},
		},
		&group{ // row 2
			&square4Map.gdescs[2], []int{0, 8, 0, 6, 0}, intset{}, intset{},
		},
		&group{ // row 3
			&square4Map.gdescs[3], []int{0, 11, 0, 9, 10}, intset{}, intset{},
		},
		&group{ // row 4
			&square4Map.gdescs[4], []int{0, 14, 13, 16, 15}, intset{}, intset{},
		},
		&group{ // column 1
			&square4Map.gdescs[5], []int{0, 1, 13, 9, 0}, intset{}, intset{},
		},
		&group{ // column 2
			&square4Map.gdescs[6], []int{0, 14, 0, 6, 10}, intset{}, intset{},
		},
		&group{ // column 3
			&square4Map.gdescs[7], []int{0, 11, 0, 3, 15}, intset{}, intset{},
		},
		&group{ // column 4
			&square4Map.gdescs[8], []int{0, 8, 0, 16, 0}, intset{}, intset{},
		},
		&group{ // tile 1
			&square4Map.gdescs[9], []int{0, 1, 0, 6, 0}, intset{}, intset{},
		},
		&group{ // tile 2
			&square4Map.gdescs[10], []int{0, 8, 0, 3, 0}, intset{}, intset{},
		},
		&group{ // tile 3
			&square4Map.gdescs[11], []int{0, 14, 13, 9, 10}, intset{}, intset{},
		},
		&group{ // tile 4
			&square4Map.gdescs[12], []int{0, 11, 0, 16, 15}, intset{}, intset{},
		},
	}
	rotation4Puzzle1PartialAssign3CapitalSquares = []Square{
		Square{Index: 1, Aval: 1},
		Square{Index: 2, Pvals: intset{2}},
		Square{Index: 3, Aval: 3},
		Square{Index: 4,
			Pvals: intset{2, 4},
			Bval:  4,
			Bsrc: []GroupID{
				GroupID{GtypeRow, 1},
				GroupID{GtypeCol, 4},
				GroupID{GtypeTile, 2},
			},
		},
		Square{Index: 5, Pvals: intset{4}},
		Square{Index: 6, Aval: 3},
		Square{Index: 7, Pvals: intset{2}},
		Square{Index: 8, Aval: 1},
		Square{Index: 9, Aval: 3},
		Square{Index: 10, Aval: 4},
		Square{Index: 11, Aval: 1},
		Square{Index: 12, Pvals: intset{2}},
		Square{Index: 13, Aval: 2},
		Square{Index: 14, Aval: 1},
		Square{Index: 15, Aval: 4},
		Square{Index: 16, Aval: 3},
	}
	rotation4Puzzle1Complete1 = []int{
		1, 2, 3, 4,
		4, 3, 2, 1,
		3, 4, 1, 2,
		2, 1, 4, 3,
	}
	rotation4Puzzle1Complete1CapitalSquares = []Square{
		Square{Index: 1, Aval: 1},
		Square{Index: 2, Aval: 2},
		Square{Index: 3, Aval: 3},
		Square{Index: 4, Aval: 4},
		Square{Index: 5, Aval: 4},
		Square{Index: 6, Aval: 3},
		Square{Index: 7, Aval: 2},
		Square{Index: 8, Aval: 1},
		Square{Index: 9, Aval: 3},
		Square{Index: 10, Aval: 4},
		Square{Index: 11, Aval: 1},
		Square{Index: 12, Aval: 2},
		Square{Index: 13, Aval: 2},
		Square{Index: 14, Aval: 1},
		Square{Index: 15, Aval: 4},
		Square{Index: 16, Aval: 3},
	}
	rotation4Puzzle1Complete2 = []int{
		1, 4, 3, 2,
		2, 3, 4, 1,
		3, 2, 1, 4,
		4, 1, 2, 3,
	}
	rotation4Puzzle2PartialValues = []int{
		1, 0, 3, 0,
		3, 0, 1, 0,
		2, 0, 4, 0,
		4, 0, 2, 0,
	}
	rotation4Puzzle2PartialPossibles = [][]int{
		[]int{1}, []int{2, 4}, []int{3}, []int{2, 4},
		[]int{3}, []int{2, 4}, []int{1}, []int{2, 4},
		[]int{2}, []int{1, 3}, []int{4}, []int{1, 3},
		[]int{4}, []int{1, 3}, []int{2}, []int{1, 3},
	}
	rotation4Puzzle2PartialSquares = []*square{
		nil,
		&square{index: 1, aval: 1},
		&square{index: 2, pvals: intset{2, 4}},
		&square{index: 3, aval: 3},
		&square{index: 4, pvals: intset{2, 4}},
		&square{index: 5, aval: 3},
		&square{index: 6, pvals: intset{2, 4}},
		&square{index: 7, aval: 1},
		&square{index: 8, pvals: intset{2, 4}},
		&square{index: 9, aval: 2},
		&square{index: 10, pvals: intset{1, 3}},
		&square{index: 11, aval: 4},
		&square{index: 12, pvals: intset{1, 3}},
		&square{index: 13, aval: 4},
		&square{index: 14, pvals: intset{1, 3}},
		&square{index: 15, aval: 2},
		&square{index: 16, pvals: intset{1, 3}},
	}
	rotation4Puzzle2PartialGroups = []*group{
		nil,
		&group{ // row 1
			&square4Map.gdescs[1], []int{0, 1, 0, 3, 0}, intset{2, 4}, intset{2, 4},
		},
		&group{ // row 2
			&square4Map.gdescs[2], []int{0, 7, 0, 5, 0}, intset{2, 4}, intset{6, 8},
		},
		&group{ // row 3
			&square4Map.gdescs[3], []int{0, 0, 9, 0, 11}, intset{1, 3}, intset{10, 12},
		},
		&group{ // row 4
			&square4Map.gdescs[4], []int{0, 0, 15, 0, 13}, intset{1, 3}, intset{14, 16},
		},
		&group{ // column 1
			&square4Map.gdescs[5], []int{0, 1, 9, 5, 13}, intset{}, intset{},
		},
		&group{ // column 2
			&square4Map.gdescs[6],
			[]int{0, 0, 0, 0, 0}, intset{1, 2, 3, 4}, intset{2, 6, 10, 14},
		},
		&group{ // column 3
			&square4Map.gdescs[7], []int{0, 7, 15, 3, 11}, intset{}, intset{},
		},
		&group{ // column 4
			&square4Map.gdescs[8],
			[]int{0, 0, 0, 0, 0}, intset{1, 2, 3, 4}, intset{4, 8, 12, 16},
		},
		&group{ // tile 1
			&square4Map.gdescs[9], []int{0, 1, 0, 5, 0}, intset{2, 4}, intset{2, 6},
		},
		&group{ // tile 2
			&square4Map.gdescs[10], []int{0, 7, 0, 3, 0}, intset{2, 4}, intset{4, 8},
		},
		&group{ // tile 3
			&square4Map.gdescs[11], []int{0, 0, 9, 0, 13}, intset{1, 3}, intset{10, 14},
		},
		&group{ // tile 4
			&square4Map.gdescs[12], []int{0, 0, 15, 0, 11}, intset{1, 3}, intset{12, 16},
		},
	}
	rotation4Puzzle2Complete1 = []int{
		1, 2, 3, 4,
		3, 4, 1, 2,
		2, 3, 4, 1,
		4, 1, 2, 3,
	}
	rotation4Puzzle2Complete2 = []int{
		1, 4, 3, 2,
		3, 2, 1, 4,
		4, 3, 2, 1,
		2, 1, 4, 3,
	}
	empty4PuzzleValues = []int{
		0, 0, 0, 0,
		0, 0, 0, 0,
		0, 0, 0, 0,
		0, 0, 0, 0,
	}
	empty4PuzzlePossibles = [][]int{
		[]int{1, 2, 3, 4}, []int{1, 2, 3, 4}, []int{1, 2, 3, 4}, []int{1, 2, 3, 4},
		[]int{1, 2, 3, 4}, []int{1, 2, 3, 4}, []int{1, 2, 3, 4}, []int{1, 2, 3, 4},
		[]int{1, 2, 3, 4}, []int{1, 2, 3, 4}, []int{1, 2, 3, 4}, []int{1, 2, 3, 4},
		[]int{1, 2, 3, 4}, []int{1, 2, 3, 4}, []int{1, 2, 3, 4}, []int{1, 2, 3, 4},
	}
	empty4PuzzleSquares = []*square{
		nil,
		&square{index: 1, pvals: intset{1, 2, 3, 4}},
		&square{index: 2, pvals: intset{1, 2, 3, 4}},
		&square{index: 3, pvals: intset{1, 2, 3, 4}},
		&square{index: 4, pvals: intset{1, 2, 3, 4}},
		&square{index: 5, pvals: intset{1, 2, 3, 4}},
		&square{index: 6, pvals: intset{1, 2, 3, 4}},
		&square{index: 7, pvals: intset{1, 2, 3, 4}},
		&square{index: 8, pvals: intset{1, 2, 3, 4}},
		&square{index: 9, pvals: intset{1, 2, 3, 4}},
		&square{index: 10, pvals: intset{1, 2, 3, 4}},
		&square{index: 11, pvals: intset{1, 2, 3, 4}},
		&square{index: 12, pvals: intset{1, 2, 3, 4}},
		&square{index: 13, pvals: intset{1, 2, 3, 4}},
		&square{index: 14, pvals: intset{1, 2, 3, 4}},
		&square{index: 15, pvals: intset{1, 2, 3, 4}},
		&square{index: 16, pvals: intset{1, 2, 3, 4}},
	}
	empty4PuzzleCapitalSquares = []Square{
		Square{Index: 1, Pvals: intset{1, 2, 3, 4}},
		Square{Index: 2, Pvals: intset{1, 2, 3, 4}},
		Square{Index: 3, Pvals: intset{1, 2, 3, 4}},
		Square{Index: 4, Pvals: intset{1, 2, 3, 4}},
		Square{Index: 5, Pvals: intset{1, 2, 3, 4}},
		Square{Index: 6, Pvals: intset{1, 2, 3, 4}},
		Square{Index: 7, Pvals: intset{1, 2, 3, 4}},
		Square{Index: 8, Pvals: intset{1, 2, 3, 4}},
		Square{Index: 9, Pvals: intset{1, 2, 3, 4}},
		Square{Index: 10, Pvals: intset{1, 2, 3, 4}},
		Square{Index: 11, Pvals: intset{1, 2, 3, 4}},
		Square{Index: 12, Pvals: intset{1, 2, 3, 4}},
		Square{Index: 13, Pvals: intset{1, 2, 3, 4}},
		Square{Index: 14, Pvals: intset{1, 2, 3, 4}},
		Square{Index: 15, Pvals: intset{1, 2, 3, 4}},
		Square{Index: 16, Pvals: intset{1, 2, 3, 4}},
	}
	empty4PuzzleGroups = []*group{
		nil,
		&group{ // row 1
			&square4Map.gdescs[1],
			[]int{0, 0, 0, 0, 0}, intset{1, 2, 3, 4}, intset{1, 2, 3, 4},
		},
		&group{ // row 2
			&square4Map.gdescs[2],
			[]int{0, 0, 0, 0, 0}, intset{1, 2, 3, 4}, intset{5, 6, 7, 8},
		},
		&group{ // row 3
			&square4Map.gdescs[3],
			[]int{0, 0, 0, 0, 0}, intset{1, 2, 3, 4}, intset{9, 10, 11, 12},
		},
		&group{ // row 4
			&square4Map.gdescs[4],
			[]int{0, 0, 0, 0, 0}, intset{1, 2, 3, 4}, intset{13, 14, 15, 16},
		},
		&group{ // column 1
			&square4Map.gdescs[5],
			[]int{0, 0, 0, 0, 0}, intset{1, 2, 3, 4}, intset{1, 5, 9, 13},
		},
		&group{ // column 2
			&square4Map.gdescs[6],
			[]int{0, 0, 0, 0, 0}, intset{1, 2, 3, 4}, intset{2, 6, 10, 14},
		},
		&group{ // column 3
			&square4Map.gdescs[7],
			[]int{0, 0, 0, 0, 0}, intset{1, 2, 3, 4}, intset{3, 7, 11, 15},
		},
		&group{ // column 4
			&square4Map.gdescs[8],
			[]int{0, 0, 0, 0, 0}, intset{1, 2, 3, 4}, intset{4, 8, 12, 16},
		},
		&group{ // tile 1
			&square4Map.gdescs[9],
			[]int{0, 0, 0, 0, 0}, intset{1, 2, 3, 4}, intset{1, 2, 5, 6},
		},
		&group{ // tile 2
			&square4Map.gdescs[10],
			[]int{0, 0, 0, 0, 0}, intset{1, 2, 3, 4}, intset{3, 4, 7, 8},
		},
		&group{ // tile 3
			&square4Map.gdescs[11],
			[]int{0, 0, 0, 0, 0}, intset{1, 2, 3, 4}, intset{9, 10, 13, 14},
		},
		&group{ // tile 4
			&square4Map.gdescs[12],
			[]int{0, 0, 0, 0, 0}, intset{1, 2, 3, 4}, intset{11, 12, 15, 16},
		},
	}
	empty4PuzzleAssign1Values = []int{
		1, 0, 0, 0,
		0, 0, 0, 0,
		0, 0, 0, 0,
		0, 0, 0, 0,
	}
	bound4PuzzleValues = []int{
		1, 2, 3, 0,
		0, 0, 0, 0,
		0, 0, 0, 0,
		0, 0, 0, 0,
	}
	bound4PuzzleSquares = []*square{
		nil,
		&square{index: 1, aval: 1},
		&square{index: 2, aval: 2},
		&square{index: 3, aval: 3},
		&square{index: 4, pvals: intset{4}},
		&square{index: 5, pvals: intset{3, 4}},
		&square{index: 6, pvals: intset{3, 4}},
		&square{index: 7, pvals: intset{1, 2, 4}},
		&square{index: 8, pvals: intset{1, 2, 4}},
		&square{index: 9, pvals: intset{2, 3, 4}},
		&square{index: 10, pvals: intset{1, 3, 4}},
		&square{index: 11, pvals: intset{1, 2, 4}},
		&square{index: 12, pvals: intset{1, 2, 3, 4}},
		&square{index: 13, pvals: intset{2, 3, 4}},
		&square{index: 14, pvals: intset{1, 3, 4}},
		&square{index: 15, pvals: intset{1, 2, 4}},
		&square{index: 16, pvals: intset{1, 2, 3, 4}},
	}
	conflicting4Puzzle1 = []int{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 0, 0,
		0, 0, 0, 0,
	}
	conflicting4Puzzle2 = []int{
		0, 0, 0, 0,
		0, 0, 2, 0,
		0, 0, 0, 0,
		0, 0, 2, 0,
	}
	conflicting4Puzzle3 = []int{
		0, 0, 0, 0,
		0, 0, 0, 0,
		3, 0, 0, 3,
		0, 0, 0, 0,
	}
	conflicting4Puzzle4 = []int{
		0, 0, 0, 0,
		0, 2, 0, 0,
		0, 0, 0, 0,
		0, 2, 0, 0,
	}
	unsatisfiable4Puzzle = []int{
		1, 0, 0, 0,
		0, 0, 0, 4,
		0, 0, 4, 0,
		0, 4, 0, 0,
	}
)

/*

Integer Sets

*/

func TestNewIntsetRange(t *testing.T) {
	ivals := []int{-1024, -3, 0, 1, 6, 17, 30, 150}
	norm := intset{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	for i := range ivals {
		out := newIntsetRange(i)
		if out == nil {
			t.Fatalf("Creating intset range(%d) produced nil", i)
		}
		if i < 1 {
			if len(out) != 0 {
				t.Errorf("Creating intset range(%d) produced non-empty result: %v", i, out)
			}
		} else if i <= len(norm) {
			if !reflect.DeepEqual(out, norm[:i]) {
				t.Errorf("Creating intset range(%d) produced %v, expected %v", i, out, norm[:i])
			}
		} else {
			if len(out) != i || out[i-1] != i || !reflect.DeepEqual(out[:len(norm)], norm) {
				t.Errorf("Creating intset range(%d) produced unexpected out: %v", i, out)
			}
		}
	}
}

func TestNewIntsetCopy(t *testing.T) {
	testcases := []intset{
		nil,
		intset{},
		newIntsetRange(5),
		newIntsetRange(100),
		intset{-3, -100, 50, 3, 19, 275},
		intset{3, 7, 9},
	}
	for _, tc := range testcases {
		out := newIntsetCopy(tc)
		if !reflect.DeepEqual(out, tc) {
			t.Errorf("newIntsetCopy(%v) produced different output: %v", tc, out)
		}
	}
}

func TestIntsetFind(t *testing.T) {
	// keeping it simple is best, this is not a complex function
	inputpvals := []int{3, 4, 5, 6, 7, 1}
	inputIntset := []intset{
		intset{2, 3, 5, 6},
		intset{2, 3, 5, 6},
		intset{2, 3, 5, 6},
		intset{2, 3, 5, 6},
		intset{2, 3, 5, 6},
		intset{2, 3, 5, 6},
	}
	outputindices := []int{1, 2, 2, 3, 4, 0}
	outputFlags := []bool{true, false, true, true, false, false}
	for i, inPval := range inputpvals {
		where, found := inputIntset[i].find(inPval)
		if where != outputindices[i] || found != outputFlags[i] {
			t.Errorf("%v.find(%d) gave %d, %v, expected %d, %v",
				inputIntset[i], inPval, where, found, outputindices[i], outputFlags[i])
		}
	}
}

func TestIntsetInsert(t *testing.T) {
	// just like TestIntsetFind, but does the insertion.
	inputpvals := []int{3, 4, 5, 6, 7, 1}
	inputIntset := []intset{
		intset{2, 3, 5, 6},
		intset{2, 3, 5, 6},
		intset{2, 3, 5, 6},
		intset{2, 3, 5, 6},
		intset{2, 3, 5, 6},
		intset{2, 3, 5, 6},
	}
	outputIntset := []intset{
		intset{2, 3, 5, 6},
		intset{2, 3, 4, 5, 6},
		intset{2, 3, 5, 6},
		intset{2, 3, 5, 6},
		intset{2, 3, 5, 6, 7},
		intset{1, 2, 3, 5, 6},
	}
	outputFlags := []bool{true, false, true, true, false, false}
	for i, inPval := range inputpvals {
		input := newIntsetCopy(inputIntset[i])
		found := input.insert(inPval)
		if !reflect.DeepEqual(input, outputIntset[i]) || found != outputFlags[i] {
			t.Errorf("%v.insert(%d) gave %v, %v expected %v, %v",
				inputIntset[i], inPval, input, found, outputIntset[i], outputFlags[i])
		}
	}
}

func TestIntsetRemove(t *testing.T) {
	// like Intset.insert, so use essentially the same tests.
	inputIvals := []int{3, 4, 5, 6, 7, 1}
	inputIntsets := []intset{
		intset{2, 3, 5, 6},
		intset{2, 3, 5, 6},
		intset{2, 3, 5, 6},
		intset{2, 3, 5, 6},
		intset{2, 3, 5, 6},
		intset{2, 3, 5, 6},
	}
	outputIntsets := []intset{
		intset{2, 5, 6},
		intset{2, 3, 5, 6},
		intset{2, 3, 6},
		intset{2, 3, 5},
		intset{2, 3, 5, 6},
		intset{2, 3, 5, 6},
	}
	for i, inIval := range inputIvals {
		input := newIntsetCopy(inputIntsets[i])
		input.remove(inIval)
		if !reflect.DeepEqual(input, outputIntsets[i]) {
			t.Errorf("%v.remove(%d) is %v, expected %v",
				inputIntsets[i], inIval, input, outputIntsets[i])
		}
	}
}

type intsetSubtractTestcase struct {
	starter    intset
	marker     int
	tosubtract intset
	remaining  intset
	removed    bool
	gotmarker  bool
}

func TestIntsetSubtract(t *testing.T) {
	testcases := []intsetSubtractTestcase{
		intsetSubtractTestcase{ // input equal to target
			newIntsetRange(9), 0,
			newIntsetRange(9),
			intset{},
			true, false,
		},
		intsetSubtractTestcase{ // input overlaps target
			newIntsetRange(9), -1,
			intset{0, 3, 4, 6, 9, 12, 13, 15, 16, 17},
			intset{1, 2, 5, 7, 8},
			true, false,
		},
		intsetSubtractTestcase{ // input subset of target
			newIntsetRange(9), 0,
			intset{2, 5, 7, 8},
			intset{1, 3, 4, 6, 9},
			true, false,
		},
		intsetSubtractTestcase{ // input overlaps and disjoint from target
			intset{3, 4, 6, 8}, 0,
			intset{1, 2, 5, 7, 9},
			intset{3, 4, 6, 8},
			false, false,
		},
		intsetSubtractTestcase{ // input internal to and disjoint from target
			intset{1, 4, 6, 9}, 0,
			intset{2, 3, 5, 7, 8},
			intset{1, 4, 6, 9},
			false, false,
		},
		intsetSubtractTestcase{ // input leaves just one possible, which is marker
			intset{3, 4, 6, 9}, 6,
			intset{1, 2, 3, 4, 5, 7, 8, 9},
			intset{6},
			true, false,
		},
		// same tests using larger squares
		intsetSubtractTestcase{ // input equal to target
			newIntsetRange(16), 0,
			newIntsetRange(16),
			intset{},
			true, false,
		},
		intsetSubtractTestcase{ // input overlaps target
			newIntsetRange(16), -1,
			intset{0, 3, 4, 6, 9, 12, 13, 15, 16, 17},
			intset{1, 2, 5, 7, 8, 10, 11, 14},
			true, false,
		},
		intsetSubtractTestcase{ // input subset of target
			newIntsetRange(16), 0,
			intset{2, 5, 7, 8, 10, 11, 14},
			intset{1, 3, 4, 6, 9, 12, 13, 15, 16},
			true, false,
		},
		intsetSubtractTestcase{ // input overlaps and disjoint from target
			intset{3, 4, 6, 8, 10, 15}, 0,
			intset{1, 2, 5, 7, 9, 11, 13, 16},
			intset{3, 4, 6, 8, 10, 15},
			false, false,
		},
		intsetSubtractTestcase{ // input internal to and disjoint from target
			intset{1, 4, 6, 9, 12, 13, 15, 16}, 0,
			intset{2, 3, 5, 7, 8, 10, 11, 14},
			intset{1, 4, 6, 9, 12, 13, 15, 16},
			false, false,
		},
		intsetSubtractTestcase{ // input leaves just one possible, which is marker
			intset{3, 4, 6, 9, 12, 13, 15, 16}, 16,
			intset{1, 2, 3, 4, 5, 6, 7, 8, 9, 12, 13, 15},
			intset{16},
			true, false,
		},
		// marker tests
		intsetSubtractTestcase{ // marker at start
			newIntsetRange(9), 1,
			intset{1, 3, 5},
			intset{2, 4, 6, 7, 8, 9},
			true, true,
		},
		intsetSubtractTestcase{ // marker in middle
			newIntsetRange(9), 3,
			intset{1, 3, 5},
			intset{2, 4, 6, 7, 8, 9},
			true, true,
		},
		intsetSubtractTestcase{ // marker at end
			newIntsetRange(9), 9,
			intset{1, 3, 9},
			intset{2, 4, 5, 6, 7, 8},
			true, true,
		},
		intsetSubtractTestcase{ // marker in input but not target
			intset{1, 5, 9}, 3,
			intset{1, 3, 9},
			intset{5},
			true, false,
		},
	}
	for _, tc := range testcases {
		// dup input to preserve test case for error messages
		input := newIntsetCopy(tc.starter)
		removed, gotmarker := input.subtract(tc.tosubtract, tc.marker)
		if !reflect.DeepEqual(input, tc.remaining) {
			t.Errorf("intset.remove(%v, %d) from %v left %v not %v",
				tc.tosubtract, tc.marker, tc.starter, input, tc.remaining)
		}
		if removed != tc.removed || gotmarker != tc.gotmarker {
			t.Errorf("intset.remove(%v, %d) from %v returned (%v, %v) not (%v, %v)",
				tc.tosubtract, tc.marker, tc.starter,
				removed, tc.removed, gotmarker, tc.gotmarker)
		}
	}
}

type intsetIntersectTestcase struct {
	starter   intset
	marker    int
	tokeep    intset
	remaining intset
	removed   bool
	gotmarker bool
}

func TestIntsetIntersect(t *testing.T) {
	testcases := []intsetIntersectTestcase{
		intsetIntersectTestcase{ // input equal to target
			newIntsetRange(9), 0,
			newIntsetRange(9),
			newIntsetRange(9),
			false, false,
		},
		intsetIntersectTestcase{ // input overlaps target
			newIntsetRange(9), -1,
			intset{0, 3, 4, 6, 9, 12, 13, 15, 16, 17},
			intset{3, 4, 6, 9},
			true, false,
		},
		intsetIntersectTestcase{ // input subset of target
			newIntsetRange(9), 0,
			intset{2, 5, 7, 8},
			intset{2, 5, 7, 8},
			true, false,
		},
		intsetIntersectTestcase{ // input internal to and disjoint from target
			intset{3, 4, 6, 8}, 0,
			intset{1, 2, 5, 7, 9},
			intset{},
			true, false,
		},
		intsetIntersectTestcase{ // input overlaps and disjoint from target
			intset{1, 4, 6, 9}, 0,
			intset{2, 3, 5, 7, 8},
			intset{},
			true, false,
		},
		intsetIntersectTestcase{ // input leaves just one possible, which is marker
			intset{3, 4, 6, 9}, 6,
			intset{1, 6, 12},
			intset{6},
			true, false,
		},
		// same tests using larger squares
		intsetIntersectTestcase{ // input equal to target
			newIntsetRange(16), 0,
			newIntsetRange(16),
			newIntsetRange(16),
			false, false,
		},
		intsetIntersectTestcase{ // input overlaps target
			newIntsetRange(16), -1,
			intset{0, 3, 4, 6, 9, 12, 13, 15, 16, 17},
			intset{3, 4, 6, 9, 12, 13, 15, 16},
			true, false,
		},
		intsetIntersectTestcase{ // input subset of target
			newIntsetRange(16), 0,
			intset{2, 5, 7, 8, 10, 11, 14},
			intset{2, 5, 7, 8, 10, 11, 14},
			true, false,
		},
		intsetIntersectTestcase{ // input internal to and disjoint from target
			intset{3, 4, 6, 8, 10, 15}, 0,
			intset{1, 2, 5, 7, 9, 11, 13, 16},
			intset{},
			true, false,
		},
		intsetIntersectTestcase{ // input overlaps and disjoint from target
			intset{1, 4, 6, 9, 12, 13, 15, 16}, 0,
			intset{2, 3, 5, 7, 8, 10, 11, 14},
			intset{},
			true, false,
		},
		intsetIntersectTestcase{ // input leaves just one possible, which is marker
			intset{3, 4, 6, 9, 12, 13, 15, 16}, 16,
			intset{1, 5, 14, 16, 17, 19},
			intset{16},
			true, false,
		},
		// marker tests
		intsetIntersectTestcase{ // marker at start
			newIntsetRange(9), 1,
			intset{2, 3, 5},
			intset{2, 3, 5},
			true, true,
		},
		intsetIntersectTestcase{ // marker in middle
			newIntsetRange(9), 3,
			intset{0, 1, 4, 5},
			intset{1, 4, 5},
			true, true,
		},
		intsetIntersectTestcase{ // marker at end (tail past intersection)
			newIntsetRange(9), 9,
			intset{1, 3, 6},
			intset{1, 3, 6},
			true, true,
		},
		intsetIntersectTestcase{ // marker in input but not target
			intset{1, 5, 6, 9}, 3,
			intset{1, 3, 5, 9},
			intset{1, 5, 9},
			true, false,
		},
	}
	for _, tc := range testcases {
		// dup input to preserve test case for error messages
		input := newIntsetCopy(tc.starter)
		removed, gotmarker := input.intersect(tc.tokeep, tc.marker)
		if !reflect.DeepEqual(input, tc.remaining) {
			t.Errorf("intset.intersect(%v, %d) from %v left %v not %v",
				tc.tokeep, tc.marker, tc.starter, input, tc.remaining)
		}
		if removed != tc.removed || gotmarker != tc.gotmarker {
			t.Errorf("intset.intersect(%v, %d) from %v returned (%v, %v) not (%v, %v)",
				tc.tokeep, tc.marker, tc.starter,
				removed, tc.removed, gotmarker, tc.gotmarker)
		}
	}
}

type intsetRemoveBenchcase struct {
	starter  intset
	toremove int
}

func BenchmarkIntsetRemove(b *testing.B) {
	testcases := []intsetRemoveBenchcase{
		intsetRemoveBenchcase{
			newIntsetRange(9),
			12,
		},
		intsetRemoveBenchcase{
			newIntsetRange(9),
			1,
		},
		intsetRemoveBenchcase{
			newIntsetRange(9),
			10,
		},
		intsetRemoveBenchcase{
			intset{6, 9},
			6,
		},
		intsetRemoveBenchcase{
			newIntsetRange(16),
			16,
		},
		intsetRemoveBenchcase{
			newIntsetRange(16),
			1,
		},
		intsetRemoveBenchcase{
			newIntsetRange(16),
			25,
		},
		intsetRemoveBenchcase{
			intset{3, 16},
			16,
		},
	}

	for i := 0; i < b.N; i++ {
		for _, tc := range testcases {
			// dup input intset to preserve test case for next loop
			input := newIntsetCopy(tc.starter)
			input.remove(tc.toremove)
		}
	}
}

type intsetSubtractBenchcase struct {
	starter    intset
	tosubtract intset
}

func BenchmarkIntsetSubtractMulti(b *testing.B) {
	testcases := []intsetSubtractBenchcase{
		intsetSubtractBenchcase{
			newIntsetRange(9),
			intset{0, 3, 4, 6, 9, 12, 13, 15, 16, 17},
		},
		intsetSubtractBenchcase{
			newIntsetRange(9),
			intset{1, 2, 5, 7, 8},
		},
		intsetSubtractBenchcase{
			intset{3, 4, 6, 9},
			intset{1, 2, 3, 4, 5, 7, 8, 9},
		},
		intsetSubtractBenchcase{
			intset{3, 4, 6, 9},
			intset{1, 2, 3, 4, 5, 6, 7, 8},
		},
		intsetSubtractBenchcase{
			newIntsetRange(16),
			intset{0, 3, 4, 6, 9, 12, 13, 15, 16, 17},
		},
		intsetSubtractBenchcase{
			newIntsetRange(16),
			intset{1, 2, 5, 7, 8, 10, 11, 14},
		},
		intsetSubtractBenchcase{
			intset{3, 4, 6, 9, 12, 13, 15, 16},
			intset{1, 2, 3, 4, 5, 6, 7, 8, 9, 12, 13, 15},
		},
		intsetSubtractBenchcase{
			intset{3, 4, 6, 9, 12, 13, 15, 16},
			intset{1, 2, 4, 5, 6, 7, 8, 9, 12, 13, 15, 16},
		},
	}

	for i := 0; i < b.N; i++ {
		for _, tc := range testcases {
			// dup input intset to preserve test case for next loop
			input := newIntsetCopy(tc.starter)
			input.subtract(tc.tosubtract, -1)
		}
	}
}

/*

Squares

*/

// remember, no error checking in this function
func TestNewEmptySquares(t *testing.T) {
	sidelens := []int{9, 13, 16}
	indices := []int{-1, 0, 1, 12, 13, 80, 81, 255, 256, 257, 300}

	for _, s := range sidelens {
		for _, i := range indices {
			sq := newEmptySquare(i, s, nil)
			if sq.index != i || sq.aval != 0 || sq.bval != 0 || sq.bsrc != nil ||
				!reflect.DeepEqual(sq.pvals, newIntsetRange(s)) {
				t.Fatalf("newEmptySquare(%d, %d) incorrect: %v", i, s, sq)
			}
		}
	}
}

// remember, no error checking in this function
func TestNewFilledSquares(t *testing.T) {
	sidelens := []int{9, 13, 16}
	indices := []int{-1, 0, 1, 12, 13, 80, 81, 255, 256, 257, 300}
	values := []int{0, 1, 8, 9, 10, 12, 13, 14, 15, 16, 17, 19}

	for _, s := range sidelens {
		for _, i := range indices {
			for _, v := range values {
				sq := newFilledSquare(i, s, v, nil)
				if sq.index != i || sq.aval != v ||
					sq.bval != 0 || sq.bsrc != nil ||
					sq.pvals != nil {
					t.Fatalf("newFilledSquare(%d, %d, %d) incorrect: %v", i, s, v, sq)
				}
			}
		}
	}
}

type squareAssignErrcase struct {
	square   *square
	toassign int
	cond     ErrorCondition
}

type squareAssignTestcase struct {
	square   *square
	toassign int
	bsrc     []GroupID
}

func TestSquareAssign(t *testing.T) {
	errcases := []squareAssignErrcase{
		squareAssignErrcase{
			&square{index: 2, pvals: intset{3, 4, 5, 7}, bval: 4, bsrc: helperBsrc(5)},
			3,
			NoGroupValueCondition,
		},
		squareAssignErrcase{
			&square{index: 1, pvals: intset{3, 5}},
			4,
			NotInSetCondition,
		},
	}
	for _, e := range errcases {
		input := helperDupSquare(e.square)
		if errs := input.assign(e.toassign); len(errs) == 0 {
			t.Errorf("Assign of %v to %+v didn't err", e.toassign, *e.square)
		} else {
			t.Logf("Assign of %v to %+v: %v", e.toassign, *e.square, errs)
			if !helperCheckCondition(e.cond, errs) {
				t.Errorf("Wrong error!")
			}
		}
	}

	testcases := []squareAssignTestcase{
		squareAssignTestcase{ // one in the middle
			&square{index: 1, pvals: intset{1, 2, 3, 4, 5, 6, 7, 8, 9}},
			4,
			nil,
		},
		squareAssignTestcase{ // one at the end
			&square{index: 2, pvals: intset{3, 4, 6, 9}},
			9,
			nil,
		},
		squareAssignTestcase{ // one at the beginning
			&square{index: 3, pvals: intset{3, 4, 6, 9}},
			3,
			nil,
		},
		squareAssignTestcase{ // one already bound, with a binding source
			&square{index: 4, pvals: intset{7, 9}, bval: 9, bsrc: helperBsrc(4)},
			9,
			helperBsrc(4),
		},
		squareAssignTestcase{ // one already bound, with a double binding source
			&square{index: 5, pvals: intset{3, 5, 9}, bval: 9, bsrc: helperBsrc(1, 10)},
			9,
			helperBsrc(1, 10),
		},
	}
	for _, tc := range testcases {
		// dup input square to preserve test case for error messages
		input := helperDupSquare(tc.square)
		errs := input.assign(tc.toassign)
		if len(errs) != 0 {
			t.Fatalf("Assigning %v to %v produced errors: %v", tc.toassign, *tc.square, errs)
		}
		if input.aval != tc.toassign {
			t.Errorf("Assigning %v to %v gave assignment %v",
				tc.toassign, *tc.square, input.aval)
		}
		if input.pvals != nil {
			t.Errorf("Assigning %v to %v gave pvals %v",
				tc.toassign, *tc.square, input.pvals)
		}
		if input.bval != tc.square.bval || !reflect.DeepEqual(input.bsrc, tc.bsrc) {
			t.Errorf("Assigning %v to %v gave binding %v: %v not %v: %v",
				tc.toassign, *tc.square, input.bval, input.bsrc, tc.toassign, tc.bsrc)
		}
	}
}

type squareBindErrcase struct {
	square *square
	tobind int
	bsrcin GroupID
	cond   ErrorCondition
}

type squareBindTestcase struct {
	square  *square
	tobind  int
	bsrcin  GroupID
	bsrcout []GroupID
}

func TestSquareBind(t *testing.T) {
	errcases := []squareBindErrcase{
		squareBindErrcase{
			&square{index: 2, bval: 4, bsrc: helperBsrc(6), pvals: intset{3, 4, 5, 6}},
			3, helperGID(102),
			NoGroupValueCondition,
		},
		squareBindErrcase{
			&square{index: 3, pvals: intset{3, 5}},
			4, helperGID(103),
			NotInSetCondition,
		},
		squareBindErrcase{
			&square{index: 4, pvals: intset{5}},
			4, helperGID(103),
			NotInSetCondition,
		},
	}
	for _, e := range errcases {
		input := helperDupSquare(e.square)
		if errs := input.bind(e.tobind, e.bsrcin); len(errs) == 0 {
			t.Errorf("Binding %v to %+v didn't return error", e.tobind, *e.square)
		} else {
			t.Logf("Bind of %v to %+v: %v", e.tobind, *e.square, errs)
			if !helperCheckCondition(e.cond, errs) {
				t.Errorf("Wrong error!")
			}
		}
	}

	testcases := []squareBindTestcase{
		squareBindTestcase{ // one in the middle
			&square{index: 1, pvals: intset{1, 2, 3, 4, 5, 6, 7, 8, 9}},
			4, helperGID(101),
			helperBsrc(101),
		},
		squareBindTestcase{ // one at the end
			&square{index: 2, pvals: intset{3, 4, 6, 9}},
			9, helperGID(102),
			helperBsrc(102),
		},
		squareBindTestcase{ // one at the beginning
			&square{index: 3, pvals: intset{3, 4, 6, 9}},
			3, helperGID(103),
			helperBsrc(103),
		},
		squareBindTestcase{ // one already bound, with a binding source
			&square{index: 4, bval: 9, pvals: intset{7, 9}, bsrc: helperBsrc(7)},
			9, helperGID(6),
			helperBsrc(7, 6),
		},
		squareBindTestcase{ // one already bound, with a double binding source
			&square{index: 6, pvals: intset{3, 5, 9}, bval: 9, bsrc: helperBsrc(4, 7)},
			9, helperGID(8),
			helperBsrc(4, 7, 8),
		},
		squareBindTestcase{ // one with a single value
			&square{index: 7, pvals: intset{1}},
			1, helperGID(1),
			helperBsrc(1),
		},
	}
	for _, tc := range testcases {
		// dup input square to preserve test case for error messages
		input := helperDupSquare(tc.square)
		errs := input.bind(tc.tobind, tc.bsrcin)
		if len(errs) != 0 {
			t.Fatalf("Binding %v to %v produced errors %v", tc.tobind, *tc.square, errs)
		}
		if !reflect.DeepEqual(input.pvals, tc.square.pvals) {
			t.Errorf("Binding %v to %v altered possible values to %v",
				tc.tobind, *tc.square, input.pvals)
		}
		if input.aval != tc.square.aval {
			t.Errorf("Binding %v to %v altered assignment to %v",
				tc.tobind, *tc.square, input.aval)
		}
		if input.bval != tc.tobind ||
			!reflect.DeepEqual(input.bsrc, tc.bsrcout) {
			t.Errorf("Binding %v to %v gave binding %v: %v not %v: %v",
				tc.tobind, *tc.square, input.bval, input.bsrc, tc.tobind, tc.bsrcout)
		}
	}
}

type squareRemoveErrcase struct {
	square   *square
	toremove int
	cond     ErrorCondition
}

type squareRemoveTestcase struct {
	square    *square
	toremove  int
	remaining intset
	bval      int
	bsrc      []GroupID
}

func TestSquareRemove(t *testing.T) {
	errcases := []squareRemoveErrcase{
		squareRemoveErrcase{
			helperBindSquare(newEmptySquare(2, 9, nil), 5, helperGID(2)),
			5,
			NoGroupValueCondition,
		},
		squareRemoveErrcase{
			&square{index: 3, pvals: intset{6}},
			6,
			NoPossibleValuesCondition,
		},
	}
	for _, e := range errcases {
		input := helperDupSquare(e.square)
		if errs := input.remove(e.toremove); len(errs) == 0 {
			t.Errorf("Removal of %v from %v didn't return error", e.toremove, *e.square)
		} else {
			t.Logf("Removal of %v from %+v: %v", e.toremove, *e.square, errs)
			if !helperCheckCondition(e.cond, errs) {
				t.Errorf("Wrong error!")
			}
		}
	}

	testcases := []squareRemoveTestcase{
		squareRemoveTestcase{ // input one of many
			newEmptySquare(1, 9, nil),
			6,
			intset{1, 2, 3, 4, 5, 7, 8, 9},
			0, nil,
		},
		squareRemoveTestcase{ // input not present
			&square{index: 3, pvals: intset{3, 4, 6, 9}},
			2,
			intset{3, 4, 6, 9},
			0, nil,
		},
		squareRemoveTestcase{ // input leaves just one possible
			&square{index: 4, pvals: intset{6, 9}},
			9,
			intset{6},
			0, nil,
		},
		squareRemoveTestcase{ // reduce to already bound
			&square{index: 105, pvals: intset{3, 12}, bval: 3, bsrc: helperBsrc(5)},
			12,
			intset{3},
			3, helperBsrc(5),
		},
	}
	for _, tc := range testcases {
		// dup input square to preserve test case for error messages
		input := helperDupSquare(tc.square)
		e := input.remove(tc.toremove)
		if e != nil {
			t.Fatalf("Removing %v from %v produced error %v", tc.toremove, tc.square, e)
		}
		if !reflect.DeepEqual(input.pvals, tc.remaining) {
			t.Errorf("Removing %v from %v left %v not %v",
				tc.toremove, *tc.square, input.pvals, tc.remaining)
		}
		if input.bval != tc.bval || !reflect.DeepEqual(input.bsrc, tc.bsrc) {
			t.Errorf("Removing %v from %v left binding %v: %v not %v: %v",
				tc.toremove, *tc.square, input.bval, input.bsrc, tc.bval, tc.bsrc)
		}
	}
}

type squareSubtractErrcase struct {
	square     *square
	tosubtract intset
	cond       ErrorCondition
}

type squareSubtractTestcase struct {
	square     *square
	tosubtract intset
	remaining  intset
	bval       int
	bsrc       []GroupID
}

func TestSquareSubtract(t *testing.T) {
	errcases := []squareSubtractErrcase{
		squareSubtractErrcase{
			helperBindSquare(newEmptySquare(2, 9, nil), 5, helperGID(2)),
			intset{1, 3, 5},
			NoGroupValueCondition,
		},
		squareSubtractErrcase{
			&square{index: 3, pvals: intset{3, 5}},
			intset{1, 3, 5},
			NoPossibleValuesCondition,
		},
	}
	for _, e := range errcases {
		input := helperDupSquare(e.square)
		if errs := input.subtract(e.tosubtract); len(errs) == 0 {
			t.Errorf("Removal of %v from %v didn't return error", e.tosubtract, *e.square)
		} else {
			t.Logf("Removal of %v from %+v: %v", e.tosubtract, *e.square, errs)
			if !helperCheckCondition(e.cond, errs) {
				t.Errorf("Wrong error!")
			}
		}
	}

	testcases := []squareSubtractTestcase{
		squareSubtractTestcase{ // input larger than range
			newEmptySquare(1, 9, nil),
			intset{0, 3, 4, 6, 9, 12, 13, 15, 16, 17},
			intset{1, 2, 5, 7, 8},
			0, nil,
		},
		squareSubtractTestcase{ // input subset of empty square
			newEmptySquare(2, 9, nil),
			intset{1, 2, 5, 7, 8},
			intset{3, 4, 6, 9},
			0, nil,
		},
		squareSubtractTestcase{ // input disjoint from range
			&square{index: 3, pvals: intset{3, 4, 6, 9}},
			intset{1, 2, 5, 7, 8},
			intset{3, 4, 6, 9},
			0, nil,
		},
		squareSubtractTestcase{ // input leaves just one possible
			&square{index: 4, pvals: intset{3, 4, 6, 9}},
			intset{1, 2, 3, 4, 5, 7, 8, 9},
			intset{6},
			0, nil,
		},
		squareSubtractTestcase{ // reduce to already bound
			&square{index: 105, pvals: intset{3, 4, 6, 9, 12, 13, 15, 16},
				bval: 3, bsrc: helperBsrc(9)},
			intset{1, 2, 4, 5, 6, 7, 8, 9, 12, 13, 15, 16},
			intset{3},
			3, helperBsrc(9),
		},
		// same first four tests using larger squares
		squareSubtractTestcase{ // input larger than range
			newEmptySquare(101, 16, nil),
			intset{0, 3, 4, 6, 9, 12, 13, 15, 16, 17},
			intset{1, 2, 5, 7, 8, 10, 11, 14},
			0, nil,
		},
		squareSubtractTestcase{ // input subset of empty square
			newEmptySquare(102, 16, nil),
			intset{1, 2, 5, 7, 8, 10, 11, 14},
			intset{3, 4, 6, 9, 12, 13, 15, 16},
			0, nil,
		},
		squareSubtractTestcase{ // input disjoint from range
			&square{index: 103, pvals: intset{3, 4, 6, 9, 12, 13, 15, 16}},
			intset{1, 2, 5, 7, 8, 10, 11, 14},
			intset{3, 4, 6, 9, 12, 13, 15, 16},
			0, nil,
		},
		squareSubtractTestcase{ // input leaves just one possible
			&square{index: 104, pvals: intset{3, 4, 6, 9, 12, 13, 15, 16}},
			intset{1, 2, 3, 4, 5, 6, 7, 8, 9, 12, 13, 15},
			intset{16},
			0, nil,
		},
	}
	for _, tc := range testcases {
		// dup input square to preserve test case for error messages
		input := helperDupSquare(tc.square)
		e := input.subtract(tc.tosubtract)
		if e != nil {
			t.Fatalf("Removing %v from %v produced error %v", tc.tosubtract, tc.square, e)
		}
		if !reflect.DeepEqual(input.pvals, tc.remaining) {
			t.Errorf("Removing %v from %v left %v not %v",
				tc.tosubtract, *tc.square, input.pvals, tc.remaining)
		}
		if input.bval != tc.bval || !reflect.DeepEqual(input.bsrc, tc.bsrc) {
			t.Errorf("Removing %v from %v left binding %v: %v not %v: %v",
				tc.tosubtract, *tc.square, input.bval, input.bsrc, tc.bval, tc.bsrc)
		}
	}
}

type squareIntersectErrcase struct {
	square      *square
	tointersect intset
	cond        ErrorCondition
}

type squareIntersectTestcase struct {
	square      *square
	tointersect intset
	remaining   intset
	bval        int
	bsrc        []GroupID
}

func TestSquareIntersect(t *testing.T) {
	errcases := []squareIntersectErrcase{
		squareIntersectErrcase{
			helperBindSquare(newEmptySquare(2, 9, nil), 5, helperGID(2)),
			intset{1, 3},
			NoGroupValueCondition,
		},
		squareIntersectErrcase{
			&square{index: 3, pvals: intset{3, 5}},
			intset{1, 2, 4},
			NoPossibleValuesCondition,
		},
	}
	for _, e := range errcases {
		input := helperDupSquare(e.square)
		if errs := input.intersect(e.tointersect); len(errs) == 0 {
			t.Errorf("Intersection of %v with %v didn't return error",
				e.tointersect, *e.square)
		} else {
			t.Logf("Intersection of %v with %+v: %v", e.tointersect, *e.square, errs)
			if !helperCheckCondition(e.cond, errs) {
				t.Errorf("Wrong error!")
			}
		}
	}

	testcases := []squareIntersectTestcase{
		squareIntersectTestcase{ // input larger than range
			newEmptySquare(1, 9, nil),
			intset{0, 3, 4, 6, 9, 12, 13, 15, 16, 17},
			intset{3, 4, 6, 9},
			0, nil,
		},
		squareIntersectTestcase{ // input subset of empty square
			newEmptySquare(2, 9, nil),
			intset{1, 2, 5, 7, 8},
			intset{1, 2, 5, 7, 8},
			0, nil,
		},
		squareIntersectTestcase{ // input equal to range
			&square{index: 3, pvals: intset{3, 4, 6, 9}},
			intset{3, 4, 6, 9},
			intset{3, 4, 6, 9},
			0, nil,
		},
		squareIntersectTestcase{ // input leaves just one possible
			&square{index: 4, pvals: intset{3, 4, 6, 9}},
			intset{6},
			intset{6},
			0, nil,
		},
		squareIntersectTestcase{ // reduce to already bound
			&square{index: 105, pvals: intset{3, 4, 6, 9, 12, 13, 15, 16},
				bval: 3, bsrc: helperBsrc(105)},
			intset{3},
			intset{3},
			3, helperBsrc(105),
		},
		// same first four tests using larger squares
		squareIntersectTestcase{ // input larger than range
			newEmptySquare(101, 16, nil),
			intset{0, 3, 4, 6, 9, 12, 13, 15, 16, 17},
			intset{3, 4, 6, 9, 12, 13, 15, 16},
			0, nil,
		},
		squareIntersectTestcase{ // input subset of empty square
			newEmptySquare(102, 16, nil),
			intset{1, 2, 5, 7, 8, 10, 11, 14},
			intset{1, 2, 5, 7, 8, 10, 11, 14},
			0, nil,
		},
		squareIntersectTestcase{ // input equal to range
			&square{index: 103, pvals: intset{3, 4, 6, 9, 12, 13, 15, 16}},
			intset{3, 4, 6, 9, 12, 13, 15, 16},
			intset{3, 4, 6, 9, 12, 13, 15, 16},
			0, nil,
		},
		squareIntersectTestcase{ // input leaves just one possible
			&square{index: 104, pvals: intset{3, 4, 6, 9, 12, 13, 15, 16}},
			intset{16},
			intset{16},
			0, nil,
		},
	}
	for _, tc := range testcases {
		// dup input square to preserve test case for error messages
		input := helperDupSquare(tc.square)
		e := input.intersect(tc.tointersect)
		if e != nil {
			t.Fatalf("Removing %v from %v produced error %v", tc.tointersect, tc.square, e)
		}
		if !reflect.DeepEqual(input.pvals, tc.remaining) {
			t.Errorf("Removing %v from %v left %v not %v",
				tc.tointersect, *tc.square, input.pvals, tc.remaining)
		}
		if input.bval != tc.bval || !reflect.DeepEqual(input.bsrc, tc.bsrc) {
			t.Errorf("Removing %v from %v left binding %v: %v not %v: %v",
				tc.tointersect, *tc.square, input.bval, input.bsrc, tc.bval, tc.bsrc)
		}
	}
}

/*

Groups

*/

type binding struct {
	index int
	bval  int
	bsrc  []GroupID
}

type newGroupErrcase struct {
	gd   *groupDescriptor
	ss   []*square
	cond ErrorCondition
}

type newGroupTestcase struct {
	name    string
	sidelen int
	gtype   string
	gindex  int
	vals    []int
	where   []int
	need    intset
	empty   intset
}

func TestNewGroup(t *testing.T) {
	// errcases have to be made with artificial square arrays,
	// rather than properly constructed groups, because they
	// can't occur unless the client does something hinky
	errcases := []newGroupErrcase{
		newGroupErrcase{ // duplicate assignment
			&groupDescriptor{1, GroupID{"error", 1}, []int{1, 2, 3, 4}},
			[]*square{
				nil,
				newFilledSquare(1, 4, 1, nil),
				newFilledSquare(2, 4, 2, nil),
				newFilledSquare(3, 4, 1, nil),
				newEmptySquare(4, 4, nil),
			},
			DuplicateGroupValuesCondition,
		},
		newGroupErrcase{ // prevent needed removal via no candidates
			&groupDescriptor{1, GroupID{"error", 2}, []int{1, 2, 3, 4}},
			[]*square{
				nil,
				newFilledSquare(1, 4, 1, nil),
				newFilledSquare(2, 4, 2, nil),
				&square{index: 3, pvals: intset{1, 2}},
				newEmptySquare(4, 4, nil),
			},
			NotInSetCondition,
		},
		newGroupErrcase{ // prevent needed removal via binding
			&groupDescriptor{1, GroupID{"error", 3}, []int{1, 2, 3, 4}},
			[]*square{
				nil,
				newFilledSquare(1, 4, 1, nil),
				newFilledSquare(2, 4, 2, nil),
				newEmptySquare(3, 4, nil),
				helperBindSquare(newEmptySquare(4, 4, nil), 2, helperGID(3)),
			},
			NoGroupValueCondition,
		},
	}
	for _, ec := range errcases {
		_, errs := newGroup(ec.gd, ec.ss)
		if len(errs) == 0 {
			t.Errorf("newGroup %v produced no errors", ec.gd.id)
		}
	}

	// do testing with size 4 groups, since they are simpler
	testcases := []newGroupTestcase{
		newGroupTestcase{ // first 2 of 4 assigned, no other info
			"test 1", 4, GtypeRow, 1,
			[]int{1, 2, 0, 0},
			[]int{0, 1, 2, 0, 0}, intset{3, 4}, intset{3, 4},
		},
		newGroupTestcase{ // first 3 of 4 assigned, forces last via removal
			"test 2", 4, GtypeRow, 1,
			[]int{1, 2, 3, 0},
			[]int{0, 1, 2, 3, 0}, intset{4}, intset{4},
		},
		newGroupTestcase{ // last 2 of 4 assigned, no other info
			"test 3", 4, GtypeRow, 1,
			[]int{0, 0, 3, 4},
			[]int{0, 0, 0, 3, 4}, intset{1, 2}, intset{1, 2},
		},
		newGroupTestcase{ // 2 of 4 assigned out of order, with a gap
			"test 4", 4, GtypeRow, 1,
			[]int{0, 4, 0, 3},
			[]int{0, 0, 0, 4, 2}, intset{1, 2}, intset{1, 3},
		},
		newGroupTestcase{ // 1 of 4 assigned out of order
			"test 5", 4, GtypeRow, 1,
			[]int{0, 0, 0, 3},
			[]int{0, 0, 0, 4, 0}, intset{1, 2, 4}, intset{1, 2, 3},
		},
		newGroupTestcase{ // 1 of 4 assigned, the other three reduced
			"test 6", 4, GtypeRow, 1,
			[]int{-2, -1, -4, 3},
			[]int{0, 0, 0, 4, 0}, intset{1, 2, 4}, intset{1, 2, 3},
		},
	}
	for _, tc := range testcases {
		gd := helperSquareGroupDescriptor(tc.sidelen, tc.gtype, tc.gindex)
		ss := helperMakeGroupSquares(gd, tc.vals...)
		eg := &group{gd, tc.where, tc.need, tc.empty}
		g, err := newGroup(gd, ss)
		if err != nil {
			t.Fatalf("newGroup %v produced error %v", tc.name, err)
		}
		if !reflect.DeepEqual(g, eg) {
			t.Errorf("newGroup %v produced %v (expected %v)", tc.name, g, eg)
		}
	}
}

type groupAnalyzeErrcase struct {
	gd   *groupDescriptor
	ss   []*square
	cond ErrorCondition
}

type groupAnalyzeTestcase struct {
	name    string
	sidelen int
	gtype   string
	gindex  int
	vals    []int
	where   []int
	need    intset
	empty   intset
	bs      []binding
}

func TestGroupAnalyze(t *testing.T) {
	// do testing with size 4 groups, since they are simpler

	// NOTE: be sure to order the value higest to lowest, to make
	// sure they are not pre-sorted

	// errcases have to be made with artificial square arrays,
	// rather than properly constructed groups, because they
	// can't occur unless the client does something hinky
	errcases := []groupAnalyzeErrcase{
		groupAnalyzeErrcase{ // restrict candidates so no possible square
			&groupDescriptor{1, GroupID{"error", 1}, []int{1, 2, 3, 4}},
			[]*square{
				nil,
				newFilledSquare(2, 4, 1, nil),
				newFilledSquare(1, 4, 2, nil),
				&square{index: 3, pvals: intset{1, 3}},
				&square{index: 4, pvals: intset{2, 3}},
			},
			NoGroupValueCondition,
		},
		groupAnalyzeErrcase{ // prevent needed binding
			&groupDescriptor{1, GroupID{"error", 2}, []int{1, 2, 3, 4}},
			[]*square{
				nil,
				newFilledSquare(2, 4, 1, nil),
				newFilledSquare(1, 4, 2, nil),
				&square{index: 3, pvals: intset{1, 3}},
				&square{index: 4, pvals: intset{3, 4}, bval: 3, bsrc: helperBsrc(2)},
			},
			NoGroupValueCondition,
		},
	}
	for _, ec := range errcases {
		g, errs := newGroup(ec.gd, ec.ss)
		if len(errs) != 0 {
			t.Fatalf("Invalid testcase %v: newGroup errors %v", ec.gd.id, errs)
		}
		errs = g.analyze(ec.ss)
		if len(errs) == 0 {
			t.Errorf("Error case %v: (group).analyze gave no errors", ec.gd.id)
		}
	}

	testcases := []groupAnalyzeTestcase{
		groupAnalyzeTestcase{ // first 2 of 4 assigned, no other info
			"test 1", 4, GtypeRow, 1,
			[]int{2, 1, 0, 0},
			[]int{0, 2, 1, 0, 0}, intset{3, 4}, intset{3, 4},
			nil,
		},
		groupAnalyzeTestcase{ // first 3 of 4 assigned, forces last
			"test 2", 4, GtypeRow, 1,
			[]int{3, 2, 1, 0},
			[]int{0, 3, 2, 1, 0}, intset{}, intset{},
			nil,
		},
		groupAnalyzeTestcase{ // last 2 of 4 assigned, no other info
			"test 3", 4, GtypeRow, 1,
			[]int{0, 0, 4, 3},
			[]int{0, 0, 0, 4, 3}, intset{1, 2}, intset{1, 2},
			nil,
		},
		groupAnalyzeTestcase{ // 2 of 4 assigned, with a gap
			"test 4", 4, GtypeRow, 1,
			[]int{0, 3, 0, 1},
			[]int{0, 4, 0, 2, 0}, intset{2, 4}, intset{1, 3},
			nil,
		},
		groupAnalyzeTestcase{ // 1 of 4 assigned
			"test 5", 4, GtypeRow, 1,
			[]int{0, 0, 0, 3},
			[]int{0, 0, 0, 4, 0}, intset{1, 2, 4}, intset{1, 2, 3},
			nil,
		},
		groupAnalyzeTestcase{ // 1 of 4 assigned, the other three reduced
			"test 6", 4, GtypeRow, 1,
			[]int{-2, -1, -4, 3},
			[]int{0, 0, 0, 4, 0}, intset{1, 2, 4}, intset{1, 2, 3},
			nil,
		},
		groupAnalyzeTestcase{ // 2 of 4 assigned, reduction forces binding
			"test 7", 4, GtypeRow, 1,
			[]int{0, 4, -1, 2},
			[]int{0, 0, 4, 0, 2}, intset{}, intset{},
			[]binding{binding{1, 1, helperBsrc(0 + 1)}},
		},
		groupAnalyzeTestcase{ // like the prior one, but a tile instead.
			"test 8", 4, GtypeTile, 2,
			[]int{0, 4, -1, 2},
			[]int{0, 0, 8, 0, 4}, intset{}, intset{},
			[]binding{binding{3, 1, helperBsrc(8 + 2)}},
		},
	}
	for _, tc := range testcases {
		gd := helperSquareGroupDescriptor(tc.sidelen, tc.gtype, tc.gindex)
		ss := helperMakeGroupSquares(gd, tc.vals...)
		eg := &group{gd, tc.where, tc.need, tc.empty}
		g, err := newGroup(gd, ss)
		if err != nil {
			t.Errorf("invalid testcase %v: newGroup error %v", tc.name, err)
		}
		err = g.analyze(ss)
		if err != nil {
			t.Fatalf("groupAnalyze %v produced error %v", tc.name, err)
		}
		if !reflect.DeepEqual(g, eg) {
			t.Errorf("groupAnalyze %v produced %v (expected %v)", tc.name, g, eg)
		}
		bi, bcount := 0, len(tc.bs)
		for _, si := range gd.indices {
			s := ss[si]
			if s.aval != 0 {
				continue // ignore assigned squares
			}
			switch {
			case bi >= bcount || si < tc.bs[bi].index:
				if s.bval != 0 || s.bsrc != nil {
					t.Errorf("groupAnalyze %v: square %d binds %d %v (expected none)",
						tc.name, si, s.bval, s.bsrc)
				}
			case si == tc.bs[bi].index:
				b := tc.bs[bi]
				if s.bval != b.bval || !reflect.DeepEqual(s.bsrc, b.bsrc) {
					t.Errorf("groupAnalyze %v: square %d binds %d %v (expected %d %v)",
						tc.name, si, s.bval, s.bsrc, b.bval, b.bsrc)
				}
				bi++
			default:
				t.Fatalf("invalid test: binding out of order or for non-square: %v", tc.bs[bi])
			}
		}
	}
}

type groupAssignErrcase struct {
	gd   *groupDescriptor
	ss   []*square
	ai   int
	av   int
	cond ErrorCondition
}

type groupAssignTestcase struct {
	name    string
	sidelen int
	gtype   string
	gindex  int
	vals    []int
	ai      int
	av      int
	bs      []binding
}

func TestGroupAssign(t *testing.T) {
	// errcases involving unsatisfiable squares
	errcases := []groupAssignErrcase{
		// there's no way to get this error with an actual square
		// assignment, because it would fail due to group removal
		// of possible values.  So we simulate the assignment by
		// hand.
		groupAssignErrcase{
			&groupDescriptor{1, GroupID{"error", 1}, []int{1, 2, 3, 4}},
			[]*square{
				nil,
				newFilledSquare(1, 4, 1, nil),
				newFilledSquare(2, 4, 2, nil),
				newEmptySquare(3, 4, nil),
				newEmptySquare(4, 4, nil),
			},
			3, 1, DuplicateGroupValuesCondition,
		},
		groupAssignErrcase{ // restriction means no possible values will be left
			&groupDescriptor{1, GroupID{"error", 2}, []int{1, 2, 3, 4}},
			[]*square{
				nil,
				newFilledSquare(1, 4, 1, nil),
				newFilledSquare(2, 4, 2, nil),
				newEmptySquare(3, 4, nil),
				helperRestrictedSquare(4, 4, 4),
			},
			3, 3, NoPossibleValuesCondition,
		},
		groupAssignErrcase{ // binding means removal of assigned value will fail
			&groupDescriptor{1, GroupID{"error", 3}, []int{1, 2, 3, 4}},
			[]*square{
				nil,
				newFilledSquare(1, 4, 1, nil),
				newFilledSquare(2, 4, 2, nil),
				newEmptySquare(3, 4, nil),
				helperBindSquare(newEmptySquare(4, 4, nil), 4, helperGID(2)),
			},
			3, 4, NoGroupValueCondition,
		},
	}
	for _, ec := range errcases {
		g, errs := newGroup(ec.gd, ec.ss)
		if len(errs) != 0 {
			t.Fatalf("Invalid case %v: newGroup: %v", ec.gd.id, errs)
		}
		errs = g.analyze(ec.ss)
		if len(errs) != 0 {
			t.Fatalf("Invalid case %v: (group).analyze: %v", ec.gd.id, errs)
		}
		ec.ss[ec.ai].aval = ec.av // simulate the assignment
		errs = g.assign(ec.ss, ec.ai)
		if len(errs) == 0 {
			t.Errorf("groupAssign case %v didn't fail, produced %+v", ec.gd.id, *g)
		} else {
			if !helperCheckCondition(ec.cond, errs) {
				t.Errorf("groupAssign case %v produced wrong errors: %+v", ec.gd.id, errs)
			}
		}
	}

	// do testing with size 4 groups, since they are simpler
	testcases := []groupAssignTestcase{
		groupAssignTestcase{ // first 2 of 4 assigned, then last
			"test 2", 4, GtypeRow, 1,
			[]int{1, 2, 0, 0},
			4, 4,
			nil,
		},
		groupAssignTestcase{ // like the prior one, last 2 assigned, then first
			"test 3", 4, GtypeRow, 1,
			[]int{0, 0, 3, 4},
			1, 1,
			nil,
		},
		groupAssignTestcase{ // 3 reduced, 1 assigned, then another assigned
			"test 4", 4, GtypeRow, 1,
			[]int{-2, -1, -4, 3},
			3, 2,
			[]binding{binding{1, 1, helperBsrc(0 + 1)}},
		},
		groupAssignTestcase{ // like the prior one, but a tile instead.
			"test 5", 4, GtypeTile, 2,
			[]int{-2, -1, -4, 3},
			7, 2,
			[]binding{binding{3, 1, helperBsrc(8 + 2)}},
		},
	}
	for _, tc := range testcases {
		gd := helperSquareGroupDescriptor(tc.sidelen, tc.gtype, tc.gindex)
		ss := helperMakeGroupSquares(gd, tc.vals...)
		g, errs := newGroup(gd, ss)
		if len(errs) != 0 {
			t.Fatalf("groupAssign invalid case %s: newGroup: %v", tc.name, errs)
		}
		errs = g.analyze(ss)
		if len(errs) != 0 {
			t.Fatalf("groupAssign invalid case %s: (group).analyze: %v", tc.name, errs)
		}
		e := ss[tc.ai].assign(tc.av)
		if e != nil {
			t.Fatalf("groupAssign invalid case %s: (square).assign: %v", tc.name, errs)
		}
		errs = g.assign(ss, tc.ai)
		if len(errs) != 0 {
			t.Fatalf("groupAssign case %v assign produced error %v", tc.name, errs)
		}
		errs = g.analyze(ss)
		if len(errs) != 0 {
			t.Fatalf("groupAssign case %v analyze produced error %v", tc.name, errs)
		}
		bi, bcount := 0, len(tc.bs)
		for _, si := range gd.indices {
			s := ss[si]
			if si == tc.ai {
				// make sure group noticed the assignment
				_, needed := g.need.find(tc.av)
				_, free := g.free.find(tc.ai)
				if g.where[tc.av] != si || needed || free {
					t.Errorf("groupAssign case %v: assign(%d, %d) didn't take: %v",
						tc.name, tc.ai, tc.av, g)
				}
			}
			if s.aval != 0 {
				continue // ignore other assigned squares
			}
			switch {
			case bi >= bcount || si < tc.bs[bi].index:
				if s.bval != 0 || s.bsrc != nil {
					t.Errorf("groupAssign case %v: square %d binds %d %v (expected none)",
						tc.name, si, s.bval, s.bsrc)
				}
			case si == tc.bs[bi].index:
				b := tc.bs[bi]
				if s.bval != b.bval || !reflect.DeepEqual(s.bsrc, b.bsrc) {
					t.Errorf("groupAssign case %v: square %d binds %d %v (expected %d %v)",
						tc.name, si, s.bval, s.bsrc, b.bval, b.bsrc)
				}
				bi++
			default:
				t.Fatalf("invalid test: binding out of order or for non-square: %v", tc.bs[bi])
			}
		}
	}
}

/*

Puzzle Construction

*/

type newSudokuErrcase struct {
	name      string
	vals      []int
	canCreate bool
	cond      ErrorCondition
}

type newSudokuTestcase struct {
	name string
	vals []int
	ss   []*square
	gs   []*group
}

func TestNewSudoku(t *testing.T) {
	errcases := []newSudokuErrcase{
		newSudokuErrcase{
			"err 1",
			conflicting4Puzzle1,
			true, DuplicateGroupValuesCondition,
		},
		newSudokuErrcase{
			"err 2",
			conflicting4Puzzle2,
			true, DuplicateGroupValuesCondition,
		},
		newSudokuErrcase{
			"err 3",
			conflicting4Puzzle3,
			true, DuplicateGroupValuesCondition,
		},
		newSudokuErrcase{
			"err 4",
			conflicting4Puzzle4,
			true, DuplicateGroupValuesCondition,
		},
		newSudokuErrcase{
			"err 5",
			unsatisfiable4Puzzle,
			true, NoGroupValueCondition,
		},
		newSudokuErrcase{
			"err 6",
			[]int{0, 0, 0, 0},
			false, TooSmallCondition,
		},
		newSudokuErrcase{
			"err 7",
			[]int{0, 374, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			false, TooLargeCondition,
		},
	}
	for _, ec := range errcases {
		p, e := helperNewSudokuPuzzle(ec.vals)
		if e != nil {
			if ec.canCreate {
				t.Errorf("newSudoku case %s didn't create puzzle: %v", ec.name, e)
			} else {
				if e.(Error).Condition != ec.cond {
					t.Errorf("newSudoku case %s wrong error: %v", ec.name, e)
				}
			}
		} else if len(p.errors) > 0 {
			if !helperCheckCondition(ec.cond, p.errors) {
				t.Errorf("newSudoku case %s: wrong errors: %v", ec.name, p.errors)
			}
		} else {
			t.Errorf("newSudoku case %s produced no errors: %v", ec.name, p.State())
		}
	}

	testcases := []newSudokuTestcase{
		newSudokuTestcase{
			"test 1",
			rotation4Puzzle1PartialValues,
			rotation4Puzzle1PartialSquares,
			rotation4Puzzle1PartialGroups,
		},
		newSudokuTestcase{
			"test 2",
			rotation4Puzzle2PartialValues,
			rotation4Puzzle2PartialSquares,
			rotation4Puzzle2PartialGroups,
		},
		newSudokuTestcase{
			"test 3",
			bound4PuzzleValues,
			bound4PuzzleSquares,
			nil,
		},
	}
	for _, tc := range testcases {
		p, e := helperNewSudokuPuzzle(tc.vals)
		if e != nil {
			t.Fatalf("newSudoku case %s failed: %s", tc.name, e.Error())
		}
		if len(p.squares) != len(tc.ss) {
			t.Fatalf("newSudoku case %s: gave %d squares, expected %d.",
				tc.name, len(p.squares), len(tc.ss))
		}
		if !helperSquaresEqual(p.squares, tc.ss) {
			t.Errorf("newSudoku case %s unexpected squares:", tc.name)
			for i := range p.squares {
				if !helperSquareEqual(p.squares[i], tc.ss[i]) {
					t.Errorf("%s Square %d: is %+v, expected %+v",
						tc.name, p.squares[i].index, *p.squares[i], *tc.ss[i])
				}
			}
		}
		if tc.gs != nil {
			if len(p.groups) != len(tc.gs) {
				t.Fatalf("newSudoku case %s: gave %d groups, expected %d.",
					tc.name, len(p.groups), len(tc.gs))
			}
			if !reflect.DeepEqual(p.groups, tc.gs) {
				t.Errorf("newSudoku case %s unexpected groups:", tc.name)
				for i := range p.groups {
					if !reflect.DeepEqual(p.groups[i], tc.gs[i]) {
						t.Errorf("%s Group %v: is %+v, expected %+v",
							tc.name, p.groups[i].desc.id, *p.groups[i], *tc.gs[i])
					}
				}
			}
		}
	}
}

type newEmptySudokuTestcase struct {
	name    string
	sidelen int
	ss      []*square
	gs      []*group
}

func TestNewEmptySudoku(t *testing.T) {
	testcases := []newEmptySudokuTestcase{
		newEmptySudokuTestcase{
			"test 1",
			4,
			empty4PuzzleSquares,
			empty4PuzzleGroups,
		},
	}
	for _, tc := range testcases {
		p, e := helperNewEmptySudokuPuzzle(tc.sidelen)
		if e != nil {
			t.Fatalf("newEmptySudoku case %s failed: %s", tc.name, e.Error())
		}
		if len(p.squares) != len(tc.ss) {
			t.Fatalf("newEmptySudoku case %s: gave %d squares, expected %d.",
				tc.name, len(p.squares), len(tc.ss))
		}
		if !helperSquaresEqual(p.squares, tc.ss) {
			t.Errorf("newEmptySudoku case %s unexpected squares:", tc.name)
			for i := range p.squares {
				if !helperSquareEqual(p.squares[i], tc.ss[i]) {
					t.Errorf("%s Square %d: is %+v, expected %+v",
						tc.name, p.squares[i].index, *p.squares[i], *tc.ss[i])
				}
			}
		}
		if tc.gs != nil {
			if len(p.groups) != len(tc.gs) {
				t.Fatalf("newEmptySudoku case %s: gave %d groups, expected %d.",
					tc.name, len(p.groups), len(tc.gs))
			}
			if !reflect.DeepEqual(p.groups, tc.gs) {
				t.Errorf("newEmptySudoku case %s unexpected groups:", tc.name)
				for i := range p.groups {
					if !reflect.DeepEqual(p.groups[i], tc.gs[i]) {
						t.Errorf("%s Group %v: is %+v, expected %+v",
							tc.name, p.groups[i].desc.id, *p.groups[i], *tc.gs[i])
					}
				}
			}
		}
	}
}

/*

Puzzle Operations

*/

type stateTestcase struct {
	name   string
	vals   []int
	estate State
}

func TestState(t *testing.T) {
	testcases := []stateTestcase{
		stateTestcase{
			"test 1",
			rotation4Puzzle1PartialAssign1Values,
			State{SudokuGeometryCode, 4, rotation4Puzzle1PartialAssign1Values, nil},
		},
		stateTestcase{
			"test 2",
			empty4PuzzleValues,
			State{SudokuGeometryCode, 4, empty4PuzzleValues, nil},
		},
		stateTestcase{
			"test 3",
			rotation4Puzzle1Complete1,
			State{SudokuGeometryCode, 4, rotation4Puzzle1Complete1, nil},
		},
		stateTestcase{
			"test 4",
			conflicting4Puzzle1,
			State{SudokuGeometryCode, 4, conflicting4Puzzle1, []Error{
				Error{
					Scope:     GroupScope,
					Structure: ScopeStructure,
					Condition: DuplicateGroupValuesCondition,
					Values:    ErrorData{GroupID{GtypeTile, 1}, 1},
					Message:   "Problem in tile 1: Multiple squares have value 1",
				},
				Error{
					Scope:     GroupScope,
					Structure: ScopeStructure,
					Condition: NoGroupValueCondition,
					Values:    ErrorData{GroupID{GtypeTile, 2}, 1},
					Message:   "Problem in tile 2: No square can contain 1",
				},
				Error{
					Scope:     GroupScope,
					Structure: ScopeStructure,
					Condition: NoGroupValueCondition,
					Values:    ErrorData{GroupID{GtypeTile, 3}, 1},
					Message:   "Problem in tile 3: No square can contain 1",
				},
			}},
		},
	}
	for _, tc := range testcases {
		p, e := helperNewSudokuPuzzle(tc.vals)
		if e != nil {
			t.Fatalf("State case %s creation failed: %s", tc.name, e.Error())
		}
		state := p.State()
		if !reflect.DeepEqual(state, tc.estate) {
			t.Errorf("State case %s returned %v, expected %v",
				tc.name, state, tc.estate)
		}
	}
}

type assignInternalTestcase struct {
	name   string
	ai, av int
	ss     []*square
	pss    []*square
	gs     []*group
}

func TestInternalAssign(t *testing.T) {
	testcases := []assignInternalTestcase{
		assignInternalTestcase{
			"test 1", 13, 2,
			rotation4Puzzle1PartialAssign1Squares,
			rotation4Puzzle1PartialSquares,
			rotation4Puzzle1PartialAssign1Groups,
		},
		assignInternalTestcase{
			"test 2", 10, 4,
			rotation4Puzzle1PartialAssign2Squares,
			rotation4Puzzle1PartialAssign1Squares,
			rotation4Puzzle1PartialAssign2Groups,
		},
		assignInternalTestcase{
			"test 3", 15, 4,
			rotation4Puzzle1PartialAssign3Squares,
			rotation4Puzzle1PartialAssign2Squares,
			rotation4Puzzle1PartialAssign3Groups,
		},
	}
	// we apply the testcases in sequence to a base setup
	p, e := helperNewSudokuPuzzle(rotation4Puzzle1PartialValues)
	if e != nil {
		t.Fatalf("Creation of rotation4Puzzle1 failed: %s", e.Error())
	}
	for _, tc := range testcases {
		is := p.assign(tc.ai, tc.av)
		if len(p.errors) != 0 {
			t.Fatalf("invalid assign %s: assign(%d, %d) failed: %s",
				tc.name, tc.ai, tc.av, e.Error())
		}
		if !reflect.DeepEqual(is, helperDiffSquares(tc.pss, tc.ss)) {
			t.Errorf("%s assign(%d, %d) logged %v, expected %v",
				tc.name, tc.ai, tc.av, is, helperDiffSquares(tc.pss, tc.ss))
		}
		if !helperSquaresEqual(p.squares, tc.ss) {
			t.Errorf("%s assign(%d, %d) unexpected squares:", tc.name, tc.ai, tc.av)
			for i := range p.squares {
				if !helperSquareEqual(p.squares[i], tc.ss[i]) {
					t.Errorf("%s Square %d: is %+v, expected %+v",
						tc.name, p.squares[i].index, *p.squares[i], *tc.ss[i])
				}
			}
		}
		if tc.gs != nil {
			if len(p.groups) != len(tc.gs) {
				t.Fatalf("%s assign(%d, %d): gave %d groups, expected %d.",
					tc.name, tc.ai, tc.av, len(p.groups), len(tc.gs))
			}
			if !reflect.DeepEqual(p.groups, tc.gs) {
				t.Errorf("%s assign(%d, %d) unexpected groups:", tc.name, tc.ai, tc.av)
				for i := range p.groups {
					if !reflect.DeepEqual(p.groups[i], tc.gs[i]) {
						t.Errorf("%s Group %v: is %+v, expected %+v",
							tc.name, p.groups[i].desc.id, *p.groups[i], *tc.gs[i])
					}
				}
			}
		}
	}
}

type assignInternalBenchcase struct {
	name   string
	ai, av int
}

func BenchmarkInternalAssign(b *testing.B) {
	benchcases := []assignInternalBenchcase{
		assignInternalBenchcase{"test 1", 13, 2},
		assignInternalBenchcase{"test 2", 10, 4},
		assignInternalBenchcase{"test 3", 15, 4},
	}
	// we apply the benchcases in sequence to a base setup
	master, e := helperNewSudokuPuzzle(rotation4Puzzle1PartialValues)
	if e != nil {
		b.Fatalf("Creation of rotation4Puzzle1 failed: %s", e.Error())
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := master.copy()
		for _, bc := range benchcases {
			p.assign(bc.ai, bc.av)
			if len(p.errors) != 0 {
				b.Fatalf("invalid assign %s: assign(%d, %d) failed: %s",
					bc.name, bc.ai, bc.av, e.Error())
			}
		}
	}
}

type assignExternalTestcase struct {
	name   string
	ai, av int
	SS     []Square
}

// just need to test the outputs and errors, not the logic
func TestExternalAssign(t *testing.T) {
	// multiple boundary cases
	pi := &puzzle{errors: []Error{{Message: "test error"}}}
	_, e := pi.Assign(Choice{1, 1})
	if e == nil {
		t.Errorf("Assign to puzzle with one issue didn't err")
	}
	if e.(Error).Scope != ArgumentScope {
		t.Errorf("Assign to puzzle with one issue returned wrong error: %v", e.Error())
	}
	pi, e = helperNewSudokuPuzzle(rotation4Puzzle1PartialValues)
	if e != nil {
		t.Fatalf("Creation of valid 4 puzzle produced error: %v", e)
	}
	_, e = pi.Assign(Choice{0, 3})
	if e == nil || e.(Error).Condition != TooSmallCondition {
		t.Errorf("Assignment of index too small produced incorrect error: %v", e)
	}
	_, e = pi.Assign(Choice{205, 3})
	if e == nil || e.(Error).Condition != TooLargeCondition {
		t.Errorf("Assignment of index too large produced incorrect error: %v", e)
	}
	_, e = pi.Assign(Choice{3, 0})
	if e == nil || e.(Error).Condition != TooSmallCondition {
		t.Errorf("Assignment of value too small produced incorrect error: %v", e)
	}
	_, e = pi.Assign(Choice{3, 205})
	if e == nil || e.(Error).Condition != TooLargeCondition {
		t.Errorf("Assignment of value too large produced incorrect error: %v", e)
	}
	_, e = pi.Assign(Choice{1, 1})
	if e == nil || e.(Error).Condition != DuplicateAssignmentCondition {
		t.Errorf("Re-assignment of same value produced incorrect error: %v", e)
	}

	testcases := []assignExternalTestcase{
		assignExternalTestcase{
			"test 1", 13, 2,
			rotation4Puzzle1PartialAssign1CapitalSquares,
		},
		assignExternalTestcase{
			"test 2", 10, 4,
			rotation4Puzzle1PartialAssign2CapitalSquares,
		},
		assignExternalTestcase{
			"test 3", 15, 4,
			rotation4Puzzle1PartialAssign3CapitalSquares,
		},
	}
	// we apply the testcases in sequence to a base setup
	p, e := helperNewSudokuPuzzle(rotation4Puzzle1PartialValues)
	if e != nil {
		t.Fatalf("Creation of rotation4Puzzle1 failed: %s", e.Error())
	}
	for _, tc := range testcases {
		_, e := p.Assign(Choice{tc.ai, tc.av})
		if e != nil {
			t.Fatalf("%s: Assign(Choice{%d, %d}) failed: %s",
				tc.name, tc.ai, tc.av, e.Error())
		}
		for i, S := range p.Squares() {
			if !reflect.DeepEqual(S, tc.SS[i]) {
				t.Errorf("%s Assign(Choice{%d, %d}) Square %d was %v, expected %v",
					tc.name, tc.ai, tc.av, S.Index, S, tc.SS[i])
			}
		}
	}
}

type squaresTestcase struct {
	name   string
	ai, av int
	ss     []Square
}

// depends on assignment so follows it
// also tests internal allSquares
func TestSquares(t *testing.T) {
	testcases := []squaresTestcase{
		squaresTestcase{
			"test 1", 13, 2,
			rotation4Puzzle1PartialAssign1CapitalSquares,
		},
		squaresTestcase{
			"test 2", 10, 4,
			rotation4Puzzle1PartialAssign2CapitalSquares,
		},
		squaresTestcase{
			"test 3", 15, 4,
			rotation4Puzzle1PartialAssign3CapitalSquares,
		},
	}
	// we apply the testcases in sequence to a base setup
	p, e := helperNewSudokuPuzzle(rotation4Puzzle1PartialValues)
	if e != nil {
		t.Fatalf("Creation of rotation4Puzzle1 failed: %s", e.Error())
	}
	for _, tc := range testcases {
		_, e := p.Assign(Choice{tc.ai, tc.av})
		if e != nil {
			t.Fatalf("invalid Squares %s: Assign(&Choice{%d, %d}) failed: %s",
				tc.name, tc.ai, tc.av, e.Error())
		}
		ss := p.Squares()
		if len(ss) != len(tc.ss) {
			t.Fatalf("Squares %s: gave %d squares, expected %d.",
				tc.name, len(ss), len(tc.ss))
		}
		if !reflect.DeepEqual(ss, tc.ss) {
			t.Errorf("Squares case %s unexpected squares:", tc.name)
			for i := range ss {
				if !reflect.DeepEqual(ss[i], tc.ss[i]) {
					t.Errorf("%s Square %d: is %+v, expected %+v",
						tc.name, ss[i].Index, ss[i], tc.ss[i])
				}
			}
		}
	}
}

type puzzleCopyTestcase struct {
	name   string
	vals   []int
	ai, av int
}

func TestPuzzleInternalCopy(t *testing.T) {
	ep := (*puzzle)(nil).copy()
	if ep != nil {
		t.Errorf("Copy of nil puzzle returned non-nil: %v", ep)
	}

	testcases := []puzzleCopyTestcase{
		puzzleCopyTestcase{
			"test 1",
			rotation4Puzzle1PartialValues,
			2, 2,
		},
		puzzleCopyTestcase{
			"test 2",
			rotation4Puzzle2PartialValues,
			2, 4,
		},
		puzzleCopyTestcase{
			"test 3",
			rotation4Puzzle1Complete1,
			0, 0,
		},
		puzzleCopyTestcase{
			"test 4",
			rotation4Puzzle1Complete2,
			0, 0,
		},
		puzzleCopyTestcase{
			"test 5",
			bound4PuzzleValues,
			4, 4,
		},
		puzzleCopyTestcase{
			"test 6",
			conflicting4Puzzle1,
			0, 0,
		},
		puzzleCopyTestcase{
			"test 7",
			conflicting4Puzzle2,
			0, 0,
		},
	}
	for _, tc := range testcases {
		p, e := helperNewSudokuPuzzle(tc.vals)
		if e != nil {
			t.Fatalf("puzzleCopy %s failed to make puzzle: %v", tc.name, e)
		}
		c := p.copy()
		if reflect.ValueOf(c.logger).Pointer() == reflect.ValueOf(p.logger).Pointer() {
			t.Errorf("puzzleCopy %s: copy logger shared with original", tc.name)
		}
		// although the loggers are different instances, they have the same state,
		// so the puzzles will compare perfectly with DeepEqual
		if !reflect.DeepEqual(p, c) {
			t.Errorf("puzzleCopy %s: copy doesn't match original", tc.name)
		}
		// make sure copys and originals are fully separate and behave the same
		if tc.ai != 0 {
			_, e = c.Assign(Choice{tc.ai, tc.av})
			if e != nil {
				t.Fatalf("puzzleCopy %s Assign failed: %v", tc.name, e)
			}
			if reflect.DeepEqual(p, c) {
				t.Errorf("puzzleCopy %s copy.Assign altered original!", tc.name)
			}
			_, e = p.Assign(Choice{tc.ai, tc.av})
			if e != nil {
				t.Fatalf("puzzleCopy %s original.Assign failed: %v", tc.name, e)
			}
			if !reflect.DeepEqual(p, c) {
				t.Errorf("puzzleCopy %s copy/original Assigns had different effect!", tc.name)
			}
		}
	}
}

func TestPuzzleExternalCopy(t *testing.T) {
	in, e := helperNewSudokuPuzzle(rotation4Puzzle1PartialValues)
	if e != nil {
		t.Fatalf("Creation of rotation4Puzzle1 failed: %s", e.Error())
	}
	In := Puzzle(in)
	Copy := In.Copy()
	copy := Copy.(*puzzle)
	if reflect.ValueOf(copy).Pointer() == reflect.ValueOf(in).Pointer() {
		t.Errorf("Puzzle.Copy wraps same puzzle as original")
	}
	if !reflect.DeepEqual(In, Copy) {
		t.Errorf("Puzzle.Copy differs from original")
	}
}

func BenchmarkCopy(b *testing.B) {
	master, e := helperNewSudokuPuzzle(rotation4Puzzle1PartialValues)
	if e != nil {
		b.Fatalf("Creation of rotation4Puzzle1 failed: %s", e.Error())
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := master.copy()
		if p == nil {
			b.Fatalf("Copy failed, returned nil")
		}
	}
}

/*

A few end-to-end tests with puzzle construction and sequences of assignments

*/

type assignment struct {
	ai, av int
}

type assignseq struct {
	init  []int        // initial values for 4 puzzle, nil means empty puzzle
	setup []assignment // assignments that should succeed
	final assignment   // final assignment
}

func TestEndToEndPuzzleAssignment(t *testing.T) {
	var p *puzzle

	tryassign := func(i, v int) error {
		_, e := p.Assign(Choice{i, v})
		if e == nil {
			t.Logf("Assign(Choice{%d, %d}) succeeded:\n%v", i, v, p)
		} else {
			t.Logf("Assign(Choice{%d, %d}) failed: %s\n%v", i, v, e.Error(), p)
		}
		return e
	}

	tests := []assignseq{
		assignseq{
			nil,
			[]assignment{
				assignment{1, 1},
				assignment{14, 4},
				assignment{11, 4},
			},
			assignment{8, 4},
		},
		assignseq{
			nil,
			[]assignment{
				assignment{14, 4},
				assignment{11, 4},
				assignment{8, 4},
			},
			assignment{1, 1},
		},
		assignseq{
			rotation4Puzzle1PartialValues,
			[]assignment{
				assignment{13, 4},
			},
			assignment{4, 4},
		},
	}
	for _, test := range tests {
		if test.init == nil {
			t.Log("NEW TEST (starter puzzle empty)")
			p, _ = helperNewEmptySudokuPuzzle(4)
		} else {
			p, _ = helperNewSudokuPuzzle(test.init)
			t.Logf("NEW TEST, starter puzzle:\n%v", p)
		}
		for _, assign := range test.setup {
			e := tryassign(assign.ai, assign.av)
			if e != nil {
				t.Fatalf("Assign(Choice{%d, %d}) failed: %v",
					assign.ai, assign.av, e.Error())
			} else if len(p.errors) > 0 {
				t.Fatalf("Assign(Choice{%d, %d}) failed: %v",
					assign.ai, assign.av, p.errors)
			}
		}
		e := tryassign(test.final.ai, test.final.av)
		if e == nil && len(p.errors) == 0 {
			t.Errorf("Assign(Choice{%d, %d}) didn't fail", test.final.ai, test.final.av)
		}
	}
}
