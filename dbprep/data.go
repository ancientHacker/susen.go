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

package dbprep

import (
	"fmt"
	"github.com/ancientHacker/susen.go/Godeps/_workspace/src/github.com/jackc/pgx"
	"github.com/ancientHacker/susen.go/puzzle"
	"os"
	"time"
)

/*

entries

*/

type dataFunction func(*pgx.Tx) error

var (
	upFunctions = []dataFunction{
		insertSamples,
	}
	downFunctions = []dataFunction{
		deleteSamples,
	}
)

// DataUp: load the sample data into the database.  You should do
// this after you get the schema up!
func DataUp() error {
	return applyFunctions(upFunctions)
}

// DataDown: remove the sample data from the database.  You
// should do this before you tear the schema down!
func DataDown() error {
	return applyFunctions(downFunctions)
}

// apply dataFunctions to the database.  Each is applied in a
// separate transaction, so later ones can rely on the effect of
// earlier ones having been committed.
func applyFunctions(fns []dataFunction) error {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://localhost/susen?sslmode=disable"
	}

	// open the database, defer the close
	cfg, err := pgx.ParseURI(url)
	if err != nil {
		return err
	}
	conn, err := pgx.Connect(cfg)
	if err != nil {
		return err
	}
	defer conn.Close()

	// helper that runs each function inside a transaction, and
	// ensures that any problems are rolled back.
	runFunc := func(fn dataFunction) error {
		tx, err := conn.Begin()
		if err != nil {
			return err
		}
		defer func() {
			if e := recover(); e != nil {
				tx.Rollback()
				panic(e)
			}
		}()
		if err := fn(tx); err != nil {
			tx.Rollback()
			return err
		}
		return tx.Commit()
	}

	// run the functions
	for _, fn := range fns {
		if err := runFunc(fn); err != nil {
			return fmt.Errorf("%v failed: %v", fn, err)
		}
	}
	return nil
}

/*

insert sample puzzles in a special session

*/

const SampleSessionName = "Sūsen Sample Session - not a user session"

var (
	samplePuzzles = []*puzzle.Summary{
		&puzzle.Summary{
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
		&puzzle.Summary{
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
		&puzzle.Summary{
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
		&puzzle.Summary{
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
		&puzzle.Summary{
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
		&puzzle.Summary{
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
		&puzzle.Summary{
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
		&puzzle.Summary{
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
		&puzzle.Summary{
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
		&puzzle.Summary{
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
	sampleHashes []string // see init
	sampleNames  []string // see init
)

// initialize the hashes and names from the sample puzzles
func init() {
	sampleHashes = make([]string, len(samplePuzzles))
	for i := range samplePuzzles {
		hash, err := samplePuzzles[i].Hash()
		if err != nil {
			panic(fmt.Errorf("Can't happen! Sample summary %d is invalid!", i))
		}
		sampleHashes[i] = string(hash)
	}
	sampleNames = make([]string, len(samplePuzzles))
	for i := range samplePuzzles {
		sampleNames[i] = fmt.Sprintf("sample-%d", i+1)
	}
}

// Create and insert the sample puzzles and sample session
func insertSamples(tx *pgx.Tx) error {
	// idempotency: if the sample session already exists, we are done
	var count int64
	row := tx.QueryRow("SELECT COUNT(*) FROM sessions "+
		"WHERE sessionId = $1", SampleSessionName)
	if err := row.Scan(&count); err != nil {
		return fmt.Errorf("Database error looking for session %q: %v", SampleSessionName, err)
	}
	if count > 0 {
		return nil
	}

	// get the timestamp of this load
	now := time.Now()

	// first save the puzzles
	for i, sum := range samplePuzzles {
		values := make([]int32, len(sum.Values))
		for i, v := range sum.Values {
			values[i] = int32(v) // use 4-byte ints in database
		}
		_, err := tx.Exec(
			"INSERT INTO puzzles (puzzleId, geometry, sideLength, valueList, created) "+
				"VALUES ($1, $2, $3, $4, $5)",
			sampleHashes[i], sum.Geometry, int32(sum.SideLength), values, now)
		if err != nil {
			return fmt.Errorf("Database error saving sample puzzle %d: %v", i, err)
		}
	}

	// next save the session
	_, err := tx.Exec(
		"INSERT INTO sessions (sessionId, created, updated) "+
			"VALUES ($1, $2, $3)",
		SampleSessionName, now, now)
	if err != nil {
		return fmt.Errorf("Database error saving sample session: %v", err)
	}

	// next save the session entries
	for i := range samplePuzzles {
		_, err := tx.Exec(
			"INSERT INTO sessionPuzzles (sessionId, puzzleId, puzzleName, lastWorked) "+
				"VALUES ($1, $2, $3, $4)",
			SampleSessionName, sampleHashes[i], sampleNames[i], now)
		if err != nil {
			return fmt.Errorf("Database error saving sample session puzzle %d: %v", i, err)
		}
	}

	return nil
}

// Delete the common puzzles
func deleteSamples(tx *pgx.Tx) error {
	// first remove the puzzle summaries from the database
	_, err := tx.Exec(
		"DELETE from sessionPuzzles where sessionId = $1", SampleSessionName)
	if err != nil {
		return fmt.Errorf("Database error deleting sample session: %v", err)
	}

	// then remove the session
	_, err = tx.Exec(
		"DELETE from sessions where sessionId = $1", SampleSessionName)
	if err != nil {
		return fmt.Errorf("Database error deleting sample session: %v", err)
	}

	// then remove the puzzles themselves
	for i, hash := range sampleHashes {
		_, err := tx.Exec(
			"DELETE from puzzles where puzzleId = $1", hash)
		if err != nil {
			return fmt.Errorf("Database error deleting sample puzzle %d: %v", i, err)
		}
	}
	return nil
}
