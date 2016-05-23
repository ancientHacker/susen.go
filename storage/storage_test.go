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
	"time"
)

/*

known-good data for sample session puzzles

*/

const sampleDefaultName = "sample-1"

type testDataEntry struct {
	name     string
	geometry string
	choices  []puzzle.Choice
}

var testData = []testDataEntry{
	{"sample-1", puzzle.StandardGeometryName,
		[]puzzle.Choice{{51, 1}, {41, 8}, {31, 2}}},
	{"sample-7", puzzle.RectangularGeometryName,
		[]puzzle.Choice{{1, 2}, {6, 3}}},
	{"sample-8", puzzle.RectangularGeometryName,
		[]puzzle.Choice{{22, 4}, {23, 5}, {15, 1}, {16, 3}}},
}

/*

setup

*/

// we are creating sessions up the wazoo; make sure they don't
// persist past the end of the test run.
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

/*

connection, sample session

*/

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
	defer Close()

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
}

/*

operations on a single session

*/

var (
	sid = "test session with known name"
)

func TestSessionOpsPhase1(t *testing.T) {
	os.Setenv("DBPREP_PATH", filepath.Join("..", "dbprep"))
	if _, _, err := Connect(); err != nil {
		t.Fatalf("Couldn't connect to storage: %v", err)
	}
	defer Close()

	// load a non-existent session, should be the sample
	ts := LoadSession(sid)
	if ts.active < 0 || ts.active >= len(ts.entries) {
		t.Errorf("No active puzzle (%d)", ts.active)
	}
	if ts.entries[ts.active].PuzzleName != sampleDefaultName {
		t.Errorf("Wrong active puzzle: %q", ts.entries[ts.active].PuzzleName)
	}
	// for each puzzle, switch to them and add the choices
	ids := make([]string, len(testData))
	sums := make([]*puzzle.Summary, len(testData))
	for i, td := range testData {
		ts.SelectPuzzle(td.name)
		if ts.Info.Name != td.name {
			t.Errorf("Wrong selected puzzle: %q", ts.Info.Name)
		}
		ids[i] = ts.Info.PuzzleId
		if len(ts.Info.Choices) > 0 {
			t.Errorf("%s starts with %d choices", ts.Info.Name, len(ts.Info.Choices))
		}
		for j, c := range td.choices {
			_, err := ts.Puzzle.Assign(c)
			if err != nil {
				t.Errorf("Failed assign %d to %s: %v", j, ts.Info.Name, err)
			}
			ts.AddStep(c)
		}
		if count := len(ts.Info.Choices); count != len(td.choices) {
			t.Errorf("Puzzles %s has %d choices", td.name, count)
		}
		if count := len(ts.entries[ts.active].Choices); count != 2*len(td.choices) {
			t.Errorf("Puzzles %s has %d flattened choices", td.name, count)
		}
		sum, err := ts.Puzzle.Summary()
		if err != nil {
			t.Fatalf("Failed to summarize sample puzzle 7: %v", err)
		}
		sums[i] = sum
	}
	// for each puzzle, switch back to them and check the choices
	for i, td := range testData {
		ts.SelectPuzzle(ids[i])
		if ts.Info.Name != td.name {
			t.Errorf("Select puzzle %s by ID gave name %s", td.name, ts.Info.Name)
		}
		if count := len(ts.entries[ts.active].Choices); count != 2*len(td.choices) {
			t.Errorf("Puzzle %s has %d flattened choices", td.name, count)
		}
		sum, err := ts.Puzzle.Summary()
		if err != nil {
			t.Fatalf("Failed to summarize puzzle %s: %v", td.name, err)
		}
		if !reflect.DeepEqual(sum, sums[i]) {
			t.Errorf("Old and new puzzle %s are different", td.name)
		}
	}
}

func TestSessionOpsPhase2(t *testing.T) {
	os.Setenv("DBPREP_PATH", filepath.Join("..", "dbprep"))
	if _, _, err := Connect(); err != nil {
		t.Fatalf("Couldn't connect to storage: %v", err)
	}
	defer Close()

	// the session from the first run should be on the last testData puzzle
	ts := LoadSession(sid)
	if ts.active < 0 || ts.active >= len(ts.entries) {
		t.Errorf("No active puzzle (%d)", ts.active)
	}
	if last := testData[len(testData)-1]; ts.Info.Name != last.name {
		t.Errorf("Selected puzzle is %s but should be %s", ts.Info.Name, last.name)
	}

	// make sure all the counts are right in the various puzzles,
	// then subtract a step and check again.
	for _, td := range testData {
		ts.SelectPuzzle(td.name)
		expected := len(td.choices)
		if count := len(ts.Info.Choices); count != expected {
			t.Errorf("Puzzle %s has %d choices, should be %d", td.name, count, expected)
		}
		ts.RemoveStep()
		expected--
		if count := len(ts.Info.Choices); count != expected {
			t.Errorf("After remove: %d choices, should be %d", count, expected)
		}
		if count := len(ts.entries[ts.active].Choices); count != 2*expected {
			t.Errorf("After remove: %d flattened choices, should be %d", count, 2*expected)
		}
	}
}

func TestSessionOpsPhase3(t *testing.T) {
	os.Setenv("DBPREP_PATH", filepath.Join("..", "dbprep"))
	if _, _, err := Connect(); err != nil {
		t.Fatalf("Couldn't connect to storage: %v", err)
	}
	defer Close()

	// the session from the last run should be on the last testData puzzle
	ts := LoadSession(sid)
	if ts.active < 0 || ts.active >= len(ts.entries) {
		t.Errorf("No active puzzle (%d)", ts.active)
	}
	if last := testData[len(testData)-1]; ts.Info.Name != last.name {
		t.Errorf("Selected puzzle is %s but should be %s", ts.Info.Name, last.name)
	}

	// make sure all the counts are right in the various puzzles,
	// then remove all steps and check again.
	for _, td := range testData {
		ts.SelectPuzzle(td.name)
		expected := len(td.choices) - 1
		if count := len(ts.Info.Choices); count != expected {
			t.Errorf("Puzzle %s has %d choices, should be %d", td.name, count, expected)
		}
		ts.RemoveAllSteps()
		expected = 0
		if count := len(ts.Info.Choices); count != expected {
			t.Errorf("After remove: %d choices, should be %d", count, expected)
		}
		if count := len(ts.entries[ts.active].Choices); count != 2*expected {
			t.Errorf("After remove: %d flattened choices, should be %d", count, 2*expected)
		}
	}
}

func TestSessionOpsPhase4(t *testing.T) {
	os.Setenv("DBPREP_PATH", filepath.Join("..", "dbprep"))
	if _, _, err := Connect(); err != nil {
		t.Fatalf("Couldn't connect to storage: %v", err)
	}
	defer Close()

	// the session from the last run should be on the last testData puzzle
	ts := LoadSession(sid)
	if ts.active < 0 || ts.active >= len(ts.entries) {
		t.Errorf("No active puzzle (%d)", ts.active)
	}
	if last := testData[len(testData)-1]; ts.Info.Name != last.name {
		t.Errorf("Selected puzzle is %s but should be %s", ts.Info.Name, last.name)
	}

	// make sure all the counts are right in the various puzzles,
	for _, td := range testData {
		ts.SelectPuzzle(td.name)
		expected := 0
		if count := len(ts.Info.Choices); count != expected {
			t.Errorf("After remove: %d choices, should be %d", count, expected)
		}
		if count := len(ts.entries[ts.active].Choices); count != 2*expected {
			t.Errorf("After remove: %d flattened choices, should be %d", count, 2*expected)
		}
	}
}

func TestSelectPuzzle(t *testing.T) {
	os.Setenv("DBPREP_PATH", filepath.Join("..", "dbprep"))
	if _, _, err := Connect(); err != nil {
		t.Fatalf("Couldn't connect to storage: %v", err)
	}
	defer Close()

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
	ts.SelectPuzzle("this is not an actual puzzle name or id!!")
}

/*

multiple, concurrent threads

*/

const (
	clientCount = 5
	runCount    = 3
)

type sessionClient struct {
	id       int    // which client this is
	interval int    // the interval, in msec, between calls
	sName    string // the name of the session for this client
}

func TestSessionIsolation(t *testing.T) {
	os.Setenv("DBPREP_PATH", filepath.Join("..", "dbprep"))
	if _, _, err := Connect(); err != nil {
		t.Fatalf("Couldn't connect to storage: %v", err)
	}
	defer Close()

	// make clients
	clients := make([]*sessionClient, clientCount)
	for i := 0; i < clientCount; i++ {
		clients[i] = &sessionClient{
			id:       i + 1,
			interval: (i*17)%60 + 70,
			sName:    fmt.Sprintf("testSessionClient %d", i+1),
		}
	}

	// Each client operates on a separate thread, reloading its
	// session before each operation.  Each selects the same
	// puzzles in the same order and do the same assignments to
	// that puzzle in the same order and then reset the steps on
	// that puzzle.  Any interference between the clients will
	// show up as assignment failures.
	ch := make(chan [2]int, clientCount*runCount)
	start := time.Now()
	for i := 0; i < clientCount; i++ {
		go func(client *sessionClient) {
			for i := 0; i < runCount; i++ {
				for _, td := range testData {
					var ts *Session
					time.Sleep(time.Duration(client.interval) * time.Millisecond)
					ts = LoadSession(client.sName)
					ts.SelectPuzzle(td.name)
					if len(ts.Info.Choices) > 0 {
						t.Fatalf("Client %d: %s starts with %d choices",
							client.id, ts.Info.Name, len(ts.Info.Choices))
					}
					for j, c := range td.choices {
						time.Sleep(time.Duration(client.interval) * time.Millisecond)
						ts = LoadSession(client.sName)
						_, err := ts.Puzzle.Assign(c)
						if err != nil {
							t.Fatalf("Client %d: Failed assign %d to %s: %v",
								client.id, j, ts.Info.Name, err)
						}
						ts.AddStep(c)
					}
					time.Sleep(time.Duration(client.interval) * time.Millisecond)
					ts = LoadSession(client.sName)
					if len(ts.Info.Choices) != len(td.choices) {
						t.Fatalf("Client %d: %s ends with %d choices",
							client.id, ts.Info.Name, len(ts.Info.Choices))
					}
					ts.RemoveAllSteps()
				}
				ch <- [2]int{client.id, i + 1}
			}
		}(clients[i])
	}
	for i := 0; i < clientCount; i++ {
		for j := 0; j < runCount; j++ {
			cr := <-ch
			if testing.Short() {
				fmt.Printf("%v: Client %d finished run %d\n", time.Since(start), cr[0], cr[1])
			}
		}
	}
}
