// susen.go - a web-based Sudoku game and teaching tool.
// Copyright (C) 2015 Daniel C. Brotsky.
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program; if not, write to the Free Software Foundation, Inc.,
// 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA.
// Licensed under the LGPL v3.  See the LICENSE file for details

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
	s := (*Puzzle)(nil).String()
	e := ""
	if s != e {
		t.Errorf("Unexpected empty puzzle string: %q, Expected: %q", s, e)
	}
	// do a 4x4 test with all the different states except unknown
	p, err := New(&Summary{nil, SudokuGeometryName, 4, rotation4Puzzle1PartialAssign2Values, nil})
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
	p, err = New(&Summary{nil, SudokuGeometryName, 9, nil, nil})
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
