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
	"github.com/ancientHacker/susen.go/puzzle"
	"log"
)

// Puzzle collections map a puzzle ID to a puzzle summary
type NamedSummaries map[string]*puzzle.Summary

/*

The default puzzle is alway available

*/

func DefaultPuzzleID() string {
	return "default"
}

func DefaultPuzzleSummary() *puzzle.Summary {
	return &puzzle.Summary{
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
		},
	}
}

/*

Summaries particular to a session

*/

func LoadSessionSummaries(sessionId string) NamedSummaries {
	result := make(map[string]*puzzle.Summary)
	body := func() error {
		rows, err := pgdb.Query(
			"SELECT puzzleId, summary FROM puzzles WHERE sessionID = $1", sessionId)
		if err != nil {
			log.Printf("Failed to fetch common puzzles: %v", err)
			return err
		}
		for rows.Next() {
			var puzzleId, summaryJson string
			if err := rows.Scan(&puzzleId, &summaryJson); err != nil {
				log.Printf("Failed to scan puzzles row: %v", err)
				return err
			}
			result[puzzleId] = unmarshalSummary(puzzleId, summaryJson)
		}
		return nil
	}
	pgExecute(body)
	return result
}

/*

Summaries common to all sessions

*/

var (
	commonSummaries NamedSummaries
)

func CommonSummaries() NamedSummaries {
	if commonSummaries == nil {
		commonSummaries = LoadSessionSummaries("common")
	}
	return commonSummaries
}

/*

serialization of summaries into and out of the database

*/

// marshalSummary - get JSON string for the current puzzle
func marshalSummary(id string, summary *puzzle.Summary) string {
	bytes, err := json.Marshal(summary)
	if err != nil {
		log.Printf("Failed to marshal summary of %v as JSON: %v", id, err)
		panic(err)
	}
	return string(bytes)
}

// unmarshalSummary - get puzzle for the saved puzzle
func unmarshalSummary(id string, summaryJson string) *puzzle.Summary {
	var summary *puzzle.Summary
	err := json.Unmarshal([]byte(summaryJson), &summary)
	if err != nil {
		log.Printf("Failed to unmarshal saved JSON of %s: %v", id, err)
		panic(err)
	}
	return summary
}
