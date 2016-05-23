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
	"flag"
	"fmt"
	"github.com/ancientHacker/susen.go/dbprep"
	"log"
)

var (
	clear      = flag.Bool("clear", false, "Clear but don't reload the data")
	initialize = flag.Bool("initialize", false, "Initialize but don't clear the data")
)

func main() {
	flag.Parse()
	if flag.NArg() > 0 {
		flag.PrintDefaults()
		log.Fatalf("Usage error.")
	}
	if err := doit(); err != nil {
		log.Fatalf("Fatal error: %v", err)
	}
}

func doit() error {
	log.Printf("Removing existing data storage and cache...")
	if err := dbprep.ClearCache(); err != nil {
		return fmt.Errorf("Couldn't clear cache: %v", err)
	}
	if !*initialize {
		if err := dbprep.RemoveData(); err != nil {
			return fmt.Errorf("Couldn't clear database storage: %v", err)
		}
	}
	if !*clear {
		log.Printf("Intializing data storage...")
		if err := dbprep.EnsureData(); err != nil {
			return fmt.Errorf("Couldn't clear storage: %v", err)
		}
	}
	log.Printf("Done.")
	return nil
}
