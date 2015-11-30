package client

import (
	"fmt"
	"github.com/ancientHacker/susen.go/puzzle"
	"os"
	"path/filepath"
	"testing"
)

var (
	rotation4Puzzle1PartialValues = []int{0,
		1, 0, 3, 0,
		0, 3, 0, 1,
		3, 0, 1, 0,
		0, 1, 0, 3,
	}
	oneStarValues = []int{0,
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

func TestSolverPage(t *testing.T) {
	p0, e := puzzle.New(rotation4Puzzle1PartialValues)
	if e != nil {
		t.Fatalf("Failed to create p0: %v", e)
	}
	session0, puzzle0 := "httpx-Test0", "test-0"
	body0 := SolverPage(session0, puzzle0, p0.State())
	if !sameAsResultFile(body0, "TestSolverPage0.html") {
		t.Errorf("Test Solver 0: got unexpected result body:\n%v\n", body0)
	}

	p1, e := puzzle.New(oneStarValues)
	if e != nil {
		t.Fatalf("Failed to create p1: %v", e)
	}
	session1, puzzle1 := "https-Test1", "test-1"
	body1 := SolverPage(session1, puzzle1, p1.State())
	if !sameAsResultFile(body1, "TestSolverPage1.html") {
		t.Errorf("Test Solver 1: got unexpected result body:\n%v\n", body1)
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
