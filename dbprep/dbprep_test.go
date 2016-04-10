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
	"testing"
)

func TestClearCache(t *testing.T) {
	if err := ClearCache(); err != nil {
		t.Errorf("Couldn't clear cache: %v", err)
	}
}

func TestSchemaUpDown(t *testing.T) {
	if err := SchemaUp(); err != nil {
		t.Errorf("Schema up failed: %v", err)
	}
	if err := SchemaDown(); err != nil {
		t.Errorf("Schema down failed: %v", err)
	}
}

func TestSchemaDoubleUp(t *testing.T) {
	if err := SchemaUp(); err != nil {
		t.Errorf("Schema up failed: %v", err)
	}
	if err := SchemaUp(); err != nil {
		t.Errorf("Schema 2nd up failed: %v", err)
	}
	if err := SchemaDown(); err != nil {
		t.Errorf("Schema down failed: %v", err)
	}
}

func TestSchemaDoubleDown(t *testing.T) {
	if err := SchemaUp(); err != nil {
		t.Errorf("Schema up failed: %v", err)
	}
	if err := SchemaDown(); err != nil {
		t.Errorf("Schema down failed: %v", err)
	}
	if err := SchemaDown(); err != nil {
		t.Errorf("Schema 2nd down failed: %v", err)
	}
}

func TestDataUpDown(t *testing.T) {
	if err := SchemaUp(); err != nil {
		t.Errorf("Schema up failed: %v", err)
	}
	if err := DataUp(); err != nil {
		t.Errorf("Data up failed: %v", err)
	}

	if err := DataDown(); err != nil {
		t.Errorf("Data down failed: %v", err)
	}
	if err := SchemaDown(); err != nil {
		t.Errorf("Schema down failed: %v", err)
	}
}

func TestDataDoubleUp(t *testing.T) {
	if err := SchemaUp(); err != nil {
		t.Errorf("Schema up failed: %v", err)
	}
	if err := DataUp(); err != nil {
		t.Errorf("Data up failed: %v", err)
	}
	if err := DataUp(); err != nil {
		t.Errorf("Data 2nd up failed: %v", err)
	}

	if err := DataDown(); err != nil {
		t.Errorf("Data down failed: %v", err)
	}
	if err := SchemaDown(); err != nil {
		t.Errorf("Schema down failed: %v", err)
	}
}

func TestDataDoubleDown(t *testing.T) {
	if err := SchemaUp(); err != nil {
		t.Errorf("Schema up failed: %v", err)
	}
	if err := DataUp(); err != nil {
		t.Errorf("Data up failed: %v", err)
	}

	if err := DataDown(); err != nil {
		t.Errorf("Data down failed: %v", err)
	}
	if err := DataDown(); err != nil {
		t.Errorf("Data 2nd down failed: %v", err)
	}
	if err := SchemaDown(); err != nil {
		t.Errorf("Schema down failed: %v", err)
	}
}

func TestEnsureData(t *testing.T) {
	inVersion, err := SchemaVersion()
	if err != nil {
		t.Fatalf("Coun't get schema inVersion: %v", err)
	}
	if inVersion != 0 {
		t.Fatalf("Starting version was not 0: %v", inVersion)
	}
	if err := EnsureData(); err != nil {
		t.Errorf("%v", err)
	}
	outVersion, err := SchemaVersion()
	if err != nil {
		t.Fatalf("Couldn't get schema outVersion: %v", err)
	}
	if inVersion == outVersion {
		t.Errorf("inVersion == outVersion: %v", inVersion)
	}
	if err := DataDown(); err != nil {
		t.Errorf("Data down failed: %v", err)
	}
	if err := SchemaDown(); err != nil {
		t.Errorf("Schema down failed: %v", err)
	}
}

func TestRemoveData(t *testing.T) {
	inVersion, err := SchemaVersion()
	if err != nil {
		t.Fatalf("Coun't get schema inVersion: %v", err)
	}
	if inVersion != 0 {
		t.Fatalf("Starting version was not 0: %v", inVersion)
	}
	if err := EnsureData(); err != nil {
		t.Fatalf("Couldn't EnsureData: %v", err)
	}
	if err := RemoveData(); err != nil {
		t.Errorf("%v", err)
	}
	outVersion, err := SchemaVersion()
	if err != nil {
		t.Fatalf("Couldn't get schema outVersion: %v", err)
	}
	if outVersion != 0 {
		t.Errorf("outVersion != 0: %v", outVersion)
	}
}

func TestReinitializeAll(t *testing.T) {
	inVersion, err := SchemaVersion()
	if err != nil {
		t.Fatalf("Coun't get schema inVersion: %v", err)
	}
	if inVersion != 0 {
		t.Fatalf("Starting version was not 0: %v", inVersion)
	}
	if err := ReinitializeAll(); err != nil {
		t.Errorf("%v", err)
	}
	outVersion, err := SchemaVersion()
	if err != nil {
		t.Fatalf("Couldn't get schema outVersion: %v", err)
	}
	if inVersion == outVersion {
		t.Errorf("inVersion == outVersion: %v", inVersion)
	}
	if err := DataDown(); err != nil {
		t.Errorf("Data down failed: %v", err)
	}
	if err := SchemaDown(); err != nil {
		t.Errorf("Schema down failed: %v", err)
	}
}
