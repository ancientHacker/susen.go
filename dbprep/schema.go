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
	_ "github.com/ancientHacker/susen.go/Godeps/_workspace/src/github.com/mattes/migrate/driver/postgres"
	"github.com/ancientHacker/susen.go/Godeps/_workspace/src/github.com/mattes/migrate/migrate"
	"os"
)

// figure out the mattes/migrate parameters
func getMigrateParams() (url string, path string) {
	url = os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://localhost/susen?sslmode=disable"
	}
	path = os.Getenv("DBPREP_PATH")
	if path == "" {
		if fi, err := os.Stat("dbprep"); err == nil && fi.IsDir() {
			// running from root directory
			path = "dbprep"
		} else {
			path = "."
		}
	}
	return
}

//SchemaUp creates the database with the right schema
func SchemaUp() error {
	url, path := getMigrateParams()
	if errs, ok := migrate.UpSync(url, path); !ok {
		return fmt.Errorf("Table creation had errors: %v", errs)
	}
	return nil
}

//SchemaDown tears down the database
func SchemaDown() error {
	url, path := getMigrateParams()
	if errs, ok := migrate.DownSync(url, path); !ok {
		return fmt.Errorf("Table deletion had errors: %v", errs)
	}
	return nil
}

//SchemaVersion returns the version of the database
func SchemaVersion() (uint64, error) {
	url, path := getMigrateParams()
	return migrate.Version(url, path)
}
