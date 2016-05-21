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
	"encoding/json"
	"fmt"
	"github.com/ancientHacker/susen.go/puzzle"
	"github.com/ancientHacker/susen.go/storage"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

/*

known-good data for sample session puzzles

*/

const sampleDefaultName = "sample-1"

type testDataEntry struct {
	name     string
	geometry string
	choices  []puzzle.Choice
}

var testData = []testDataEntry{
	{"sample-1", puzzle.StandardGeometryName,
		[]puzzle.Choice{{51, 1}, {41, 8}, {31, 2}}},
	{"sample-7", puzzle.RectangularGeometryName,
		[]puzzle.Choice{{1, 2}, {6, 3}}},
	{"sample-8", puzzle.RectangularGeometryName,
		[]puzzle.Choice{{22, 4}, {23, 5}, {15, 1}, {16, 3}}},
}

/*

test setup:

1. divert top-level logging to test log
2. connect to storage and divert logging

*/

type tLogger struct {
	t    *testing.T
	name string
	log  bytes.Buffer
}

func (t *tLogger) Write(p []byte) (n int, e error) {
	n, e = t.log.Write(p)
	t.t.Log(string(p[:n-1]))
	return
}

func (t *tLogger) shutdown(reason shutdownCause) {
	t.t.Errorf("Shutdown: reason code is %v", reason)
	fmt.Fprintf(os.Stderr, "Shutdown in %s: reason code is %d\n", t.name, reason)
	fmt.Fprintf(os.Stderr, "Dumping log of test run before exit...\n")
	t.log.WriteTo(os.Stderr)
	os.Exit(int(reason))
}

func storageConnect(t *testing.T, name string) {
	tlog := &tLogger{t: t, name: name}
	if !testing.Short() {
		log.SetOutput(tlog)
	}
	alternateShutdown = tlog.shutdown

	os.Setenv("DBPREP_PATH", filepath.Join("..", "..", "dbprep"))
	if _, _, err := storage.Connect(); err != nil {
		shutdown(startupFailureShutdown)
	}
}

/*

tests

*/

func TestIssue1(t *testing.T) {
	storageConnect(t, "TestIssue1")
	defer storage.Close()

	// server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s := &session{sid: getCookie(w, r)}
		s.load(w, r)
		if testing.Short() {
			t.Logf("Session %v handling %s %s.", s.sid, r.Method, r.URL.Path)
		}
		http.Error(w, "This is a test", http.StatusOK)
	}))
	defer srv.Close()

	// client
	jar, e := cookiejar.New(nil)
	if e != nil {
		t.Fatalf("Failed to create cookie jar: %v", e)
	}
	c := http.Client{Jar: jar}

	// for each heroku protocol indicator, do two pairs of
	// requests, one to get the cookie set, one to use it.  We
	// also handle the case where there is no heroku protocol
	// indicator, which is a bit of overkill, since no server
	// should get both Heroku and non-Heroku requests, but you
	// never know :).
	for i, herokuProtocol := range []string{"", "http", "https"} {
		for j, expectSetCookie := range []bool{true, false} {
			target := fmt.Sprintf("%s/home", srv.URL)
			req, e := http.NewRequest("GET", target, nil)
			if e != nil {
				t.Fatalf("Failed to create request %d: %v", 2*i+j, e)
			}
			if herokuProtocol != "" {
				req.Header.Add("X-Forwarded-Proto", herokuProtocol)
			}
			r, e := c.Do(req)
			if e != nil {
				t.Fatalf("Request error: %v", e)
			}
			if r.StatusCode != http.StatusOK {
				t.Errorf("Got status %q, expected OK", r.Status)
			}
			r.Body.Close()
			if expectSetCookie {
				if h := r.Header.Get("Set-Cookie"); h == "" {
					t.Errorf("No Set-Cookie received on request %d.", 2*i+j)
				}
			} else {
				if h := r.Header.Get("Set-Cookie"); h != "" {
					t.Errorf("Set-Cookie received on request %d.", 2*i+j)
				}
			}
		}
	}

	// now make sure the protocol cookies are set for the next round
	for i, herokuProtocol := range []string{"", "http", "https"} {
		for j, expectSetCookie := range []bool{false, false} {
			target := fmt.Sprintf("%s", srv.URL)
			req, e := http.NewRequest("GET", target, nil)
			if e != nil {
				t.Fatalf("Failed to create request %d: %v", 2*i+j, e)
			}
			if herokuProtocol != "" {
				req.Header.Add("X-Forwarded-Proto", herokuProtocol)
			}
			r, e := c.Do(req)
			if e != nil {
				t.Fatalf("Request error: %v", e)
			}
			if r.StatusCode != http.StatusOK {
				t.Errorf("Got status %q, expected OK", r.Status)
			}
			r.Body.Close()
			if expectSetCookie {
				if h := r.Header.Get("Set-Cookie"); h == "" {
					t.Errorf("No Set-Cookie received on request %d.", 2*i+j)
				}
			} else {
				if h := r.Header.Get("Set-Cookie"); h != "" {
					t.Errorf("Set-Cookie received on request %d.", 2*i+j)
				}
			}
		}
	}
}

func TestIssue11(t *testing.T) {
	storageConnect(t, "TestIssue11")
	defer storage.Close()

	// server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if testing.Short() {
			t.Logf("Handling %s %s...", r.Method, r.URL.Path)
		}
		s := &session{sid: getCookie(w, r)}
		s.load(w, r)
		s.rootHandler(w, r)
	}))
	defer srv.Close()

	// client
	jar, e := cookiejar.New(nil)
	if e != nil {
		t.Fatalf("Failed to create cookie jar: %v", e)
	}
	c := http.Client{Jar: jar}

	// for each test puzzle
	for _, td := range testData {
		r, e := c.Get(srv.URL + "/reset/" + td.name)
		if e != nil || r.StatusCode != http.StatusOK {
			t.Fatalf("Request error on /reset/%s: %v", td.name, e)
		}

		// read the puzzle contents
		r, e = c.Get(srv.URL + "/api/state")
		if e != nil {
			t.Fatalf("State before request error: %v", e)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("State before status was %v not %v", r.StatusCode, http.StatusOK)
		}
		stateOriginal, e := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if e != nil {
			t.Fatalf("Read error on original state: %v", e)
		}

		// do the assignments
		for i, choice := range td.choices {
			b, e := json.Marshal(choice)
			if e != nil {
				t.Fatalf("Case %d: Failed to encode choice: %v", i, e)
			}
			r, e := c.Post(srv.URL+"/api/assign", "application/json", strings.NewReader(string(b)))
			if e != nil {
				t.Fatalf("assignment %d: Request error: %v", i, e)
			}
			if r.StatusCode != http.StatusOK {
				t.Errorf("case %d: Status was %v, expected %v", i, r.StatusCode, http.StatusOK)
			}
			_, e = ioutil.ReadAll(r.Body)
			r.Body.Close()
			if e != nil {
				t.Fatalf("test %d: Read error on update: %v", i, e)
			}
		}

		// read the puzzle contents
		r, e = c.Get(srv.URL + "/api/state")
		if e != nil {
			t.Fatalf("State before request error: %v", e)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("State before status was %v not %v", r.StatusCode, http.StatusOK)
		}
		stateBefore, e := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if e != nil {
			t.Fatalf("Read error on state before: %v", e)
		}

		// read the puzzle contents again, compare
		r, e = c.Get(srv.URL + "/api/state")
		if e != nil {
			t.Fatalf("State after request error: %v", e)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("State after status was %v not %v", r.StatusCode, http.StatusOK)
		}
		stateAfter, e := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if e != nil {
			t.Fatalf("Read error on state after: %v", e)
		}
		if string(stateAfter) != string(stateBefore) {
			t.Errorf("States don't match before (%+v) and after (%+v).", stateBefore, stateAfter)
		}

		// now go back over all the choices
		for i := range td.choices {
			r, e = c.Get(srv.URL + "/api/back/")
			if e != nil {
				t.Fatalf("Go Back %d error: %v", i, e)
			}
			if r.StatusCode != http.StatusOK {
				t.Errorf("Go Back %d status was %v not %v", i, r.StatusCode, http.StatusOK)
			}
		}

		// read the puzzle contents
		r, e = c.Get(srv.URL + "/api/state")
		if e != nil {
			t.Fatalf("State before request error: %v", e)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("State before status was %v not %v", r.StatusCode, http.StatusOK)
		}
		stateEnding, e := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if e != nil {
			t.Fatalf("Read error on ending state: %v", e)
		}

		if string(stateOriginal) != string(stateEnding) {
			t.Errorf("States don't match original (%+v) and ending (%+v).",
				stateOriginal, stateEnding)
		}
	}
}
