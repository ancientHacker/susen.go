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

// Clear and re-initialize the susen storage system
package main

import (
	"fmt"
	"github.com/ancientHacker/susen.go/dbprep"
	"log"
)

func main() {
	log.Printf("Removing existing data storage and cache...")
	if err := clearStorage(); err != nil {
		log.Fatalf("Couldn't clear storage: %v", err)
	}
	log.Printf("Database re-initialized.")
}

func clearStorage() error {
	// clear cache
	if err := dbprep.ClearCache(); err != nil {
		return fmt.Errorf("Couldn't clear cache: %v", err)
	}

	// tear down existing database
	version, err := dbprep.SchemaVersion()
	if err != nil {
		return fmt.Errorf("Couldn't get initial data schema version: %v", err)
	}
	if version > 0 {
		if err := dbprep.SchemaDown(); err != nil {
			return fmt.Errorf("Couldn't remove database: %v", err)
		}
	}
	if err := dbprep.SchemaUp(); err != nil {
		return fmt.Errorf("Couldn't get data schema version: %v", err)
	}
	version, err = dbprep.SchemaVersion()
	if err != nil {
		return fmt.Errorf("Couldn't get upgraded data schema version: %v", err)
	}
	if version == 0 {
		return fmt.Errorf("Database schema still at version 0, shouldn't be.")
	}
	if err := dbprep.DataUp(); err != nil {
		return fmt.Errorf("Couldn't load base data: %v", err)
	}
	return nil
}
