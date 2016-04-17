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
	"fmt"
	"github.com/ancientHacker/susen.go/dbprep"
	"github.com/ancientHacker/susen.go/puzzle"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	os.Setenv("DBPREP_PATH", filepath.Join("..", "dbprep"))
	if err := dbprep.ReinitializeAll(); err != nil {
		panic(fmt.Errorf("Failed to reinitialize data at startup: %v", err))
	}
	defer func(code int) {
		if code == 0 {
			if err := dbprep.ReinitializeAll(); err != nil {
				panic(fmt.Errorf("Failed to reinitialize data at teardown: %v", err))
			}
		}
		os.Exit(code)
	}(m.Run())
}

func TestConnect(t *testing.T) {
	os.Setenv("DBPREP_PATH", filepath.Join("..", "dbprep"))
	if cid, dbid, err := Connect(); err != nil {
		t.Errorf("Couldn't connect to storage: %v", err)
	} else if cid != rdUrl || dbid != pgUrl {
		t.Errorf("Connected to wrong cache (%s) or wrong database (%s)", cid, dbid)
	}
	Close()
}

func TestSampleSession(t *testing.T) {
	os.Setenv("DBPREP_PATH", filepath.Join("..", "dbprep"))
	if _, _, err := Connect(); err != nil {
		t.Fatalf("Couldn't connect to storage: %v", err)
	}
	ss := loadSampleSession()
	if len(ss.entries) == 0 {
		t.Errorf("No sample session entries")
	}
	ts := &Session{sid: "Test Session 1", active: -1}
	ts.initializeFromSample()
	if len(ts.entries) != len(ss.entries) {
		t.Fatalf("Test session has %d entries, should be %d", len(ts.entries), len(ss.entries))
	}
	if !reflect.DeepEqual(ts.entries, ss.entries) {
		t.Errorf("Test session entries differ from sample session entries:")
		for i := range ts.entries {
			if !reflect.DeepEqual(ts.entries[i], ss.entries[i]) {
				t.Errorf("Sample %d: Got: %+v, Expected:%+v",
					i, *ts.entries[i], *ss.entries[i])
			}
		}
	}
	Close()
}

var (
	sid                  = "test session with known name"
	puzzle1Id, puzzle7Id string
)

func TestSessionOpsPhase1(t *testing.T) {
	os.Setenv("DBPREP_PATH", filepath.Join("..", "dbprep"))
	if _, _, err := Connect(); err != nil {
		t.Fatalf("Couldn't connect to storage: %v", err)
	}

	// load a non-existent session, should be the sample
	ts := LoadSession(sid)
	if ts.active < 0 || ts.active >= len(ts.entries) {
		t.Errorf("No active puzzle (%d)", ts.active)
	}
	if ts.entries[ts.active].PuzzleName != "sample-1" {
		t.Errorf("Wrong active puzzle: %q", ts.entries[ts.active].PuzzleName)
	}
	puzzle1Id = ts.entries[ts.active].PuzzleId
	// find and switch to sample-7 (rectangular)
	for _, se := range ts.entries {
		if se.PuzzleName == "sample-7" {
			puzzle7Id = se.PuzzleId
			ts.SelectPuzzle(se.PuzzleId)
		}
	}
	if ts.entries[ts.active].PuzzleName != "sample-7" {
		t.Errorf("Wrong selected puzzle: %q", ts.entries[ts.active].PuzzleName)
	}
	if len(ts.Info.Choices) > 0 {
		t.Errorf("Sample puzzle 7 has %d choices", len(ts.Info.Choices))
	}
	choice0 := puzzle.Choice{Index: 1, Value: 2}
	_, err := ts.Puzzle.Assign(choice0)
	if err != nil {
		t.Errorf("Failed first assign to sample-7: %v", err)
	}
	ts.AddStep(choice0)
	choice1 := puzzle.Choice{Index: 6, Value: 3}
	_, err = ts.Puzzle.Assign(choice1)
	if err != nil {
		t.Errorf("Failed second assign to sample-7: %v", err)
	}
	ts.AddStep(choice1)
	if count := len(ts.entries[ts.active].Choices); count != 4 {
		t.Errorf("Sample puzzle 7 has %d flattened choices", count)
	}
	sum1, err := ts.Puzzle.Summary()
	if err != nil {
		t.Fatalf("Failed to summarize sample puzzle 7: %v", err)
	}
	ts.SelectPuzzle(puzzle1Id)
	if count := len(ts.entries[ts.active].Choices); count != 0 {
		t.Errorf("Sample puzzle 1 has %d flattened choices", count)
	}
	ts.SelectPuzzle(puzzle7Id)
	if count := len(ts.entries[ts.active].Choices); count != 4 {
		t.Errorf("Sample puzzle 7 has %d flattened choices", count)
	}
	if len(ts.Info.Choices) != 2 {
		t.Errorf("Sample puzzle 7 has %d choices", len(ts.Info.Choices))
	}
	sum2, err := ts.Puzzle.Summary()
	if err != nil {
		t.Fatalf("Failed to summarize second sample puzzle 7: %v", err)
	}
	if !reflect.DeepEqual(sum1, sum2) {
		t.Errorf("Old and new puzzle 7 are different")
	}
}

func TestSessionOpsPhase2(t *testing.T) {
	os.Setenv("DBPREP_PATH", filepath.Join("..", "dbprep"))
	if _, _, err := Connect(); err != nil {
		t.Fatalf("Couldn't connect to storage: %v", err)
	}

	// the session from the first run
	ts := LoadSession(sid)
	if ts.active < 0 || ts.active >= len(ts.entries) {
		t.Errorf("No active puzzle (%d)", ts.active)
	}
	if ts.entries[ts.active].PuzzleName != "sample-7" {
		t.Errorf("Wrong selected puzzle: %q", ts.entries[ts.active].PuzzleName)
	}
	if len(ts.Info.Choices) != 2 {
		t.Errorf("Sample puzzle 7 has %d choices", len(ts.Info.Choices))
	}
	ts.SelectPuzzle(puzzle1Id)
	if count := len(ts.entries[ts.active].Choices); count != 0 {
		t.Errorf("Sample puzzle 1 has %d flattened choices", count)
	}
	ts.SelectPuzzle(puzzle7Id)
	if count := len(ts.entries[ts.active].Choices); count != 4 {
		t.Errorf("Sample puzzle 7 has %d flattened choices", count)
	}
}

func TestSessionOpsPhase3(t *testing.T) {
	os.Setenv("DBPREP_PATH", filepath.Join("..", "dbprep"))
	if _, _, err := Connect(); err != nil {
		t.Fatalf("Couldn't connect to storage: %v", err)
	}

	// the session from the first run
	ts := LoadSession(sid)
	if ts.active < 0 || ts.active >= len(ts.entries) {
		t.Errorf("No active puzzle (%d)", ts.active)
	}
	if ts.entries[ts.active].PuzzleName != "sample-7" {
		t.Errorf("Wrong selected puzzle: %q", ts.entries[ts.active].PuzzleName)
	}
	if len(ts.Info.Choices) != 2 {
		t.Errorf("Sample puzzle 7 has %d choices", len(ts.Info.Choices))
	}
	ts.RemoveStep()
	if count := len(ts.entries[ts.active].Choices); count != 2 {
		t.Errorf("After remove, there are %d flattened choices", count)
	}
	if len(ts.Info.Choices) != 1 {
		t.Errorf("After remove, there are %d choices", len(ts.Info.Choices))
	}
	ts.RemoveAllSteps()
	if count := len(ts.entries[ts.active].Choices); count != 0 {
		t.Errorf("After remove, there are %d flattened choices", count)
	}
	if len(ts.Info.Choices) != 0 {
		t.Errorf("After remove, there are %d choices", len(ts.Info.Choices))
	}
}

func TestSessionOpsPhase4(t *testing.T) {
	os.Setenv("DBPREP_PATH", filepath.Join("..", "dbprep"))
	if _, _, err := Connect(); err != nil {
		t.Fatalf("Couldn't connect to storage: %v", err)
	}

	// the session from the first run
	ts := LoadSession(sid)
	if ts.active < 0 || ts.active >= len(ts.entries) {
		t.Errorf("No active puzzle (%d)", ts.active)
	}
	if ts.entries[ts.active].PuzzleName != "sample-7" {
		t.Errorf("Wrong selected puzzle: %q", ts.entries[ts.active].PuzzleName)
	}
	if len(ts.Info.Choices) != 0 {
		t.Errorf("Sample puzzle 7 has %d choices", len(ts.Info.Choices))
	}
	ts.SelectPuzzle(puzzle1Id)
	if count := len(ts.entries[ts.active].Choices); count != 0 {
		t.Errorf("Sample puzzle 1 has %d flattened choices", count)
	}
	ts.SelectPuzzle(puzzle7Id)
	if count := len(ts.entries[ts.active].Choices); count != 0 {
		t.Errorf("Sample puzzle 7 has %d flattened choices", count)
	}
}

func TestSelectPuzzle(t *testing.T) {
	os.Setenv("DBPREP_PATH", filepath.Join("..", "dbprep"))
	if _, _, err := Connect(); err != nil {
		t.Fatalf("Couldn't connect to storage: %v", err)
	}

	ts := LoadSession(sid)
	ts.SelectPuzzle("SAMPLE-3")
	if ts.Info.Name != "sample-3" {
		t.Errorf("Failed to select uppercase puzzle name!")
	}
	ts.SelectPuzzle(strings.ToLower(ts.Info.PuzzleId))
	if ts.Info.Name != "sample-3" {
		t.Errorf("Failed to select lowercase puzzle id!")
	}
	defer func() {
		if recover() == nil {
			t.Errorf("Didn't panic on select of non-puzzle")
		}
	}()
	ts.SelectPuzzle("this is not an actual puzzlen name or id!!")
}
