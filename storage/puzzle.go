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

package storage

import (
	"github.com/ancientHacker/susen.go/puzzle"
)

// puzzle data
var (
	defaultPuzzleID = "standard-1"
	puzzleSummaries = map[string]*puzzle.Summary{
		"standard-1": &puzzle.Summary{
			Geometry:   puzzle.StandardGeometryName,
			SideLength: 9,
			Values: []int{
				4, 0, 0, 0, 0, 3, 5, 0, 2,
				0, 0, 9, 5, 0, 6, 3, 4, 0,
				0, 0, 0, 0, 0, 0, 0, 0, 8,
				0, 0, 0, 0, 3, 4, 8, 6, 0,
				0, 0, 4, 6, 0, 5, 2, 0, 0,
				0, 2, 8, 7, 9, 0, 0, 0, 0,
				9, 0, 0, 0, 0, 0, 0, 0, 0,
				0, 8, 7, 3, 0, 2, 9, 0, 0,
				5, 0, 2, 9, 0, 0, 0, 0, 6,
			}},
		"standard-2": &puzzle.Summary{
			Geometry:   puzzle.StandardGeometryName,
			SideLength: 9,
			Values: []int{
				0, 1, 0, 5, 0, 6, 0, 2, 0,
				0, 0, 0, 0, 0, 3, 0, 1, 8,
				0, 0, 0, 0, 7, 0, 0, 0, 6,
				0, 0, 5, 0, 0, 0, 0, 3, 0,
				0, 0, 8, 0, 9, 0, 7, 0, 0,
				0, 6, 0, 0, 0, 0, 4, 0, 0,
				5, 0, 0, 0, 4, 0, 0, 0, 0,
				6, 4, 0, 2, 0, 0, 0, 0, 0,
				0, 3, 0, 9, 0, 1, 0, 8, 0,
			}},
		"standard-3": &puzzle.Summary{
			Geometry:   puzzle.StandardGeometryName,
			SideLength: 9,
			Values: []int{
				9, 0, 0, 4, 5, 0, 0, 0, 8,
				0, 2, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 1, 7, 2, 4, 0, 0,
				0, 7, 9, 0, 0, 0, 6, 8, 0,
				2, 0, 0, 0, 0, 0, 0, 0, 5,
				0, 4, 3, 0, 0, 0, 2, 7, 0,
				0, 0, 8, 3, 2, 5, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 6, 0,
				4, 0, 0, 0, 1, 6, 0, 0, 3,
			}},
		"standard-4": &puzzle.Summary{
			Geometry:   puzzle.StandardGeometryName,
			SideLength: 9,
			Values: []int{
				9, 4, 8, 0, 5, 0, 2, 0, 0,
				0, 0, 7, 8, 0, 3, 0, 0, 1,
				0, 5, 0, 0, 7, 0, 0, 0, 0,
				0, 7, 0, 0, 0, 0, 3, 0, 0,
				2, 0, 0, 6, 0, 5, 0, 0, 4,
				0, 0, 5, 0, 0, 0, 0, 9, 0,
				0, 0, 0, 0, 6, 0, 0, 1, 0,
				3, 0, 0, 5, 0, 9, 7, 0, 0,
				0, 0, 6, 0, 1, 0, 4, 2, 3,
			}},
		"standard-5": &puzzle.Summary{
			Geometry:   puzzle.StandardGeometryName,
			SideLength: 9,
			Values: []int{
				0, 0, 0, 0, 0, 0, 0, 0, 0,
				9, 0, 0, 5, 0, 7, 0, 3, 0,
				0, 0, 0, 1, 0, 0, 6, 0, 7,
				0, 4, 0, 0, 6, 0, 0, 8, 2,
				6, 7, 0, 0, 0, 0, 0, 1, 3,
				3, 8, 0, 0, 1, 0, 0, 9, 0,
				7, 0, 5, 0, 0, 8, 0, 0, 0,
				0, 2, 0, 3, 0, 9, 0, 0, 8,
				0, 0, 0, 0, 0, 0, 0, 0, 0,
			}},
		"standard-6": &puzzle.Summary{
			Geometry:   puzzle.StandardGeometryName,
			SideLength: 9,
			Values: []int{
				2, 0, 0, 8, 0, 0, 0, 5, 0,
				0, 8, 5, 0, 0, 0, 0, 0, 0,
				0, 3, 6, 7, 5, 0, 0, 0, 1,
				0, 0, 3, 0, 4, 0, 0, 9, 8,
				0, 0, 0, 3, 0, 5, 0, 0, 0,
				4, 1, 0, 0, 6, 0, 7, 0, 0,
				5, 0, 0, 0, 0, 7, 1, 2, 0,
				0, 0, 0, 0, 0, 0, 5, 6, 0,
				0, 2, 0, 0, 0, 0, 0, 0, 4,
			}},
		"rectangular-1": &puzzle.Summary{
			Geometry:   puzzle.RectangularGeometryName,
			SideLength: 6,
			Values: []int{
				0, 4, 5, 1, 6, 0,
				3, 0, 0, 0, 0, 0,
				0, 5, 0, 6, 2, 1,
				1, 0, 2, 3, 4, 0,
				5, 0, 0, 2, 1, 6,
				6, 0, 0, 0, 0, 0,
			}},
		"rectangular-2": &puzzle.Summary{
			Geometry:   puzzle.RectangularGeometryName,
			SideLength: 6,
			Values: []int{
				0, 0, 0, 2, 6, 0,
				2, 0, 3, 0, 0, 0,
				0, 5, 0, 0, 0, 6,
				3, 2, 6, 0, 0, 1,
				0, 0, 4, 0, 0, 0,
				0, 0, 0, 5, 1, 4,
			}},
		"rectangular-3": &puzzle.Summary{
			Geometry:   puzzle.RectangularGeometryName,
			SideLength: 12,
			Values: []int{
				5, 7, 0, 6, 0, 0, 0, 0, 0, 1, 11, 12,
				11, 0, 0, 0, 0, 0, 10, 0, 0, 0, 0, 3,
				8, 0, 9, 0, 0, 0, 1, 0, 5, 7, 0, 0,
				0, 0, 4, 2, 10, 11, 0, 0, 12, 0, 0, 8,
				0, 0, 0, 0, 9, 6, 0, 1, 7, 0, 0, 0,
				0, 9, 7, 0, 0, 0, 0, 2, 11, 0, 0, 0,
				0, 0, 0, 8, 7, 0, 0, 0, 0, 11, 3, 0,
				0, 0, 0, 11, 3, 0, 2, 5, 0, 0, 0, 0,
				9, 0, 0, 3, 0, 0, 11, 8, 10, 6, 0, 0,
				0, 0, 3, 7, 0, 10, 0, 0, 0, 12, 0, 2,
				2, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 11,
				6, 11, 12, 0, 0, 0, 0, 0, 3, 0, 9, 4,
			}},
		"rectangular-4": &puzzle.Summary{
			Geometry:   puzzle.RectangularGeometryName,
			SideLength: 12,
			Values: []int{
				0, 11, 3, 0, 0, 0, 0, 0, 0, 6, 0, 0,
				0, 7, 0, 0, 12, 0, 4, 0, 0, 3, 10, 8,
				4, 6, 0, 0, 10, 11, 0, 0, 1, 0, 0, 7,
				0, 0, 8, 9, 2, 0, 0, 0, 5, 0, 0, 0,
				0, 0, 0, 0, 0, 9, 6, 0, 12, 8, 11, 0,
				0, 5, 0, 0, 3, 0, 0, 11, 0, 9, 0, 0,
				0, 0, 4, 0, 8, 0, 0, 9, 0, 0, 7, 0,
				0, 9, 7, 3, 0, 10, 12, 0, 0, 0, 0, 0,
				0, 0, 0, 11, 0, 0, 0, 1, 3, 12, 0, 0,
				3, 0, 0, 7, 0, 0, 8, 2, 0, 0, 4, 1,
				2, 8, 5, 0, 0, 12, 0, 4, 0, 0, 3, 0,
				0, 0, 9, 0, 0, 0, 0, 0, 0, 7, 12, 0,
			}},
	}
)

func CommonPuzzles() map[string]*puzzle.Summary {
	return puzzleSummaries
}

func DefaultPuzzleID() string {
	return defaultPuzzleID
}
