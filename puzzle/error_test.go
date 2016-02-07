// susen.go - a web-based Sudoku game and teaching tool.
// Copyright (C) 2015 Daniel C. Brotsky.
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

package puzzle

import (
	"testing"
)

// Make sure error messages never panic and are never empty.  The
// testing of individual cases (and removal of unused errors) we
// leave to the functional testing done of other files.
func TestErrorNoPanicNoEmpty(t *testing.T) {
	defer (func() {
		if e := recover(); e != nil {
			t.Fatalf("Panic during testing: %v", e)
		}
	})()
	for sc := int(UnknownScope); sc <= int(MaxScope); sc++ {
		for st := int(UnknownStructure); st < int(MaxStructure); st++ {
			for at := int(UnknownAttribute); at < int(MaxAttribute); at++ {
				for co := int(UnknownCondition); co < int(MaxCondition); co++ {
					e := Error{
						Scope:     ErrorScope(sc),
						Structure: ErrorStructure(st),
						Attribute: ErrorAttribute(at),
						Condition: ErrorCondition(co),
					}
					m := e.Error()
					// t.Log(m)
					if len(m) == 0 {
						t.Errorf("Empty error message for %+v", e)
					}
				}
			}
		}
	}
}
