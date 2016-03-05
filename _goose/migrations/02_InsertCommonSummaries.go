package main

import (
	"database/sql"
	"encoding/json"
	"github.com/ancientHacker/susen.go/puzzle"
	"log"
)

var (
	puzzleSummaries = map[string]*puzzle.Summary{
		"standard-1": &puzzle.Summary{
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
		"standard-2": &puzzle.Summary{
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
		"standard-3": &puzzle.Summary{
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
		"standard-4": &puzzle.Summary{
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
		"standard-5": &puzzle.Summary{
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
		"rectangular-1": &puzzle.Summary{
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
		"rectangular-2": &puzzle.Summary{
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
		"rectangular-3": &puzzle.Summary{
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
		"rectangular-4": &puzzle.Summary{
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
)

// Up is executed when this migration is applied
func Up_2(txn *sql.Tx) {
	// save the puzzle summaries in the database
	for key, val := range puzzleSummaries {
		bytes, err := json.Marshal(val)
		if err != nil {
			log.Printf("Failed to marshal summary of common puzzle %v: %v", key, err)
			panic(err)
		}
		_, err = txn.Exec(
			"INSERT INTO puzzles (sessionId, puzzleId, summary) VALUES ($1, $2, $3)",
			"common", key, string(bytes))
		if err != nil {
			log.Printf("Failed to insert summary of common puzzle %v: %v", key, err)
			panic(err)
		}
	}
}

// Down is executed when this migration is rolled back
func Down_2(txn *sql.Tx) {
	// remove the puzzle summaries from the database
	for key := range puzzleSummaries {
		_, err := txn.Exec(
			"DELETE from puzzles where sessionId = $1 and puzzleId = $2",
			"common", key)
		if err != nil {
			log.Printf("Failed to delete summary of common puzzle %v: %v", key, err)
			panic(err)
		}
	}
}
