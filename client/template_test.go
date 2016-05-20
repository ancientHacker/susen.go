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

package client

import (
	"fmt"
	"github.com/ancientHacker/susen.go/puzzle"
	"github.com/ancientHacker/susen.go/storage"
	"os"
	"path/filepath"
	"testing"
)

var (
	rotation4Puzzle1PartialValues = []int{
		1, 0, 3, 0,
		0, 3, 0, 1,
		3, 0, 1, 0,
		0, 1, 0, 3,
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
	Su6Difficult1Values = []int{
		0, 0, 0, 2, 6, 0,
		2, 0, 3, 0, 0, 0,
		0, 5, 0, 0, 0, 6,
		3, 2, 6, 0, 0, 1,
		0, 0, 4, 0, 0, 0,
		0, 0, 0, 5, 1, 4,
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
)

func TestErrorPage(t *testing.T) {
	body := ErrorPage(fmt.Errorf("Test Error 0"))
	err := sameAsResultFile(body, "TestErrorPage0.html")
	if err != "" {
		t.Errorf("Test Error 0: got unexpected result body:\n%s:\n%v\n", err, body)
	}
}

func TestHomePage(t *testing.T) {
	session0 := "httpx-Test0"
	info0 := &storage.PuzzleInfo{
		PuzzleId:   "test-0-id",
		Name:       "test-0",
		Geometry:   puzzle.StandardGeometryName,
		SideLength: 9,
		Choices:    []puzzle.Choice{{1, 1}},
	}
	others0 := []*storage.PuzzleInfo{
		&storage.PuzzleInfo{
			PuzzleId:   "ps1",
			Name:       "pseudo-puzzle-1",
			Geometry:   puzzle.StandardGeometryName,
			SideLength: 9,
			Choices:    nil,
		},
		&storage.PuzzleInfo{
			PuzzleId:   "ps2",
			Name:       "pseudo-puzzle-2",
			Geometry:   puzzle.StandardGeometryName,
			SideLength: 16,
			Choices:    []puzzle.Choice{{2, 2}},
		},
		&storage.PuzzleInfo{
			PuzzleId:   "ps3",
			Name:       "pseudo-puzzle-3",
			Geometry:   puzzle.RectangularGeometryName,
			SideLength: 6,
			Choices:    []puzzle.Choice{{2, 2}, {3, 3}},
		},
		&storage.PuzzleInfo{
			PuzzleId:   "ps4",
			Name:       "pseudo-puzzle-4",
			Geometry:   puzzle.RectangularGeometryName,
			SideLength: 12,
			Choices:    []puzzle.Choice{{2, 2}, {3, 3}, {4, 4}},
		},
	}
	body := HomePage(session0, info0, others0)
	err := sameAsResultFile(body, "TestHomePage0.html")
	if err != "" {
		t.Errorf("Test Home 0: got unexpected result body:\n%s:\n%v\n", err, body)
	}
}

func TestSolverPage(t *testing.T) {
	session0, info0 := "httpx-Test0", &storage.PuzzleInfo{
		PuzzleId:   "test-0-id",
		Name:       "test-0",
		Geometry:   puzzle.StandardGeometryName,
		SideLength: 4,
		Choices:    []puzzle.Choice{{1, 1}},
	}
	body0 := SolverPage(session0, info0, rotation4Puzzle1PartialValues)
	err := sameAsResultFile(body0, "TestSolverPage0.html")
	if err != "" {
		t.Errorf("Test Solver 0: got unexpected result body:\n%s:\n%v\n", err, body0)
	}

	session1, info1 := "httpx-Test1", &storage.PuzzleInfo{
		PuzzleId:   "test-1-id",
		Name:       "test-1",
		Geometry:   puzzle.StandardGeometryName,
		SideLength: 9,
		Choices:    []puzzle.Choice{{1, 1}},
	}
	body1 := SolverPage(session1, info1, oneStarValues)
	err = sameAsResultFile(body1, "TestSolverPage1.html")
	if err != "" {
		t.Errorf("Test Solver 1: got unexpected result body:\n%s:\n%v\n", err, body1)
	}

	session2, info2 := "httpx-Test2", &storage.PuzzleInfo{
		PuzzleId:   "test-2-id",
		Name:       "test-2",
		Geometry:   puzzle.RectangularGeometryName,
		SideLength: 6,
		Choices:    []puzzle.Choice{{1, 1}},
	}
	body2 := SolverPage(session2, info2, Su6Difficult1Values)
	err = sameAsResultFile(body2, "TestSolverPage2.html")
	if err != "" {
		t.Errorf("Test Solver 2: got unexpected result body:\n%s:\n%v\n", err, body2)
	}

	session3, info3 := "httpx-Test3", &storage.PuzzleInfo{
		PuzzleId:   "test-3-id",
		Name:       "test-3",
		Geometry:   puzzle.RectangularGeometryName,
		SideLength: 12,
		Choices:    []puzzle.Choice{{1, 1}},
	}
	body3 := SolverPage(session3, info3, SuDozen78097Values)
	err = sameAsResultFile(body3, "TestSolverPage3.html")
	if err != "" {
		t.Errorf("Test Solver 3: got unexpected result body:\n%s:\n%v\n", err, body3)
	}
}

/*

footer

*/

type footerTestcase struct {
	name, version, instance, build, env string
	footer                              string
}

func TestApplicationFooter(t *testing.T) {
	testcases := []footerTestcase{
		{"", "", "", "", "",
			"[" + brandName + " local]"},
		{"susen-staging-pr-30",
			"v29",
			"",
			"ca0fd7123f918d1b6d3e65f3de47d52db09ae068",
			"dev",
			"[susen-staging-pr-30 CI/CD]"},
		{"susen-staging",
			"v29",
			"1vac4117-c29f-4312-521e-ba4d8638c1ac",
			"ca0fd7123f918d1b6d3e65f3de47d52db09ae068",
			"stg",
			"[susen-staging v29 <ca0fd71>]"},
		{"susen-production",
			"v22",
			"1vac4117-c29f-4312-521e-ba4d8638c1ac",
			"ca0fd7123f918d1b6d3e65f3de47d52db09ae068",
			"prd",
			"[susen-production v22 <ca0fd71> (dyno 1vac4117-c29f-4312-521e-ba4d8638c1ac)]"},
	}
	for i, tc := range testcases {
		os.Setenv(applicationNameEnvVar, tc.name)
		os.Setenv(applicationVersionEnvVar, tc.version)
		os.Setenv(applicationInstanceEnvVar, tc.instance)
		os.Setenv(applicationBuildEnvVar, tc.build)
		os.Setenv(applicationEnvEnvVar, tc.env)
		if footer := applicationFooter(); footer != tc.footer {
			t.Errorf("Case %d: got %q, expected %q", i, footer, tc.footer)
		}
	}
}

/*

helpers

*/

func sameAsResultFile(s, fname string) (result string) {
	path := filepath.Join(".", "testdata", fname)
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	fi, err := f.Stat()
	if err != nil {
		panic(err)
	}
	buf := make([]byte, fi.Size())
	n, err := f.Read(buf)
	if n != int(fi.Size()) || err != nil {
		panic(fmt.Errorf("Read of %v bytes failed: %v read, %v", len(s), n, err))
	}
	fs := string(buf)
	mlen, flen, slen := len(fs), len(fs), len(s)
	if slen > flen {
		result = fmt.Sprintf("Result is %d bytes longer: %q\n", slen-flen, s[flen:])
	} else if flen > slen {
		mlen = slen
		result = fmt.Sprintf("Result is missing %d bytes: %q\n", flen-slen, fs[slen:])
	}
	for i := 0; i < mlen; i++ {
		if fs[i] != s[i] {
			j := len(s) - i
			if j > 30 {
				j = 30
			}
			result += fmt.Sprintf("Result differs from expected at offset %d: %q vs. %q",
				i, s[i:i+j], fs[i:i+j])
			return
		}
	}
	return
}
