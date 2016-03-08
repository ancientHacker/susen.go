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
	"bytes"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type tLogger struct {
	t   *testing.T
	log bytes.Buffer
}

func (t *tLogger) Write(p []byte) (n int, e error) {
	n, e = t.log.Write(p)
	t.t.Log(string(p[:n-1]))
	return
}

func setLog(t *testing.T) {
	if !testing.Short() {
		log.SetOutput(&tLogger{t: t})
	}
}

func TestConnect(t *testing.T) {
	setLog(t)
	os.Setenv("DBPREP_PATH", filepath.Join("..", "dbprep"))
	if err := Connect(); err != nil {
		t.Errorf("Couldn't connect to storage: %v", err)
	}
	Close()
}

func TestLoadCommon(t *testing.T) {
	setLog(t)
	os.Setenv("DBPREP_PATH", filepath.Join("..", "dbprep"))
	if err := Connect(); err != nil {
		t.Fatalf("Couldn't connect to storage: %v", err)
	}
	sums1 := CommonSummaries()
	if len(sums1) == 0 {
		t.Errorf("Didn't get any common summaries.")
	}
	sums2 := CommonSummaries()
	if reflect.ValueOf(sums2).Pointer() != reflect.ValueOf(sums1).Pointer() {
		t.Errorf("Second call to CommonSummaries returned a different pointer than first.")
	}
	Close()
}
