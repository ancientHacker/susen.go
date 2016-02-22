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

/*

Stringer

*/

func TestPuzzleString(t *testing.T) {
	// check for the null cases
	s := (*Puzzle)(nil).String()
	e := ""
	if s != e {
		t.Errorf("Unexpected empty puzzle string: %q, Expected: %q", s, e)
	}
	// do a 4x4 test with all the different states except unknown
	p, err := New(&Summary{
		Geometry:   StandardGeometryName,
		SideLength: 4,
		Values:     rotation4Puzzle1PartialAssign1Values})
	if err != nil {
		t.Fatalf("Puzzle creation failed: %v", err)
	}
	s = p.ValuesString(false)
	e = " | 1   2 | 3   4 \n" +
		" +---+---+---+---\n" +
		"a| 1   _ | 3   _ \n" +
		"b| _   3 | _   1 \n" +
		" +---+---+---+---\n" +
		"c| 3   _ | 1   _ \n" +
		"d| 2   1 | _   3 \n"
	if s != e {
		t.Errorf("Unexpected puzzle string:\n%vExpected:\n%v", s, e)
	}
	s = p.String()
	e = " | 1   2 | 3   4 \n" +
		" +---+---+---+---\n" +
		"a| 1  +2 | 3  2,4\n" +
		"b|=4   3 |+2   1 \n" +
		" +---+---+---+---\n" +
		"c| 3  =4 | 1  +2 \n" +
		"d| 2   1 |=4   3 \n"
	if s != e {
		t.Errorf("Unexpected puzzle string:\n%vExpected:\n%v", s, e)
	}
	// do a 9x9 empty puzzle test to cover unknown squares
	p, err = New(&Summary{nil, StandardGeometryName, 9, nil, nil})
	if err != nil {
		t.Fatalf("Puzzle creation failed: %v", err)
	}
	s = p.String()
	e = " | 1   2   3 | 4   5   6 | 7   8   9 \n" +
		" +---+---+---+---+---+---+---+---+---\n" +
		"a| _   _   _ | _   _   _ | _   _   _ \n" +
		"b| _   _   _ | _   _   _ | _   _   _ \n" +
		"c| _   _   _ | _   _   _ | _   _   _ \n" +
		" +---+---+---+---+---+---+---+---+---\n" +
		"d| _   _   _ | _   _   _ | _   _   _ \n" +
		"e| _   _   _ | _   _   _ | _   _   _ \n" +
		"f| _   _   _ | _   _   _ | _   _   _ \n" +
		" +---+---+---+---+---+---+---+---+---\n" +
		"g| _   _   _ | _   _   _ | _   _   _ \n" +
		"h| _   _   _ | _   _   _ | _   _   _ \n" +
		"i| _   _   _ | _   _   _ | _   _   _ \n"
	if s != e {
		t.Errorf("Unexpected puzzle string:\n%vExpected:\n%v", s, e)
	}
	// do a 12x12 empty puzzle test to cover rectangular borders
	p, err = New(&Summary{nil, RectangularGeometryName, 12, nil, nil})
	if err != nil {
		t.Fatalf("Puzzle creation failed: %v", err)
	}
	s = p.String()
	e = " | 1   2   3   4 | 5   6   7   8 | 9  10  11  12 \n" +
		" +---+---+---+---+---+---+---+---+---+---+---+---\n" +
		"a| _   _   _   _ | _   _   _   _ | _   _   _   _ \n" +
		"b| _   _   _   _ | _   _   _   _ | _   _   _   _ \n" +
		"c| _   _   _   _ | _   _   _   _ | _   _   _   _ \n" +
		" +---+---+---+---+---+---+---+---+---+---+---+---\n" +
		"d| _   _   _   _ | _   _   _   _ | _   _   _   _ \n" +
		"e| _   _   _   _ | _   _   _   _ | _   _   _   _ \n" +
		"f| _   _   _   _ | _   _   _   _ | _   _   _   _ \n" +
		" +---+---+---+---+---+---+---+---+---+---+---+---\n" +
		"g| _   _   _   _ | _   _   _   _ | _   _   _   _ \n" +
		"h| _   _   _   _ | _   _   _   _ | _   _   _   _ \n" +
		"i| _   _   _   _ | _   _   _   _ | _   _   _   _ \n" +
		" +---+---+---+---+---+---+---+---+---+---+---+---\n" +
		"j| _   _   _   _ | _   _   _   _ | _   _   _   _ \n" +
		"k| _   _   _   _ | _   _   _   _ | _   _   _   _ \n" +
		"l| _   _   _   _ | _   _   _   _ | _   _   _   _ \n"
	if s != e {
		t.Errorf("Unexpected puzzle string:\n%vExpected:\n%v", s, e)
	}
}

/*

Markdown

*/

func TestPuzzleValuesMarkdown(t *testing.T) {
	// check for the null cases
	s := (*Puzzle)(nil).String()
	e := ""
	if s != e {
		t.Errorf("Unexpected empty puzzle string: %q, Expected: %q", s, e)
	}
	// do a 4x4 test with all the different states except unknown
	p, err := New(&Summary{
		Geometry:   StandardGeometryName,
		SideLength: 4,
		Values:     rotation4Puzzle1PartialAssign1Values})
	if err != nil {
		t.Fatalf("Puzzle creation failed: %v", err)
	}
	s = p.ValuesMarkdown(false)
	e = "|     |  1  |  2  |  3  |  4  |\n" +
		"|:---:|:---:|:---:|:---:|:---:|\n" +
		"|**a**|  1  |     |  3  |     |\n" +
		"|**b**|     |  3  |     |  1  |\n" +
		"|**c**|  3  |     |  1  |     |\n" +
		"|**d**|  2  |  1  |     |  3  |\n"
	if s != e {
		t.Errorf("Unexpected puzzle string:\n%vExpected:\n%v", s, e)
	}
	s = p.ValuesMarkdown(true)
	e = "|     |  1  |  2  |  3  |  4  |\n" +
		"|:---:|:---:|:---:|:---:|:---:|\n" +
		"|**a**|  1  | +2  |  3  | 2,4 |\n" +
		"|**b**| =4  |  3  | +2  |  1  |\n" +
		"|**c**|  3  | =4  |  1  | +2  |\n" +
		"|**d**|  2  |  1  | =4  |  3  |\n"
	if s != e {
		t.Errorf("Unexpected puzzle string:\n%vExpected:\n%v", s, e)
	}
	// do a 9x9 empty puzzle test to cover unknown squares
	p, err = New(&Summary{nil, StandardGeometryName, 9, nil, nil})
	if err != nil {
		t.Fatalf("Puzzle creation failed: %v", err)
	}
	s = p.ValuesMarkdown(true)
	e = "|     |  1  |  2  |  3  |  4  |  5  |  6  |  7  |  8  |  9  |\n" +
		"|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|\n" +
		"|**a**|     |     |     |     |     |     |     |     |     |\n" +
		"|**b**|     |     |     |     |     |     |     |     |     |\n" +
		"|**c**|     |     |     |     |     |     |     |     |     |\n" +
		"|**d**|     |     |     |     |     |     |     |     |     |\n" +
		"|**e**|     |     |     |     |     |     |     |     |     |\n" +
		"|**f**|     |     |     |     |     |     |     |     |     |\n" +
		"|**g**|     |     |     |     |     |     |     |     |     |\n" +
		"|**h**|     |     |     |     |     |     |     |     |     |\n" +
		"|**i**|     |     |     |     |     |     |     |     |     |\n"
	if s != e {
		t.Errorf("Unexpected puzzle string:\n%vExpected:\n%v", s, e)
	}
}
