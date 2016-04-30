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

package main

import (
	"bytes"
	"github.com/ancientHacker/susen.go/storage"
	"log"
	"os"
	"path/filepath"
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

func testSetup(t *testing.T) {
	// log initialization
	tlog := &tLogger{t: t}
	if !testing.Short() {
		log.SetOutput(tlog)
	} else {
		log.SetOutput(os.Stderr)
	}
	// storage initialization
	os.Setenv("DBPREP_PATH", filepath.Join("..", "..", "dbprep"))
	cacheId, databaseId, err := storage.Connect()
	if err != nil {
		t.Fatalf("Error during storage initialization: %v", err)
	}
	log.Printf("Connected to cache at %q", cacheId)
	log.Printf("Connected to database at %q", databaseId)
}

func TestNullInput(t *testing.T) {
	testSetup(t)
	defer storage.Close()

	null := new(bytes.Buffer)
	err := listener(os.Stdout, null)
	if err != nil {
		t.Fatalf("CLI failure: %v", err)
	}
}

func TestMarkdown(t *testing.T) {
	testSetup(t)
	defer storage.Close()

	in := bytes.NewBufferString("markdown\nmarkdown on\nmarkdown off\n")
	out := new(bytes.Buffer)
	err := listener(out, in)
	if err != nil {
		t.Fatalf("CLI failure: %v", err)
	}
	expected := "Markdown is off\nMarkdown is on\nMarkdown is off\n"
	result := out.String()
	if result != expected {
		t.Errorf("Got %q, expected %q", result, expected)
	}
}

func TestSmallBuffer(t *testing.T) {
	oldsize := bufsize
	bufsize := 10
	defer func() { bufsize = oldsize }()

	testSetup(t)
	defer storage.Close()

	in := bytes.NewBufferString("markdown\nmarkdown on\nmarkdown off\n")
	out := new(bytes.Buffer)
	err := listener(out, in)
	if err != nil {
		t.Fatalf("CLI failure: %v", err)
	}
	expected := "Markdown is off\nMarkdown is on\nMarkdown is off\n"
	result := out.String()
	if result != expected {
		t.Errorf("Got %q, expected %q", result, expected)
	}
}

func TestHints(t *testing.T) {
	testSetup(t)
	defer storage.Close()

	in := bytes.NewBufferString("hints\nhints off\nhints on\n")
	out := new(bytes.Buffer)
	err := listener(out, in)
	if err != nil {
		t.Fatalf("CLI failure: %v", err)
	}
	expected := "Hints are on\nHints are off\nHints are on\n"
	result := out.String()
	if result != expected {
		t.Errorf("Got %q, expected %q", result, expected)
	}
}

func TestBackFail(t *testing.T) {
	testSetup(t)
	defer storage.Close()

	in := bytes.NewBufferString("back\n")
	out := new(bytes.Buffer)
	err := listener(out, in)
	if err != nil {
		t.Fatalf("CLI failure: %v", err)
	}
	expected := "No choices to undo.\n"
	result := out.String()
	if result != expected {
		t.Errorf("Got %q, expected %q", result, expected)
	}
}
