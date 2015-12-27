package client

import (
	"fmt"
	"github.com/ancientHacker/susen.go/puzzle"
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
)

func TestErrorPage(t *testing.T) {
	body := errorPage(fmt.Errorf("Test Error 0"))
	if !sameAsResultFile(body, "TestErrorPage0.html") {
		t.Errorf("Test Error 0: got unexpected result body:\n%v\n", body)
	}
}

func TestHomePage(t *testing.T) {
	session0, puzzle0 := "httpx-Test0", "test-0"
	body := HomePage(session0, puzzle0, nil)
	if !sameAsResultFile(body, "TestHomePage0.html") {
		t.Errorf("Test Home 0: got unexpected result body:\n%v\n", body)
	}
}

func TestSolverPage(t *testing.T) {
	p0, e := puzzle.New(&puzzle.State{
		Geometry:   puzzle.SudokuGeometryName,
		SideLength: 4,
		Values:     rotation4Puzzle1PartialValues,
	})
	if e != nil {
		t.Fatalf("Failed to create p0: %v", e)
	}
	session0, puzzle0 := "httpx-Test0", "test-0"
	state0, e := p0.State()
	if e != nil {
		t.Fatalf("Failed to get state of p0: %v", e)
	}
	body0 := SolverPage(session0, puzzle0, state0)
	if !sameAsResultFile(body0, "TestSolverPage0.html") {
		t.Errorf("Test Solver 0: got unexpected result body:\n%v\n", body0)
	}

	p1, e := puzzle.New(&puzzle.State{
		Geometry:   puzzle.SudokuGeometryName,
		SideLength: 9,
		Values:     oneStarValues,
	})
	if e != nil {
		t.Fatalf("Failed to create p1: %v", e)
	}
	session1, puzzle1 := "https-Test1", "test-1"
	state1, e := p1.State()
	if e != nil {
		t.Fatalf("Failed to get state of p1: %v", e)
	}
	body1 := SolverPage(session1, puzzle1, state1)
	if !sameAsResultFile(body1, "TestSolverPage1.html") {
		t.Errorf("Test Solver 1: got unexpected result body:\n%v\n", body1)
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

func sameAsResultFile(s, fname string) bool {
	path := filepath.Join(".", "testdata", fname)
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	fi, err := f.Stat()
	if err != nil {
		panic(err)
	}
	if fi.Size() != int64(len(s)) {
		return false
	}
	buf := make([]byte, len(s))
	n, err := f.Read(buf)
	if n != len(s) || err != nil {
		panic(fmt.Errorf("Read of %v bytes failed: %v read, %v", len(s), n, err))
	}
	return string(buf) == s
}
