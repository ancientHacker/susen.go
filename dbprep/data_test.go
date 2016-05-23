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
	"strings"
	"testing"
)

// make sure string case invariants are met
func TestSampleData(t *testing.T) {
	for i, hash := range sampleHashes {
		if hash != strings.ToUpper(hash) {
			t.Errorf("Hash %d (%s) contains a non-uppercase letter.", i, hash)
		}
	}
	for i, name := range sampleNames {
		if name != strings.ToLower(name) {
			t.Errorf("Name %d (%s) contains a non-lowercase letter.", i, name)
		}
	}
}
