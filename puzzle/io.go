// susen.go - a web-based Sudoku game and teaching tool.
// Copyright (C) 2015-2016 Daniel C. Brotsky.
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
	"strconv"
)

/*

Print forms of puzzle values

*/

var (
	valueStrings = []string{
		" ", "1", "2", "3", "4", "5", "6", "7", "8", "9",
		"A", "B", "C", "D", "E", "F", "G", "H", "I", "J",
		"K", "L", "M", "N", "O", "P", "Q", "R", "S", "T",
		"U", "V", "W", "X", "Y", "Z",
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

/*

Pretty-printed puzzles in strings, for debugging.

*/

// String gives a pretty-printed view of a puzzle.
func (p *Puzzle) String() string {
	return p.ValuesString(true) + p.ErrorsString()
}

// valuesString: return a pretty-printed grid of the values.  If
// showBindings is specified, single-value squares, bound
// squares, and 2-choice squares also show their contents.
func (p *Puzzle) ValuesString(showBindings bool) (result string) {
	if p == nil {
		return
	}
	slen, tileX, tileY := p.mapping.sidelen, p.mapping.tileX, p.mapping.tileY
	// first put out the header
	result += " "
	for i := 0; i < slen; i++ {
		if i%tileX != 0 {
			result += " "
		} else {
			result += "|"
		}
		result += fmt.Sprintf("%2d ", i+1)
	}
	result += "\n"
	// next are the rows, including the separator at the top
	for ri, rowhdr := 0, 'a'; ri < slen; ri, rowhdr = ri+1, rowhdr+1 {
		if ri%tileY == 0 {
			result += " "
			for i := 0; i < slen; i++ {
				result += "+---"
			}
			result += "\n"
		}
		result += string(rowhdr)
		for i := 0; i < slen; i++ {
			s := p.squares[(ri*slen)+i+1]
			if i%tileX != 0 {
				result += " "
			} else {
				result += "|"
			}
			if s.aval != 0 {
				result += fmt.Sprintf(" %s ", vstr(s.aval))
			} else if showBindings {
				if len(s.pvals) == 1 {
					result += fmt.Sprintf("=%s ", vstr(s.pvals[0]))
				} else if s.bval != 0 {
					result += fmt.Sprintf("+%s ", vstr(s.bval))
				} else if len(s.pvals) == 2 {
					result += fmt.Sprintf("%s,%s", vstr(s.pvals[0]), vstr(s.pvals[1]))
				} else {
					result += fmt.Sprintf(" _ ")
				}
			} else {
				result += fmt.Sprintf(" _ ")
			}
		}
		result += "\n"
	}
	return
}

func (p *Puzzle) ErrorsString() (result string) {
	if p != nil {
		if elen := len(p.errors); elen > 0 {
			if elen > 1 {
				result += fmt.Sprintf("Errors (%d):\n", elen)
				for i, err := range p.errors {
					result += fmt.Sprintf("  #%d: %v\n", i+1, err)
				}
			} else {
				result += fmt.Sprintf("Error: %v\n", p.errors[0])
			}
		}
	}
	return
}

/*

Markdown-formatted tables, for documentation

*/

// ValuesMarkdown returns a markdown-format table for a puzzle as
// a sring.  Specifying showBindings produces the same variant as
// for ValuesString.
func (p *Puzzle) ValuesMarkdown(showBindings bool) (result string) {
	if p == nil {
		return
	}
	slen := p.mapping.sidelen

	// first put out the header
	result += "|     |"
	for i, header := 0, 1; i < slen; i, header = i+1, header+1 {
		result += "  " + strconv.Itoa(header) + "  |"
	}
	result += "\n"
	// next comes the header separator line
	result += "|"
	for i, header := 0, ":---:"; i < slen+1; i++ {
		result += header + "|"
	}
	result += "\n"
	// next comes the content of the puzzle,
	// with each line prefixed by a letter.
	for ri, rowhdr := 0, 'a'; ri < slen; ri, rowhdr = ri+1, rowhdr+1 {
		result += "|**" + string(rowhdr) + "**"
		for i := 0; i < slen; i++ {
			s := p.squares[(ri*slen)+i+1]
			if i == 0 {
				result += "| "
			} else {
				result += " | "
			}
			if s.aval != 0 {
				result += fmt.Sprintf(" %s ", vstr(s.aval))
			} else if showBindings {
				if len(s.pvals) == 1 {
					result += fmt.Sprintf("=%s ", vstr(s.pvals[0]))
				} else if s.bval != 0 {
					result += fmt.Sprintf("+%s ", vstr(s.bval))
				} else if len(s.pvals) == 2 {
					result += fmt.Sprintf("%s,%s", vstr(s.pvals[0]), vstr(s.pvals[1]))
				} else {
					result += fmt.Sprintf("   ")
				}
			} else {
				result += "   "
			}
		}
		result += " |\n"
	}
	return
}

func (p *Puzzle) ErrorsMarkdown() (result string) {
	if p != nil {
		if elen := len(p.errors); elen > 0 {
			if elen > 1 {
				result += fmt.Sprintf("Errors (%d):\n", elen)
				for i, err := range p.errors {
					result += fmt.Sprintf("    %d. %v\n", i+1, err)
				}
			} else {
				result += fmt.Sprintf("Error: %v\n", p.errors[0])
			}
		}
	}
	return
}
