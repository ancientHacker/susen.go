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
	"reflect"
	"testing"
)

/*

Test Values

*/

var (
	solveSimpleStartValues = []int{
		1, 0, 3, 0,
		0, 3, 0, 1,
		3, 0, 1, 0,
		0, 1, 0, 3,
	}
	solveSimpleFirstValues = []int{
		1, 2, 3, 0,
		0, 3, 0, 1,
		3, 0, 1, 0,
		0, 1, 0, 3,
	}
	solveSimpleFirstCompleteValues = []int{
		1, 2, 3, 4,
		4, 3, 2, 1,
		3, 4, 1, 2,
		2, 1, 4, 3,
	}
	solveSimpleSecondValues = []int{
		1, 4, 3, 0,
		0, 3, 0, 1,
		3, 0, 1, 0,
		0, 1, 0, 3,
	}
	solveSimpleSecondCompleteValues = []int{
		1, 4, 3, 2,
		2, 3, 4, 1,
		3, 2, 1, 4,
		4, 1, 2, 3,
	}
	multiChoiceStartValues = []int{
		1, 0, 3, 0,
		3, 0, 1, 0,
		2, 0, 4, 0,
		4, 0, 2, 0,
	}
	multiChoiceSolution1 = Solution{
		[]int{
			1, 2, 3, 4,
			3, 4, 1, 2,
			2, 1, 4, 3,
			4, 3, 2, 1,
		},
		[]Choice{Choice{2, 2}, Choice{10, 1}},
		4,
	}
	multiChoiceSolution2 = Solution{
		[]int{
			1, 2, 3, 4,
			3, 4, 1, 2,
			2, 3, 4, 1,
			4, 1, 2, 3,
		},
		[]Choice{Choice{2, 2}, Choice{10, 3}},
		4,
	}
	multiChoiceSolution3 = Solution{
		[]int{
			1, 4, 3, 2,
			3, 2, 1, 4,
			2, 1, 4, 3,
			4, 3, 2, 1,
		},
		[]Choice{Choice{2, 4}, Choice{10, 1}},
		4,
	}
	multiChoiceSolution4 = Solution{
		[]int{
			1, 4, 3, 2,
			3, 2, 1, 4,
			2, 3, 4, 1,
			4, 1, 2, 3,
		},
		[]Choice{Choice{2, 4}, Choice{10, 3}},
		4,
	}
	oneStarValues = []int{
		4, 0, 0, 0, 0, 3, 5, 0, 2,
		0, 0, 9, 5, 0, 6, 3, 4, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 8,
		0, 0, 0, 0, 3, 4, 8, 6, 0,
		0, 0, 4, 6, 0, 5, 2, 0, 0,
		0, 2, 8, 7, 9, 0, 0, 0, 0,
		9, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 8, 7, 3, 0, 2, 9, 0, 0,
		5, 0, 2, 9, 0, 0, 0, 0, 6,
	}
	oneStarBoundValues = []int{
		4, 6, 1, 8, 7, 3, 5, 9, 2,
		8, 7, 9, 5, 2, 6, 3, 4, 1,
		2, 5, 3, 4, 1, 9, 6, 7, 8,
		7, 1, 5, 2, 3, 4, 8, 6, 9,
		3, 9, 4, 6, 8, 5, 2, 1, 7,
		6, 2, 8, 7, 9, 1, 4, 3, 5,
		9, 4, 6, 1, 5, 8, 7, 2, 3,
		1, 8, 7, 3, 6, 2, 9, 5, 4,
		5, 3, 2, 9, 4, 7, 1, 8, 6,
	}
	threeStarValues = []int{
		0, 1, 0, 5, 0, 6, 0, 2, 0,
		0, 0, 0, 0, 0, 3, 0, 1, 8,
		0, 0, 0, 0, 7, 0, 0, 0, 6,
		0, 0, 5, 0, 0, 0, 0, 3, 0,
		0, 0, 8, 0, 9, 0, 7, 0, 0,
		0, 6, 0, 0, 0, 0, 4, 0, 0,
		5, 0, 0, 0, 4, 0, 0, 0, 0,
		6, 4, 0, 2, 0, 0, 0, 0, 0,
		0, 3, 0, 9, 0, 1, 0, 8, 0,
	}
	threeStarBoundValues = []int{
		3, 1, 4, 5, 8, 6, 9, 2, 7,
		9, 7, 6, 4, 2, 3, 5, 1, 8,
		8, 5, 2, 1, 7, 9, 3, 4, 6,
		1, 9, 5, 7, 6, 4, 8, 3, 2,
		4, 2, 8, 3, 9, 5, 7, 6, 1,
		7, 6, 3, 8, 1, 2, 4, 5, 9,
		5, 8, 1, 6, 4, 7, 2, 9, 3,
		6, 4, 9, 2, 3, 8, 1, 7, 5,
		2, 3, 7, 9, 5, 1, 6, 8, 4,
	}
	fiveStarValues = []int{
		2, 0, 0, 8, 3, 0, 0, 5, 0,
		0, 8, 5, 0, 0, 0, 0, 0, 0,
		0, 3, 6, 7, 5, 0, 0, 0, 1,
		0, 0, 3, 0, 4, 0, 0, 9, 8,
		0, 0, 0, 3, 0, 5, 0, 0, 0,
		4, 1, 0, 0, 6, 0, 7, 0, 0,
		5, 0, 0, 0, 0, 7, 1, 2, 0,
		0, 0, 0, 0, 0, 0, 5, 6, 0,
		0, 2, 0, 0, 0, 0, 0, 0, 4,
	}
	fiveStarSolution1 = Solution{
		[]int{
			2, 4, 7, 8, 3, 1, 9, 5, 6,
			1, 8, 5, 6, 2, 9, 3, 4, 7,
			9, 3, 6, 7, 5, 4, 2, 8, 1,
			7, 5, 3, 1, 4, 2, 6, 9, 8,
			6, 9, 8, 3, 7, 5, 4, 1, 2,
			4, 1, 2, 9, 6, 8, 7, 3, 5,
			5, 6, 9, 4, 8, 7, 1, 2, 3,
			8, 7, 4, 2, 1, 3, 5, 6, 9,
			3, 2, 1, 5, 9, 6, 8, 7, 4,
		},
		[]Choice{Choice{2, 4}},
		3,
	}
	fiveStarSolution2 = Solution{
		[]int{
			2, 7, 4, 8, 3, 1, 9, 5, 6,
			1, 8, 5, 6, 2, 9, 3, 4, 7,
			9, 3, 6, 7, 5, 4, 2, 8, 1,
			7, 5, 3, 1, 4, 2, 6, 9, 8,
			6, 9, 8, 3, 7, 5, 4, 1, 2,
			4, 1, 2, 9, 6, 8, 7, 3, 5,
			5, 6, 9, 4, 8, 7, 1, 2, 3,
			8, 4, 7, 2, 1, 3, 5, 6, 9,
			3, 2, 1, 5, 9, 6, 8, 7, 4,
		},
		[]Choice{Choice{2, 7}},
		3,
	}
	sixStarValues = []int{
		9, 0, 0, 4, 5, 0, 0, 0, 8,
		0, 2, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 1, 7, 2, 4, 0, 0,
		0, 7, 9, 0, 0, 0, 6, 8, 0,
		2, 0, 0, 0, 0, 0, 0, 0, 5,
		0, 4, 3, 0, 0, 0, 2, 7, 0,
		0, 0, 8, 3, 2, 5, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 6, 0,
		4, 0, 0, 0, 1, 6, 0, 0, 3,
	}
	sixStarSolution = Solution{
		[]int{
			9, 6, 1, 4, 5, 3, 7, 2, 8,
			7, 2, 4, 6, 8, 9, 5, 3, 1,
			8, 3, 5, 1, 7, 2, 4, 9, 6,
			5, 7, 9, 2, 3, 1, 6, 8, 4,
			2, 8, 6, 9, 4, 7, 3, 1, 5,
			1, 4, 3, 5, 6, 8, 2, 7, 9,
			6, 1, 8, 3, 2, 5, 9, 4, 7,
			3, 5, 7, 8, 9, 4, 1, 6, 2,
			4, 9, 2, 7, 1, 6, 8, 5, 3,
		},
		[]Choice{Choice{2, 6}},
		3,
	}
	multiSolutionValues = []int{
		2, 0, 0, 8, 0, 0, 0, 5, 0,
		0, 8, 5, 0, 0, 0, 0, 0, 0,
		0, 3, 6, 7, 5, 0, 0, 0, 1,
		0, 0, 3, 0, 4, 0, 0, 9, 8,
		0, 0, 0, 3, 0, 5, 0, 0, 0,
		4, 1, 0, 0, 6, 0, 7, 0, 0,
		5, 0, 0, 0, 0, 7, 1, 2, 0,
		0, 0, 0, 0, 0, 0, 5, 6, 0,
		0, 2, 0, 0, 0, 0, 0, 0, 4,
	}
	chronOneValues = []int{
		9, 4, 8, 0, 5, 0, 2, 0, 0,
		0, 0, 7, 8, 0, 3, 0, 0, 1,
		0, 5, 0, 0, 7, 0, 0, 0, 0,
		0, 7, 0, 0, 0, 0, 3, 0, 0,
		2, 0, 0, 6, 0, 5, 0, 0, 4,
		0, 0, 5, 0, 0, 0, 0, 9, 0,
		0, 0, 0, 0, 6, 0, 0, 1, 0,
		3, 0, 0, 5, 0, 9, 7, 0, 0,
		0, 0, 6, 0, 1, 0, 4, 2, 3,
	}
	chronOneBoundValues = []int{
		9, 4, 8, 1, 5, 6, 2, 3, 7,
		6, 2, 7, 8, 4, 3, 9, 5, 1,
		1, 5, 3, 9, 7, 2, 6, 4, 8,
		4, 7, 9, 2, 8, 1, 3, 6, 5,
		2, 3, 1, 6, 9, 5, 8, 7, 4,
		8, 6, 5, 4, 3, 7, 1, 9, 2,
		7, 8, 2, 3, 6, 4, 5, 1, 9,
		3, 1, 4, 5, 2, 9, 7, 8, 6,
		5, 9, 6, 7, 1, 8, 4, 2, 3,
	}
	chronTwoValues = []int{
		0, 0, 0, 0, 0, 0, 0, 0, 0,
		9, 0, 0, 5, 0, 7, 0, 3, 0,
		0, 0, 0, 1, 0, 0, 6, 0, 7,
		0, 4, 0, 0, 6, 0, 0, 8, 2,
		6, 7, 0, 0, 0, 0, 0, 1, 3,
		3, 8, 0, 0, 1, 0, 0, 9, 0,
		7, 0, 5, 0, 0, 8, 0, 0, 0,
		0, 2, 0, 3, 0, 9, 0, 0, 8,
		0, 0, 0, 0, 0, 0, 0, 0, 0,
	}
	chronTwoSolution = Solution{
		[]int{
			1, 5, 7, 8, 3, 6, 9, 2, 4,
			9, 6, 4, 5, 2, 7, 8, 3, 1,
			2, 3, 8, 1, 9, 4, 6, 5, 7,
			5, 4, 1, 9, 6, 3, 7, 8, 2,
			6, 7, 9, 4, 8, 2, 5, 1, 3,
			3, 8, 2, 7, 1, 5, 4, 9, 6,
			7, 1, 5, 2, 4, 8, 3, 6, 9,
			4, 2, 6, 3, 5, 9, 1, 7, 8,
			8, 9, 3, 6, 7, 1, 2, 4, 5,
		},
		[]Choice{Choice{2, 5}},
		3,
	}
	tileRotationCompleteValues = []int{
		1, 2, 3, 4, 5, 6, 7, 8, 9,
		4, 5, 6, 7, 8, 9, 1, 2, 3,
		7, 8, 9, 1, 2, 3, 4, 5, 6,
		2, 3, 4, 5, 6, 7, 8, 9, 1,
		5, 6, 7, 8, 9, 1, 2, 3, 4,
		8, 9, 1, 2, 3, 4, 5, 6, 7,
		3, 4, 5, 6, 7, 8, 9, 1, 2,
		6, 7, 8, 9, 1, 2, 3, 4, 5,
		9, 1, 2, 3, 4, 5, 6, 7, 8,
	}
	Su6Standard1Values = []int{
		0, 4, 5, 1, 6, 0,
		3, 0, 0, 0, 0, 0,
		0, 5, 0, 6, 2, 1,
		1, 0, 2, 3, 4, 0,
		5, 0, 0, 2, 1, 6,
		6, 0, 0, 0, 0, 0,
	}
	Su6Standard1Complete = []int{
		2, 4, 5, 1, 6, 3,
		3, 1, 6, 4, 5, 2,
		4, 5, 3, 6, 2, 1,
		1, 6, 2, 3, 4, 5,
		5, 3, 4, 2, 1, 6,
		6, 2, 1, 5, 3, 4,
	}
	Su6Difficult1Values = []int{
		0, 0, 0, 2, 6, 0,
		2, 0, 3, 0, 0, 0,
		0, 5, 0, 0, 0, 6,
		3, 2, 6, 0, 0, 1,
		0, 0, 4, 0, 0, 0,
		0, 0, 0, 5, 1, 4,
	}
	Su6Difficult1Complete = []int{
		1, 4, 5, 2, 6, 3,
		2, 6, 3, 1, 4, 5,
		4, 5, 1, 3, 2, 6,
		3, 2, 6, 4, 5, 1,
		5, 1, 4, 6, 3, 2,
		6, 3, 2, 5, 1, 4,
	}
	SuDozen78097Values = []int{
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
	}
	SuDozen78097Complete = []int{
		5, 7, 10, 6, 4, 8, 3, 9, 2, 1, 11, 12,
		11, 12, 2, 1, 5, 7, 10, 6, 9, 4, 8, 3,
		8, 3, 9, 4, 11, 2, 1, 12, 5, 7, 10, 6,
		1, 6, 4, 2, 10, 11, 7, 3, 12, 9, 5, 8,
		3, 8, 11, 5, 9, 6, 12, 1, 7, 2, 4, 10,
		10, 9, 7, 12, 8, 5, 4, 2, 11, 3, 6, 1,
		12, 2, 1, 8, 7, 9, 6, 10, 4, 11, 3, 5,
		7, 10, 6, 11, 3, 4, 2, 5, 1, 8, 12, 9,
		9, 4, 5, 3, 1, 12, 11, 8, 10, 6, 2, 7,
		4, 5, 3, 7, 6, 10, 9, 11, 8, 12, 1, 2,
		2, 1, 8, 9, 12, 3, 5, 4, 6, 10, 7, 11,
		6, 11, 12, 10, 2, 1, 8, 7, 3, 5, 9, 4,
	}
	SuDozen61054Values = []int{
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
	}
	SuDozen61054Complete = []int{
		10, 11, 3, 8, 1, 2, 9, 7, 4, 6, 5, 12,
		9, 7, 1, 2, 12, 5, 4, 6, 11, 3, 10, 8,
		4, 6, 12, 5, 10, 11, 3, 8, 1, 2, 9, 7,
		11, 3, 8, 9, 2, 4, 7, 12, 5, 1, 6, 10,
		7, 1, 2, 4, 5, 9, 6, 10, 12, 8, 11, 3,
		12, 5, 10, 6, 3, 8, 1, 11, 7, 9, 2, 4,
		1, 2, 4, 12, 8, 3, 11, 9, 6, 10, 7, 5,
		8, 9, 7, 3, 6, 10, 12, 5, 2, 4, 1, 11,
		5, 10, 6, 11, 4, 7, 2, 1, 3, 12, 8, 9,
		3, 12, 11, 7, 9, 6, 8, 2, 10, 5, 4, 1,
		2, 8, 5, 1, 7, 12, 10, 4, 9, 11, 3, 6,
		6, 4, 9, 10, 11, 1, 5, 3, 8, 7, 12, 2,
	}
)

type assignKnownTestcase struct {
	sidelen int
	before  []int
	after   []int
}

func TestAssignKnown(t *testing.T) {
	tcs := []assignKnownTestcase{
		assignKnownTestcase{9, oneStarValues, oneStarBoundValues},
		assignKnownTestcase{4, solveSimpleFirstValues, solveSimpleFirstCompleteValues},
		assignKnownTestcase{4, solveSimpleSecondValues, solveSimpleSecondCompleteValues},
	}
	for i, tc := range tcs {
		p, e := New(&Summary{Geometry: StandardGeometryName, SideLength: tc.sidelen, Values: tc.before})
		if e != nil {
			t.Fatalf("TestBindAll case %d: Failed to create test puzzle: %v", i+1, e)
		}
		if !assignKnown(p) {
			t.Errorf("TestBindAll case %d: Failed to bind all.", i+1)
		}
		if tc.after != nil {
			vs := p.allValues()
			if !reflect.DeepEqual(vs, tc.after) {
				t.Errorf("TestBindAll case %d: Binding produced: %v (expected: %v)",
					i+1, vs, tc.after)
			}
		} else {
			// show the output of the binding for debugging purposes
			// t.Logf("TestBindAll case %d: Result after binding:\n%v", i+1, p.allValues())
		}
	}
}

func TestPopThread(t *testing.T) {
	pin, e := New(&Summary{Geometry: StandardGeometryName, SideLength: 4, Values: solveSimpleStartValues})
	if e != nil {
		t.Fatalf("TestPopThread: Failed to create puzzle: %v", e)
	}
	thin := thread{choice{pin, 2, 2, 0, intset{2, 4}}} // artificial stack top
	p, th := popChoice(pin, thin)
	if reflect.DeepEqual(p, pin) ||
		len(th) != 1 || th[0].cindex != 2 ||
		th[0].cvalue != 2 || !reflect.DeepEqual(th[0].cnext, intset{4}) {
		t.Errorf("TestPopThread: 1st popped stack top is wrong: %+v", th[0])
	}
	if !reflect.DeepEqual(p.allValues(), solveSimpleFirstValues) {
		t.Fatalf("TestPopThread: 1st popped stack puzzle is %v (expected %v)",
			p.allValues(), solveSimpleFirstValues)
	}
	pin, thin = p, th
	p, th = popChoice(pin, thin)
	if reflect.DeepEqual(p, pin) ||
		len(th) != 1 || th[0].cindex != 2 ||
		th[0].cvalue != 4 || !reflect.DeepEqual(th[0].cnext, intset{}) {
		t.Errorf("TestPopThread: 2nd popped stack top is wrong: %+v", th[0])
	}
	if !reflect.DeepEqual(p.allValues(), solveSimpleSecondValues) {
		t.Fatalf("TestPopThread: 2nd popped stack puzzle is %v (expected %v)",
			p.allValues(), solveSimpleSecondValues)
	}
	pin, thin = p, th
	p, th = popChoice(pin, thin)
	if !reflect.DeepEqual(p, pin) ||
		len(th) != 0 {
		t.Errorf("TestPopThread: 3rd popped stack top is wrong: %+v", th[0])
	}
	if !reflect.DeepEqual(p.allValues(), solveSimpleSecondValues) {
		t.Fatalf("TestPopThread: 3rd popped stack puzzle is %v (expected %v)",
			p.allValues(), solveSimpleSecondValues)
	}
}

func TestPushThread(t *testing.T) {
	// first test has an early square with 2 possibles
	pin, e := New(&Summary{Geometry: StandardGeometryName, SideLength: 4, Values: solveSimpleStartValues})
	if e != nil {
		t.Fatalf("TestPushThread: Failed to create 1st puzzle: %v", e)
	}
	p, th := pushChoice(pin, nil)
	if len(th) != 1 {
		t.Fatalf("TestPushThread: 1st pushed stack is too deep.")
	}
	if reflect.DeepEqual(p, th[0].puz) ||
		th[0].cindex != 2 || th[0].cvalue != 2 ||
		!reflect.DeepEqual(th[0].cnext, intset{4}) {
		t.Errorf("TestPushThread: 1st pushed stack top is wrong: %+v", th[0])
	}
	if !reflect.DeepEqual(p.allValues(), solveSimpleFirstValues) {
		t.Errorf("TestPushThread: 1st pushed stack puzzle is %v (expected %v)",
			p.allValues(), solveSimpleFirstValues)
	}
	// second test all squares have 4 possibles
	pin, e = New(&Summary{Geometry: StandardGeometryName, SideLength: 4, Values: empty4PuzzleValues})
	if e != nil {
		t.Fatalf("TestPushThread: Failed to create 2nd puzzle: %v", e)
	}
	p, th = pushChoice(pin, nil)
	if len(th) != 1 {
		t.Fatalf("TestPushThread: 2nd pushed stack is too deep.")
	}
	if reflect.DeepEqual(p, th[0].puz) ||
		th[0].cindex != 1 || th[0].cvalue != 1 ||
		!reflect.DeepEqual(th[0].cnext, intset{2, 3, 4}) {
		t.Errorf("TestPushThread: 2nd pushed stack top is wrong: %+v", th[0])
	}
	if !reflect.DeepEqual(p.allValues(), empty4PuzzleAssign1Values) {
		t.Errorf("TestPushThread: 2nd pushed stack puzzle is %v (expected %v)",
			p.allValues(), empty4PuzzleAssign1Values)
	}
}

type solveTestcase struct {
	sidelen int
	start   []int
	done    bool
	finish  []int
	elen    int
	elasti  int
	elastv  int
	elastn  intset
}

func TestSolve(t *testing.T) {
	var p *Puzzle
	var th thread
	var e error
	// first check behavior on a puzzle with problems
	p, e = New(&Summary{Geometry: StandardGeometryName, SideLength: 4, Values: conflicting4Puzzle1})
	if e != nil {
		t.Fatalf("TestSolve: Failed to create conflicting puzzle: %v", e)
	}
	if len(p.errors) == 0 {
		t.Fatalf("TestSolve: Conflicting puzzle has no errors")
	}
	pc := p.copy()
	p, th = solve(p, th)
	if th != nil || !reflect.DeepEqual(p.summary(), pc.summary()) {
		t.Errorf("TestSolve: solving conflicting puzzle gave different puzzle:\n%v", p)
	}

	// now do the test cases
	tcs := []solveTestcase{
		solveTestcase{
			9, oneStarValues, true, oneStarBoundValues,
			0, 0, 0, nil,
		},
		solveTestcase{
			9, oneStarValues, true, oneStarBoundValues,
			0, 0, 0, nil,
		},
		solveTestcase{
			9, sixStarValues, true, sixStarSolution.Values,
			1, 2, 6, intset{},
		},
		solveTestcase{
			9, chronTwoValues, true, chronTwoSolution.Values,
			1, 2, 5, intset{},
		},
		solveTestcase{
			4, solveSimpleStartValues, true, solveSimpleFirstCompleteValues,
			1, 2, 2, intset{4},
		},
		solveTestcase{
			4, nil, true, solveSimpleSecondCompleteValues,
			1, 2, 4, intset{},
		},
	}
	for i, tc := range tcs {
		if tc.start == nil {
			if p == nil {
				t.Fatalf("Invalid case %d: no starting or exising puzzle.", i)
			}
			p, th = popChoice(p, th)
		} else {
			p, e = New(&Summary{Geometry: StandardGeometryName, SideLength: tc.sidelen, Values: tc.start})
			if e != nil {
				t.Fatalf("TestSolve case %d: Failed to create puzzle: %v", i+1, e)
			}
			th = nil
		}
		// t.Logf("TestSolve case %d: start thread %v, puzzle:\n%v", i+1, th, p)
		p, th = solve(p, th)
		// t.Logf("TestSolve case %d: finish thread %v, puzzle:\n%v", i+1, th, p)
		if tc.done {
			if len(p.errors) > 0 {
				t.Fatalf("TestSolve case %d: Failed to solve puzzle: %v", i+1, p.errors)
			}
			if tc.finish == nil {
				t.Errorf("TestSolve case %d: solution was: %v\n%s\n", i+1, p.allValues(), p)
			} else {
				if !reflect.DeepEqual(p.allValues(), tc.finish) {
					t.Errorf("TestSolve case %d: Solved puzzle is %v (expected %v)",
						i+1, p.allValues(), tc.finish)
				}
				if len(th) != tc.elen {
					t.Errorf("TestSolve case %d: Solution length %d (expected %d): %+v",
						i+1, len(th), tc.elen, th)
				} else if tc.elen > 0 {
					if th[tc.elen-1].cindex != tc.elasti ||
						th[tc.elen-1].cvalue != tc.elastv ||
						!reflect.DeepEqual(th[tc.elen-1].cnext, tc.elastn) {
						t.Errorf("TestSolve case %d: Last choice is wrong: %+v",
							i+1, th[tc.elen-1])
					}
				}
			}
		} else {
			if len(p.errors) == 0 {
				t.Errorf("TestSolve case %d: Unexpected solution: %v", i+1, p.allValues())
			}
			if len(th) != 0 {
				t.Errorf("TestSolve case %d: Unexpected remaining thread: %v", i+1, th)
			}
		}
	}
}

type solutionsTestcase struct {
	geometry string
	sidelen  int
	start    []int
	numsolns int
	solns    []Solution
}

func TestSolutions(t *testing.T) {
	tcs := []solutionsTestcase{
		// first the fully bound puzzles
		solutionsTestcase{
			StandardGeometryName, 9, oneStarValues,
			1, []Solution{Solution{oneStarBoundValues, nil, 2}},
		},
		solutionsTestcase{
			StandardGeometryName, 9, threeStarValues,
			1, []Solution{Solution{threeStarBoundValues, nil, 1}},
		},
		solutionsTestcase{
			StandardGeometryName, 9, chronOneValues,
			1, []Solution{Solution{chronOneBoundValues, nil, 1}},
		},
		// then the single-solution puzzles
		solutionsTestcase{
			StandardGeometryName, 9, sixStarValues,
			1, []Solution{sixStarSolution},
		},
		solutionsTestcase{
			StandardGeometryName, 9, chronTwoValues,
			1, []Solution{chronTwoSolution},
		},
		// then the multi-solution puzzles
		solutionsTestcase{
			StandardGeometryName, 4, solveSimpleStartValues,
			2,
			[]Solution{
				Solution{solveSimpleFirstCompleteValues, []Choice{Choice{2, 2}}, 3},
				Solution{solveSimpleSecondCompleteValues, []Choice{Choice{2, 4}}, 3},
			},
		},
		solutionsTestcase{
			StandardGeometryName, 4, multiChoiceStartValues,
			4,
			[]Solution{
				multiChoiceSolution1,
				multiChoiceSolution2,
				multiChoiceSolution3,
				multiChoiceSolution4,
			},
		},
		solutionsTestcase{
			StandardGeometryName, 9, fiveStarValues,
			2,
			[]Solution{
				fiveStarSolution1,
				fiveStarSolution2,
			},
		},
		// then the rectangular puzzles
		solutionsTestcase{
			RectangularGeometryName, 6, Su6Standard1Values,
			1, []Solution{Solution{Su6Standard1Complete, nil, 1}},
		},
		solutionsTestcase{
			RectangularGeometryName, 6, Su6Difficult1Values,
			1, []Solution{Solution{Su6Difficult1Complete, nil, 1}},
		},
		solutionsTestcase{
			RectangularGeometryName, 12, SuDozen61054Values,
			1, []Solution{Solution{SuDozen61054Complete, nil, 2}},
		},
		solutionsTestcase{
			RectangularGeometryName, 12, SuDozen78097Values,
			1, []Solution{Solution{SuDozen78097Complete, nil, 2}},
		},
		/* removed to clean the verbose output, use when needed

		// then the pathological puzzle with 14 solutions, just to
		// make sure we can handle choices that lead nowhere.
		solutionsTestcase{
			9, multiSolutionValues, 0, nil, 0
		},
		*/
	}

	for i, tc := range tcs {
		p, e := New(&Summary{Geometry: tc.geometry, SideLength: tc.sidelen, Values: tc.start})
		if e != nil {
			t.Fatalf("test %d: Failed to create puzzle: %v", i+1, e)
		}
		solns := p.allSolutions()
		if tc.numsolns == 0 {
			// this is a run for test logic only, record the solutions
			if len(solns) == 0 {
				t.Errorf("test %d puzzle has no solutions: %v", i+1, tc.start)
			}
			for j, soln := range solns {
				t.Logf("test %d solution %d: %+v", i+1, j+1, soln)
			}
		} else {
			if len(solns) != tc.numsolns {
				t.Errorf("test %d: got %d solutions, expected %d",
					i+1, len(solns), tc.numsolns)
			}
			for j := 0; j < len(solns); j++ {
				if j >= len(tc.solns) {
					t.Errorf("test %d: extra solution %d is %v",
						i+1, j+1, solns[j])
				} else {
					if !reflect.DeepEqual(solns[j], tc.solns[j]) {
						t.Errorf("test %d: solution %d is %v (expected %v)",
							i+1, j+1, solns[j], tc.solns[j])
					}
				}
			}
		}
	}
}
