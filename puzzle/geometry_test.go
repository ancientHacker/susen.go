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
	"reflect"
	"testing"
)

/*

PuzzleMappings

*/

func TestFindIntSquareRoot(t *testing.T) {
	inputs := []int{1, 2, 3, 4, 5, 8, 9, 10, 15, 16}
	outputInts := []int{1, 1, 1, 2, 2, 2, 3, 3, 3, 4}
	outputBools := []bool{true, false, false, true, false, false, true, false, false, true}
	for i, v := range inputs {
		r, f := findIntSquareRoot(v)
		if r != outputInts[i] || f != outputBools[i] {
			t.Errorf("findIntSquareRoot(%d) = (%d, %v) but expected (%d, %v)",
				v, r, f, outputInts[i], outputBools[i])
		}
	}
}

func TestSquarePuzzleMapping(t *testing.T) {
	// First make sure the boundary condition logic is working
	if _, err := squarePuzzleMapping(13); err == nil {
		t.Fatalf("Creating a square puzzle mapping for puzzle size 13 did not fail.")
	} else {
		if err.(Error).Condition != NonSquareCondition {
			t.Logf("squarePuzzleMapping(13): %v", err)
			t.Errorf("Incorrect error!")
		}
	}
	if _, err := squarePuzzleMapping(1); err == nil {
		t.Fatalf("Creating a square puzzle mapping for puzzle size 1 did not fail.")
	} else {
		if err.(Error).Condition != TooSmallCondition {
			t.Logf("squarePuzzleMapping(1): %v", err)
			t.Errorf("Incorrect error!")
		}
	}
	if _, err := squarePuzzleMapping(16 * 16 * 16 * 16); err == nil {
		t.Fatalf("Creating a square puzzle mapping for puzzle size 65,536 did not fail.")
	} else {
		if err.(Error).Condition != TooLargeCondition {
			t.Logf("squarePuzzleMapping(65536): %v", err)
			t.Errorf("Incorrect error!")
		}
	}
	if _, err := squarePuzzleMapping(13 * 13); err == nil {
		t.Fatalf("Creating a square puzzle mapping for sidelen 13 did not fail.")
	} else {
		if err.(Error).Attribute != SideLengthAttribute {
			t.Logf("squarePuzzleMapping(13 x 13): %v", err)
			t.Errorf("Incorrect error!")
		}
	}

	// we test the map for 9, which is complex but possible to
	// manually simulate.  The rest of them we assume are right
	// based on the logic working for 9.
	gd9 := []groupDescriptor{
		groupDescriptor{},
		groupDescriptor{1, GroupID{GtypeRow, 1}, []int{1, 2, 3, 4, 5, 6, 7, 8, 9}},
		groupDescriptor{2, GroupID{GtypeRow, 2}, []int{10, 11, 12, 13, 14, 15, 16, 17, 18}},
		groupDescriptor{3, GroupID{GtypeRow, 3}, []int{19, 20, 21, 22, 23, 24, 25, 26, 27}},
		groupDescriptor{4, GroupID{GtypeRow, 4}, []int{28, 29, 30, 31, 32, 33, 34, 35, 36}},
		groupDescriptor{5, GroupID{GtypeRow, 5}, []int{37, 38, 39, 40, 41, 42, 43, 44, 45}},
		groupDescriptor{6, GroupID{GtypeRow, 6}, []int{46, 47, 48, 49, 50, 51, 52, 53, 54}},
		groupDescriptor{7, GroupID{GtypeRow, 7}, []int{55, 56, 57, 58, 59, 60, 61, 62, 63}},
		groupDescriptor{8, GroupID{GtypeRow, 8}, []int{64, 65, 66, 67, 68, 69, 70, 71, 72}},
		groupDescriptor{9, GroupID{GtypeRow, 9}, []int{73, 74, 75, 76, 77, 78, 79, 80, 81}},
		groupDescriptor{10, GroupID{GtypeCol, 1}, []int{1, 10, 19, 28, 37, 46, 55, 64, 73}},
		groupDescriptor{11, GroupID{GtypeCol, 2}, []int{2, 11, 20, 29, 38, 47, 56, 65, 74}},
		groupDescriptor{12, GroupID{GtypeCol, 3}, []int{3, 12, 21, 30, 39, 48, 57, 66, 75}},
		groupDescriptor{13, GroupID{GtypeCol, 4}, []int{4, 13, 22, 31, 40, 49, 58, 67, 76}},
		groupDescriptor{14, GroupID{GtypeCol, 5}, []int{5, 14, 23, 32, 41, 50, 59, 68, 77}},
		groupDescriptor{15, GroupID{GtypeCol, 6}, []int{6, 15, 24, 33, 42, 51, 60, 69, 78}},
		groupDescriptor{16, GroupID{GtypeCol, 7}, []int{7, 16, 25, 34, 43, 52, 61, 70, 79}},
		groupDescriptor{17, GroupID{GtypeCol, 8}, []int{8, 17, 26, 35, 44, 53, 62, 71, 80}},
		groupDescriptor{18, GroupID{GtypeCol, 9}, []int{9, 18, 27, 36, 45, 54, 63, 72, 81}},
		groupDescriptor{19, GroupID{GtypeTile, 1}, []int{1, 2, 3, 10, 11, 12, 19, 20, 21}},
		groupDescriptor{20, GroupID{GtypeTile, 2}, []int{4, 5, 6, 13, 14, 15, 22, 23, 24}},
		groupDescriptor{21, GroupID{GtypeTile, 3}, []int{7, 8, 9, 16, 17, 18, 25, 26, 27}},
		groupDescriptor{22, GroupID{GtypeTile, 4}, []int{28, 29, 30, 37, 38, 39, 46, 47, 48}},
		groupDescriptor{23, GroupID{GtypeTile, 5}, []int{31, 32, 33, 40, 41, 42, 49, 50, 51}},
		groupDescriptor{24, GroupID{GtypeTile, 6}, []int{34, 35, 36, 43, 44, 45, 52, 53, 54}},
		groupDescriptor{25, GroupID{GtypeTile, 7}, []int{55, 56, 57, 64, 65, 66, 73, 74, 75}},
		groupDescriptor{26, GroupID{GtypeTile, 8}, []int{58, 59, 60, 67, 68, 69, 76, 77, 78}},
		groupDescriptor{27, GroupID{GtypeTile, 9}, []int{61, 62, 63, 70, 71, 72, 79, 80, 81}},
	}
	gm9 := [][]int{
		[]int(nil),
		[]int{1, 10, 19}, []int{1, 11, 19}, []int{1, 12, 19},
		[]int{1, 13, 20}, []int{1, 14, 20}, []int{1, 15, 20},
		[]int{1, 16, 21}, []int{1, 17, 21}, []int{1, 18, 21},
		[]int{2, 10, 19}, []int{2, 11, 19}, []int{2, 12, 19},
		[]int{2, 13, 20}, []int{2, 14, 20}, []int{2, 15, 20},
		[]int{2, 16, 21}, []int{2, 17, 21}, []int{2, 18, 21},
		[]int{3, 10, 19}, []int{3, 11, 19}, []int{3, 12, 19},
		[]int{3, 13, 20}, []int{3, 14, 20}, []int{3, 15, 20},
		[]int{3, 16, 21}, []int{3, 17, 21}, []int{3, 18, 21},
		[]int{4, 10, 22}, []int{4, 11, 22}, []int{4, 12, 22},
		[]int{4, 13, 23}, []int{4, 14, 23}, []int{4, 15, 23},
		[]int{4, 16, 24}, []int{4, 17, 24}, []int{4, 18, 24},
		[]int{5, 10, 22}, []int{5, 11, 22}, []int{5, 12, 22},
		[]int{5, 13, 23}, []int{5, 14, 23}, []int{5, 15, 23},
		[]int{5, 16, 24}, []int{5, 17, 24}, []int{5, 18, 24},
		[]int{6, 10, 22}, []int{6, 11, 22}, []int{6, 12, 22},
		[]int{6, 13, 23}, []int{6, 14, 23}, []int{6, 15, 23},
		[]int{6, 16, 24}, []int{6, 17, 24}, []int{6, 18, 24},
		[]int{7, 10, 25}, []int{7, 11, 25}, []int{7, 12, 25},
		[]int{7, 13, 26}, []int{7, 14, 26}, []int{7, 15, 26},
		[]int{7, 16, 27}, []int{7, 17, 27}, []int{7, 18, 27},
		[]int{8, 10, 25}, []int{8, 11, 25}, []int{8, 12, 25},
		[]int{8, 13, 26}, []int{8, 14, 26}, []int{8, 15, 26},
		[]int{8, 16, 27}, []int{8, 17, 27}, []int{8, 18, 27},
		[]int{9, 10, 25}, []int{9, 11, 25}, []int{9, 12, 25},
		[]int{9, 13, 26}, []int{9, 14, 26}, []int{9, 15, 26},
		[]int{9, 16, 27}, []int{9, 17, 27}, []int{9, 18, 27},
	}
	sm9 := puzzleMapping{StandardGeometryName, 9, 81, 27, gd9, gm9}
	sm9c := computeSquarePuzzleMapping(9, 3)
	sm9a, err := squarePuzzleMapping(81)
	if err != nil {
		t.Fatalf("Creating first side 9 square puzzle mapping returned an error: %v", err)
	}
	if !reflect.DeepEqual(sm9a, sm9c) {
		t.Fatalf("squarePuzzleMapping is not using computeSquarePuzzleMapping!")
	}
	if !reflect.DeepEqual(sm9a, &sm9) {
		t.Errorf("side 9 square puzzle mapping doesn't match expected:\n")
		for i := 0; i < 27; i++ {
			if !reflect.DeepEqual(sm9a.gdescs[i], sm9.gdescs[i]) {
				t.Errorf("group descriptor %d: %v (expected %v)\n",
					i, sm9a.gdescs[i], sm9.gdescs[i])
			}
		}
		for j := 0; j < 81; j++ {
			if !reflect.DeepEqual(sm9a.ixmap[j], sm9.ixmap[j]) {
				t.Errorf("cell map %d: %v (expected %v)\n", j, sm9a.ixmap[j], sm9.ixmap[j])
			}
		}
	}
	sm9b, err := squarePuzzleMapping(81)
	if err != nil {
		t.Fatalf("Creating second side 9 square puzzle mapping returned an error: %v", err)
	}
	if reflect.ValueOf(sm9a).Pointer() != reflect.ValueOf(sm9b).Pointer() {
		t.Errorf("First side 9 square puzzle mapping was not reused!")
	}
}

func TestFindDivisors(t *testing.T) {
	inputs := []int{1, 2, 3, 4, 5, 6, 9, 10, 12, 13}
	outputLows := []int{0, 1, 1, 1, 1, 2, 2, 2, 3, 3}
	outputHighs := []int{1, 2, 2, 2, 2, 3, 3, 3, 4, 4}
	outputBools := []bool{false, true, false, false, false, true, false, false, true, false}
	for i, v := range inputs {
		l, h, f := findDivisors(v)
		if l != outputLows[i] || h != outputHighs[i] || f != outputBools[i] {
			t.Errorf("findDivisors(%d) = (%d, %d, %v) but expected (%d, %d, %v)",
				v, l, h, f, outputLows[i], outputHighs[i], outputBools[i])
		}
	}
}

func TestRectangularPuzzleMapping(t *testing.T) {
	// First make sure the boundary condition logic is working
	if _, err := rectangularPuzzleMapping(13); err == nil {
		t.Fatalf("Creating a rectangular puzzle mapping for puzzle size 13 did not fail.")
	} else {
		if err.(Error).Condition != NonSquareCondition {
			t.Logf("rectangularPuzzleMapping(13): %v", err)
			t.Errorf("Incorrect error!")
		}
	}
	if _, err := rectangularPuzzleMapping(1); err == nil {
		t.Fatalf("Creating a rectangular puzzle mapping for puzzle size 1 did not fail.")
	} else {
		if err.(Error).Condition != TooSmallCondition {
			t.Logf("rectangularPuzzleMapping(1): %v", err)
			t.Errorf("Incorrect error!")
		}
	}
	if _, err := rectangularPuzzleMapping(16 * 17 * 16 * 17); err == nil {
		t.Fatalf("Creating a rectangular puzzle mapping for puzzle size 73,984 did not fail.")
	} else {
		if err.(Error).Condition != TooLargeCondition {
			t.Logf("rectangularPuzzleMapping(73,984): %v", err)
			t.Errorf("Incorrect error!")
		}
	}
	if _, err := rectangularPuzzleMapping(13 * 13); err == nil {
		t.Fatalf("Creating a rectangular puzzle mapping for sidelen 13 did not fail.")
	} else {
		if err.(Error).Condition != NonRectangularCondition {
			t.Logf("rectangularPuzzleMapping(13 * 13): %v", err)
			t.Errorf("Incorrect error!")
		}
	}

	// we test the map for 6, which is complex but possible to
	// manually simulate.  The rest of them we assume are right
	// based on the logic working for 6.
	gd6 := []groupDescriptor{
		groupDescriptor{},
		groupDescriptor{1, GroupID{GtypeRow, 1}, []int{1, 2, 3, 4, 5, 6}},
		groupDescriptor{2, GroupID{GtypeRow, 2}, []int{7, 8, 9, 10, 11, 12}},
		groupDescriptor{3, GroupID{GtypeRow, 3}, []int{13, 14, 15, 16, 17, 18}},
		groupDescriptor{4, GroupID{GtypeRow, 4}, []int{19, 20, 21, 22, 23, 24}},
		groupDescriptor{5, GroupID{GtypeRow, 5}, []int{25, 26, 27, 28, 29, 30}},
		groupDescriptor{6, GroupID{GtypeRow, 6}, []int{31, 32, 33, 34, 35, 36}},
		groupDescriptor{7, GroupID{GtypeCol, 1}, []int{1, 7, 13, 19, 25, 31}},
		groupDescriptor{8, GroupID{GtypeCol, 2}, []int{2, 8, 14, 20, 26, 32}},
		groupDescriptor{9, GroupID{GtypeCol, 3}, []int{3, 9, 15, 21, 27, 33}},
		groupDescriptor{10, GroupID{GtypeCol, 4}, []int{4, 10, 16, 22, 28, 34}},
		groupDescriptor{11, GroupID{GtypeCol, 5}, []int{5, 11, 17, 23, 29, 35}},
		groupDescriptor{12, GroupID{GtypeCol, 6}, []int{6, 12, 18, 24, 30, 36}},
		groupDescriptor{13, GroupID{GtypeTile, 1}, []int{1, 2, 3, 7, 8, 9}},
		groupDescriptor{14, GroupID{GtypeTile, 2}, []int{4, 5, 6, 10, 11, 12}},
		groupDescriptor{15, GroupID{GtypeTile, 3}, []int{13, 14, 15, 19, 20, 21}},
		groupDescriptor{16, GroupID{GtypeTile, 4}, []int{16, 17, 18, 22, 23, 24}},
		groupDescriptor{17, GroupID{GtypeTile, 5}, []int{25, 26, 27, 31, 32, 33}},
		groupDescriptor{18, GroupID{GtypeTile, 6}, []int{28, 29, 30, 34, 35, 36}},
	}
	gm6 := [][]int{
		[]int(nil),
		[]int{1, 7, 13}, []int{1, 8, 13}, []int{1, 9, 13},
		[]int{1, 10, 14}, []int{1, 11, 14}, []int{1, 12, 14},
		[]int{2, 7, 13}, []int{2, 8, 13}, []int{2, 9, 13},
		[]int{2, 10, 14}, []int{2, 11, 14}, []int{2, 12, 14},
		[]int{3, 7, 15}, []int{3, 8, 15}, []int{3, 9, 15},
		[]int{3, 10, 16}, []int{3, 11, 16}, []int{3, 12, 16},
		[]int{4, 7, 15}, []int{4, 8, 15}, []int{4, 9, 15},
		[]int{4, 10, 16}, []int{4, 11, 16}, []int{4, 12, 16},
		[]int{5, 7, 17}, []int{5, 8, 17}, []int{5, 9, 17},
		[]int{5, 10, 18}, []int{5, 11, 18}, []int{5, 12, 18},
		[]int{6, 7, 17}, []int{6, 8, 17}, []int{6, 9, 17},
		[]int{6, 10, 18}, []int{6, 11, 18}, []int{6, 12, 18},
	}
	sm6 := puzzleMapping{RectangularGeometryName, 6, 36, 18, gd6, gm6}
	sm6c := computeRectangularPuzzleMapping(6, 2, 3)
	sm6a, err := rectangularPuzzleMapping(36)
	if err != nil {
		t.Fatalf("Creating first side 6 rectangular puzzle mapping returned an error: %v", err)
	}
	if !reflect.DeepEqual(sm6a, sm6c) {
		t.Fatalf("rectangularPuzzleMapping is not using computeRectangularPuzzleMapping!")
	}
	if !reflect.DeepEqual(sm6a, &sm6) {
		t.Errorf("side 6 rectangular puzzle mapping doesn't match expected:\n")
		for i := 0; i < 18; i++ {
			if !reflect.DeepEqual(sm6a.gdescs[i], sm6.gdescs[i]) {
				t.Errorf("group descriptor %d: %v (expected %v)\n",
					i, sm6a.gdescs[i], sm6.gdescs[i])
			}
		}
		for j := 0; j < 36; j++ {
			if !reflect.DeepEqual(sm6a.ixmap[j], sm6.ixmap[j]) {
				t.Errorf("cell map %d: %v (expected %v)\n", j, sm6a.ixmap[j], sm6.ixmap[j])
			}
		}
	}
	sm6b, err := rectangularPuzzleMapping(36)
	if err != nil {
		t.Fatalf("Creating second side 6 rectangular puzzle mapping returned an error: %v", err)
	}
	if reflect.ValueOf(sm6a).Pointer() != reflect.ValueOf(sm6b).Pointer() {
		t.Errorf("First side 6 rectangular puzzle mapping was not reused!")
	}
}
