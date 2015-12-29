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
)

/*

Pretty-printed puzzles in strings, for debugging.

*/

var (
	valueStrings = []string{
		" ", "1", "2", "3", "4", "5", "6", "7", "8", "9",
		"A", "B", "C", "D", "E", "F", "G", "H", "I", "J",
		"K", "L", "M", "N", "O", "P", "Q", "R", "S", "T",
		"U", "V", "W", "X", "Y", "Z", "a", "b", "c", "d",
		"e", "f", "g", "h", "i", "j", "k", "l", "m", "n",
		"o", "p", "q", "r", "s", "t", "u", "v", "w", "x",
		"y", "z",
	}
	nonValueString = "?"
	bigValueString = "!"
)

func vstr(i int) string {
	if i < 0 {
		return nonValueString
	}
	if i < len(valueStrings) {
		return valueStrings[i]
	}
	return bigValueString
}

// The String form of a puzzle is a pretty-printed grid with
// assigned squares, bound squares, and 2-choice squares showing
// their values.
func (p *Puzzle) String() (result string) {
	if p == nil {
		return
	}
	slen := p.mapping.sidelen
	tlen, ok := findIntSquareRoot(p.mapping.sidelen)
	if !ok {
		return "<MALFORMED PUZZLE>"
	}
	for ri := 0; ri < slen; ri++ {
		if ri > 0 && ri%tlen == 0 {
			for i := 0; i < slen; i++ {
				if i > 0 && i%tlen == 0 {
					result += "+"
				}
				if i%tlen != 0 {
					result += "+"
				}
				result += "---"
			}
			result += "\n"
		}
		for i := 0; i < slen; i++ {
			s := p.squares[(ri*slen)+i+1]
			if i > 0 && i%tlen == 0 {
				result += "|"
			}
			if i%tlen != 0 {
				result += " "
			}
			if s.aval != 0 {
				result += fmt.Sprintf(" %s ", vstr(s.aval))
			} else if len(s.pvals) == 1 {
				result += fmt.Sprintf("=%s ", vstr(s.pvals[0]))
			} else if s.bval != 0 {
				result += fmt.Sprintf("+%s ", vstr(s.bval))
			} else if len(s.pvals) == 2 {
				result += fmt.Sprintf("%s,%s", vstr(s.pvals[0]), vstr(s.pvals[1]))
			} else {
				result += fmt.Sprintf(" _ ")
			}
		}
		result += "\n"
	}
	return
}
