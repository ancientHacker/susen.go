package puzzle

import (
	"fmt"
	"testing"
)

/*

Printed string forms

*/

func TestVstr(t *testing.T) {
	if vstr(-1) != nonValueString {
		t.Errorf("Value form of -1 is %s, expected %s", vstr(-1), nonValueString)
	}
	if vstr(0) != " " {
		t.Errorf("Value form of 0 is %s, expected %s", vstr(0), " ")
	}
	max := len(valueStrings)
	if vstr(max) != bigValueString {
		t.Errorf("Value form of %d is %s, expected %s", max, vstr(max), bigValueString)
	}
	for i := 1; i <= 9; i++ {
		es := fmt.Sprintf("%d", i)
		if vstr(i) != es {
			t.Errorf("Value form of %d is %s, expected %s", i, vstr(i), es)
		}
	}
	// only really care about 10-25, rarely do 36x36 puzzles
	for i := 10; i <= 25; i++ {
		es := fmt.Sprintf("%c", 'A'+i-10)
		if vstr(i) != es {
			t.Errorf("Value form of %d is %s, expected %s", i, vstr(i), es)
		}
	}
}

func TestPuzzleString(t *testing.T) {
	// check for the null cases
	s := (*puzzle)(nil).String()
	e := ""
	if s != e {
		t.Errorf("Unexpected empty puzzle string: %q, Expected: %q", s, e)
	}
	// do a 4x4 test with all the different states except unknown
	p, err := helperNewSudokuPuzzle(rotation4Puzzle1PartialAssign2Values)
	if err != nil {
		t.Fatalf("Puzzle creation failed: %v", err)
	}
	s = p.String()
	e = " 1  =2 | 3  +4 \n" +
		"=4   3 |+2   1 \n" +
		"---+---+---+---\n" +
		" 3   4 | 1  =2 \n" +
		" 2   1 |=4   3 \n"
	if s != e {
		t.Errorf("Unexpected puzzle string:\n%vExpected:\n%v", s, e)
	}
	// do a 9x9 empty puzzle test to cover unknown and the formatting
	p, err = helperNewEmptySudokuPuzzle(9)
	if err != nil {
		t.Fatalf("Puzzle creation failed: %v", err)
	}
	s = p.String()
	e = " _   _   _ | _   _   _ | _   _   _ \n" +
		" _   _   _ | _   _   _ | _   _   _ \n" +
		" _   _   _ | _   _   _ | _   _   _ \n" +
		"---+---+---+---+---+---+---+---+---\n" +
		" _   _   _ | _   _   _ | _   _   _ \n" +
		" _   _   _ | _   _   _ | _   _   _ \n" +
		" _   _   _ | _   _   _ | _   _   _ \n" +
		"---+---+---+---+---+---+---+---+---\n" +
		" _   _   _ | _   _   _ | _   _   _ \n" +
		" _   _   _ | _   _   _ | _   _   _ \n" +
		" _   _   _ | _   _   _ | _   _   _ \n"
	if s != e {
		t.Errorf("Unexpected puzzle string:\n%vExpected:\n%v", s, e)
	}
}
