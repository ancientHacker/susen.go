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
)

func EnsureData() error {
	inVersion, err := SchemaVersion()
	if err != nil {
		return fmt.Errorf("Couldn't get initial data schema version: %v", err)
	}
	if err := SchemaUp(); err != nil {
		return fmt.Errorf("Couldn't install data schema: %v", err)
	}
	outVersion, err := SchemaVersion()
	if err != nil {
		return fmt.Errorf("Couldn't get final data schema version: %v", err)
	}
	if outVersion == 0 {
		return fmt.Errorf("Database schema still at version 0, shouldn't be.")
	}
	if inVersion != outVersion {
		if err := DataUp(); err != nil {
			return fmt.Errorf("Couldn't load data: %v", err)
		}
	}
	return nil
}

func RemoveData() error {
	// tear down existing database
	version, err := SchemaVersion()
	if err != nil {
		return fmt.Errorf("Couldn't get initial data schema version: %v", err)
	}
	if version > 0 {
		if err := SchemaDown(); err != nil {
			return fmt.Errorf("Couldn't remove tables: %v", err)
		}
	}
	return nil
}

func ReinitializeAll() error {
	// clear cache
	if err := ClearCache(); err != nil {
		return fmt.Errorf("Couldn't clear cache: %v", err)
	}
	// clear database
	if err := RemoveData(); err != nil {
		return fmt.Errorf("Couldn't clear database: %v", err)
	}
	// reload database
	if err := EnsureData(); err != nil {
		return fmt.Errorf("Couldn't load database: %v", err)
	}
	return nil
}
