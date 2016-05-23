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
	"encoding/json"
	"fmt"
	"github.com/ancientHacker/susen.go/Godeps/_workspace/src/github.com/garyburd/redigo/redis"
	"github.com/ancientHacker/susen.go/Godeps/_workspace/src/github.com/jackc/pgx"
	"github.com/ancientHacker/susen.go/puzzle"
	"time"
)

/*

working puzzles

*/

// loadActivePuzzle: load the steps for the active puzzle into the cache, load
// all its steps into the cache, and construct its current state
func (s *Session) loadActivePuzzle() {
	if s.countSteps() != len(s.entries[s.active].Choices)+1 {
		s.constructActivePuzzle()
	} else {
		s.loadLastStep()
	}
	s.Info = s.makePuzzleInfo(s.active)
}

// AddStep: add a new step to the active puzzle.
func (s *Session) AddStep(choice puzzle.Choice) {
	// update the session entry, cache, and database
	se := s.entries[s.active]
	se.LastView = time.Now()
	se.Choices = append(se.Choices, int32(choice.Index), int32(choice.Value))
	s.cacheUpdateEntry(s.active)
	s.databaseUpdateEntry(s.active)
	// update the state of the session and the step cache
	s.addStep()
	s.Info = s.makePuzzleInfo(s.active)
}

// RemoveStep: remove the last step and restore the prior step in
// the active puzzle.
func (s *Session) RemoveStep() {
	// update the session entry, cache, and database
	se := s.entries[s.active]
	if len(se.Choices) == 0 {
		// nothing to do
		return
	}
	se.LastView = time.Now()
	se.Choices = se.Choices[0 : len(se.Choices)-2]
	s.cacheUpdateEntry(s.active)
	s.databaseUpdateEntry(s.active)
	// update the state of the session and the step cache
	s.removeStep()
	s.loadLastStep()
	s.Info = s.makePuzzleInfo(s.active)
}

// RemoveAllSteps: remove all the steps from the current puzzle
// and restore it to its starting point.
func (s *Session) RemoveAllSteps() {
	// update the session entry, cache, and database
	se := s.entries[s.active]
	if len(se.Choices) == 0 {
		// nothing to do
		return
	}
	se.LastView = time.Now()
	se.Choices = nil
	s.cacheUpdateEntry(s.active)
	s.databaseUpdateEntry(s.active)
	// update the state of the step cache
	s.loadActivePuzzle()
	s.Info = s.makePuzzleInfo(s.active)
}

/*

puzzle info

*/

// A PuzzleInfo is the session's exported form of the puzzles in
// the session.  It merges the sessionEntry data with the
// puzzleEntry shape data.
type PuzzleInfo struct {
	PuzzleId   string          // unique ID for this puzzle
	Name       string          // user-facing name of the puzzle
	Geometry   string          // puzzle geometry
	SideLength int             // puzzle size
	Choices    []puzzle.Choice // choices made for this puzzle
	Remaining  int             // number of remaining choices to make
	LastView   time.Time       // time when the puzzle was last viewed
}

// makePuzzleInfo - make a PuzzleInfo from a sessionEntry
func (s *Session) makePuzzleInfo(index int) *PuzzleInfo {
	se := s.entries[index]
	choices := make([]puzzle.Choice, len(se.Choices)/2)
	for i := range choices {
		choices[i] = puzzle.Choice{Index: int(se.Choices[2*i]), Value: int(se.Choices[2*i+1])}
	}
	pe := loadPuzzleEntry(se.PuzzleId)
	return &PuzzleInfo{
		PuzzleId:   se.PuzzleId,
		Name:       se.PuzzleName,
		Geometry:   pe.Geometry,
		SideLength: int(pe.SideLength),
		Choices:    choices,
		Remaining:  countZeroes(pe.Values) - len(choices),
		LastView:   se.LastView,
	}
}

// compute the number of empty squares
func countZeroes(vals []int32) (count int) {
	for _, v := range vals {
		if v == 0 {
			count++
		}
	}
	return
}

// sorting of info sequences by puzzle name
type ByName []*PuzzleInfo

func (pi ByName) Len() int           { return len(pi) }
func (pi ByName) Swap(i, j int)      { pi[i], pi[j] = pi[j], pi[i] }
func (pi ByName) Less(i, j int) bool { return pi[i].Name < pi[j].Name }

// sorting of info sequences by last viewed time
type ByLatestView []*PuzzleInfo

func (pi ByLatestView) Len() int           { return len(pi) }
func (pi ByLatestView) Swap(i, j int)      { pi[i], pi[j] = pi[j], pi[i] }
func (pi ByLatestView) Less(i, j int) bool { return pi[i].LastView.After(pi[j].LastView) }

// sorting of info sequences by attempted solution & last viewed time
type ByLatestSolutionView []*PuzzleInfo

func (pi ByLatestSolutionView) Len() int      { return len(pi) }
func (pi ByLatestSolutionView) Swap(i, j int) { pi[i], pi[j] = pi[j], pi[i] }
func (pi ByLatestSolutionView) Less(i, j int) bool {
	return !(len(pi[i].Choices) == 0 && len(pi[j].Choices) > 0) &&
		((len(pi[i].Choices) > 0 && len(pi[j].Choices) == 0) ||
			pi[i].LastView.After(pi[j].LastView))
}

/*

puzzle entries

*/

// A puzzleEntry represents the stored form of a starting-point
// or solution puzzle. It is JSON serializable so it can go into
// the cache as well as the database.
type puzzleEntry struct {
	PuzzleId   string // puzzle Signature
	Geometry   string
	SideLength int32
	Values     []int32
}

// loadPuzzleEntry first checks the cache, then the database, to
// find the puzzle's entry.  If it loads from the database, it
// caches the result.  Panics if there is no such stored entry.
func loadPuzzleEntry(id string) *puzzleEntry {
	pe := &puzzleEntry{PuzzleId: id}
	if pe.cacheLoad() {
		return pe
	}
	// cache miss, load from database and save to cache
	pe.databaseLoad()
	pe.cacheInsert()
	return pe
}

// makePuzzle: make the puzzle described in a puzzle entry
func (pe *puzzleEntry) makePuzzle() *puzzle.Puzzle {
	values := make([]int, len(pe.Values))
	for i, v := range pe.Values {
		values[i] = int(v)
	}
	p, e := puzzle.New(&puzzle.Summary{
		Geometry:   pe.Geometry,
		SideLength: int(pe.SideLength),
		Values:     values,
	})
	if e != nil {
		panic(fmt.Errorf("Failed to create puzzle %q: %v", pe.PuzzleId, e))
	}
	return p
}

// key: compute the cache key for a puzzleEntry.
func (pe *puzzleEntry) key() string {
	return "PID:" + pe.PuzzleId
}

// cacheLoad: load an already cached puzzle entry.  Returns
// whether the entry was found in the cache.
func (pe *puzzleEntry) cacheLoad() bool {
	var bytes []byte
	body := func(tx redis.Conn) (err error) {
		bytes, err = redis.Bytes(tx.Do("GET", pe.key()))
		if err == redis.ErrNil {
			return nil
		}
		if err != nil {
			err = fmt.Errorf("Cache failure loading puzzleEntry %q: %v", pe.PuzzleId, err)
		}
		return
	}
	rdExecute(body)
	if len(bytes) == 0 {
		return false
	}
	var spe *puzzleEntry
	err := json.Unmarshal(bytes, &spe)
	if err != nil {
		panic(fmt.Errorf("Failed to unmarshal puzzleEntry %q: %v", pe.PuzzleId, err))
	}
	if spe.PuzzleId != pe.PuzzleId {
		panic(fmt.Errorf("Cached puzzleEntry (id: %q) found for puzzle %q!",
			spe.PuzzleId, pe.PuzzleId))
	}
	*pe = *spe
	return true
}

// databaseLoad: load a puzzle entry from the database.  Panics
// if there is no saved entry with the given id.
func (pe *puzzleEntry) databaseLoad() {
	body := func(tx *pgx.Tx) error {
		row := tx.QueryRow(
			"SELECT geometry, sideLength, valueList FROM puzzles "+
				"WHERE puzzleId = $1", pe.PuzzleId)
		if err := row.Scan(&pe.Geometry, &pe.SideLength, &pe.Values); err != nil {
			return fmt.Errorf("Failure looking up puzzle %q: %v", pe.PuzzleId, err)
		}
		return nil
	}
	pgExecute(body)
}

// cacheInsert: insert a puzzle entry into the cache. Replaces
// any existing entry with the same id.
func (pe *puzzleEntry) cacheInsert() {
	bytes, e := json.Marshal(pe)
	if e != nil {
		panic(fmt.Errorf("Failed to marshal puzzleEntry %q: %v", pe.PuzzleId, e))
	}
	body := func(tx redis.Conn) (err error) {
		_, err = tx.Do("SET", pe.key(), bytes)
		if err != nil {
			err = fmt.Errorf("Cache failure saving puzzle entry %q: %v", pe.PuzzleId, err)
		}
		return
	}
	rdExecute(body)
}

// databaseInsert: insert a new puzzle entry into the database.
// Panics if there is already a saved entry with the given id.
func (pe *puzzleEntry) databaseInsert() {
	body := func(tx *pgx.Tx) (err error) {
		_, err = tx.Exec(
			"INSERT INTO puzzles (puzzleId, geometry, sideLength, valueList, created) "+
				"VALUES ($1, $2, $3, $4, $5)",
			pe.PuzzleId, pe.Geometry, pe.SideLength, pe.Values, time.Now())
		if err != nil {
			err = fmt.Errorf("Database error saving puzzle entry %q: %v", pe.PuzzleId, err)
		}
		return
	}
	pgExecute(body)
}

/*

solution steps

*/

// stepsKey: returns the cache key for the active puzzle's step array
func (s *Session) stepsKey() string {
	return s.key() + ":PID:" + s.entries[s.active].PuzzleId + ":Steps"
}

// countSteps: return the number of active puzzle steps in the cache
func (s *Session) countSteps() int {
	var stepCount int
	body := func(tx redis.Conn) (err error) {
		stepCount, err = redis.Int(tx.Do("LLEN", s.stepsKey()))
		if err != nil {
			err = fmt.Errorf("Cache failure reading step length: %v", err)
		}
		return
	}
	rdExecute(body)
	return stepCount
}

// loadLastStep: load the last cached step to the active puzzle
func (s *Session) loadLastStep() {
	var bytes []byte
	body := func(tx redis.Conn) (err error) {
		bytes, err = redis.Bytes(tx.Do("LINDEX", s.stepsKey(), -1))
		if err != nil {
			err = fmt.Errorf("Cache failure reading last step: %v", err)
		}
		return
	}
	rdExecute(body)
	s.Puzzle = s.unmarshalPuzzle(bytes)
}

// addStep: add a new step to the cache
func (s *Session) addStep() {
	bytes := s.marshalPuzzle(s.Puzzle)
	body := func(tx redis.Conn) (err error) {
		_, err = tx.Do("RPUSH", s.stepsKey(), bytes)
		if err != nil {
			err = fmt.Errorf("Cache failure saving step summary: %v", err)
		}
		return
	}
	rdExecute(body)
}

// removeStep: remove a step from the cache
func (s *Session) removeStep() {
	body := func(tx redis.Conn) (err error) {
		_, err = tx.Do("LTRIM", s.stepsKey(), 0, -2)
		if err != nil {
			err = fmt.Errorf("Cache failure removing cached step: %v", err)
		}
		return
	}
	rdExecute(body)
}

// constructActivePuzzle: build the active puzzle from its
// session entry choices, caching the steps as we go.
func (s *Session) constructActivePuzzle() {
	// first clear the step cache
	body := func(tx redis.Conn) (err error) {
		_, err = tx.Do("DEL", s.stepsKey())
		if err != nil {
			err = fmt.Errorf("Cache failure clearing steps: %v", err)
		}
		return
	}
	rdExecute(body)

	// then create and insert the sequence of steps
	choices := s.entries[s.active].Choices
	s.Puzzle = loadPuzzleEntry(s.entries[s.active].PuzzleId).makePuzzle()
	s.addStep()
	for j := 0; j < len(choices); j = j + 2 {
		choice := puzzle.Choice{Index: int(choices[j]), Value: int(choices[j+1])}
		_, err := s.Puzzle.Assign(choice)
		if err != nil {
			panic(fmt.Errorf("Failure assigning to puzzle: %v", err))
		}
		s.addStep()
	}
}

// marshalPuzzle: serialize a puzzle as JSON
func (s *Session) marshalPuzzle(puzzle *puzzle.Puzzle) []byte {
	summary, err := puzzle.Summary()
	if err != nil {
		panic(fmt.Errorf("Failed to create summary for puzzle: %v", err))
	}
	bytes, err := json.Marshal(summary)
	if err != nil {
		panic(fmt.Errorf("Failed to marshal summary %+v: %v", *summary, err))
	}
	return bytes
}

// unmarshalPuzzle: unserialize JSON to a puzzle
func (s *Session) unmarshalPuzzle(bytes []byte) *puzzle.Puzzle {
	var summary *puzzle.Summary
	err := json.Unmarshal(bytes, &summary)
	if err != nil {
		panic(fmt.Errorf("Failed to unmarshal summary: %v", err))
	}
	puzzle, err := puzzle.New(summary)
	if err != nil {
		panic(fmt.Errorf("Failed to create puzzle from summary: %v", err))
	}
	return puzzle
}
