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

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ancientHacker/susen.go/puzzle"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	clientCount = 5
	runCount    = 3
)

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

type sessionClient struct {
	id       int           // which client this is
	client   *http.Client  // the http client, with cookies
	PID      string        // the puzzle this client works on
	interval int           // the interval, in msec, between calls
	vals     []int         // the expected values of the puzzle
	choice   puzzle.Choice // the first choice to try in this puzzle
}

func rdcConnect(t *testing.T, name string) {
	tlog := &tLogger{t: t, name: name}
	log.SetOutput(tlog)
	alternateShutdown = tlog.shutdown

	redisInit()
	if err := redisConnect(); err != nil {
		shutdown(startupFailureShutdown)
	}
}

func TestSessionSelect(t *testing.T) {
	rdcConnect(t, "TestSessionSelect")
	defer redisClose()

	// one server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := sessionSelect(w, r)
		t.Logf("Session %v handling %s %s.", session.SID, r.Method, r.URL.Path)
		session.rootHandler(w, r)
	}))
	defer srv.Close()

	// helper - select first assigned square as choice
	firstAssigned := func(pvals []int) puzzle.Choice {
		for i := 0; i < len(pvals); i++ {
			if v := pvals[i]; v != 0 {
				return puzzle.Choice{Index: i + 1, Value: v} // 1-based indexing
			}
		}
		panic(fmt.Errorf("No assigned values!"))
	}
	// helpers - track the number of different cookies we see,
	// and how many different values they get
	cmap := make(map[string]map[string]int)
	addCookie := func(c *http.Cookie) {
		if m := cmap[c.Name]; m == nil {
			cmap[c.Name] = make(map[string]int)
		}
		cmap[c.Name][c.Value]++
	}
	countCookies := func(c *sessionClient, target string) {
		url, e := url.Parse(target)
		if e != nil {
			panic(e)
		}
		cookies := c.client.Jar.Cookies(url)
		if len(cookies) == 0 {
			// t.Logf("Client %d: No target cookies.\n", c.id)
		} else if len(cookies) == 1 {
			// t.Logf("Client %d: Target cookie: %v\n", c.id, *cookies[0])
			addCookie(cookies[0])
		} else {
			// t.Logf("Client %d: %d target cookies are:\n", c.id, len(cookies))
			for _, c := range cookies {
				// t.Logf("\tcookie %d: %v\n", i, *c)
				addCookie(c)
			}
		}
	}
	// helper - prevent redirects in a known way
	redirectCount := 0
	redirectFn := func(*http.Request, []*http.Request) error {
		redirectCount++
		return fmt.Errorf("%d", redirectCount)
	}
	// helper - make a call setting the current session puzzle, return false on error
	setPuzzle := func(c *sessionClient, pid string) bool {
		target := fmt.Sprintf("%s/reset/%s", srv.URL, pid)
		t.Logf("Client %d: getting %s", c.id, target)
		countCookies(c, target)
		r, e := c.client.Get(target)
		if e != nil && e.(*url.Error).Err.Error() != fmt.Sprintf("%d", redirectCount) {
			t.Errorf("client %d: Request error: %v", c.id, e)
			return false
		}
		// t.Logf("client %d: %q\n", c.id, r.Status)
		// t.Logf("client %d: %v\n", c.id, r.Header)
		if r.StatusCode != http.StatusFound {
			t.Errorf("client %d: Reset request did not return redirect status: %v",
				c.id, r.StatusCode)
			return false
		}
		if r.Header.Get("Location") != "/solver/" {
			t.Errorf("client %d: Reset request redirected to incorrect location: %v",
				c.id, r.Header.Get("Location"))
			return false
		}
		return true
	}
	// helper - make a squares-returning action call, return false on error
	getState := func(c *sessionClient, action string) bool {
		target := fmt.Sprintf("%s/api/%s", srv.URL, action)
		t.Logf("Client %d: getting %s", c.id, target)
		countCookies(c, target)
		r, e := c.client.Get(target)
		if e != nil {
			t.Errorf("client %d: Request error: %v", c.id, e)
			return false
		}
		// t.Logf("client %d: %q\n", c.id, r.Status)
		// t.Logf("client %d: %v\n", c.id, r.Header)
		b, e := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if e != nil {
			t.Errorf("client %d: Read error on puzzle response body: %v", c.id, e)
			return false
		}

		var content *puzzle.Content
		e = json.Unmarshal(b, &content)
		if e != nil {
			t.Errorf("client %d: Unmarshal failed: %v", c.id, e)
			return false
		}
		s := content.Squares
		if len(s) != len(c.vals) {
			t.Errorf("client %d: Got wrong number of squares: %d", c.id, len(s))
			return false
		}
		for i := 0; i < len(s); i++ {
			if s[i].Aval != c.vals[i] {
				t.Errorf("client %d: Square %d has value %d", c.id, s[i].Index, s[i].Aval)
				return false
			}
		}
		return true
	}
	// helper - make an update-returning action call, return false on error
	getUpdate := func(c *sessionClient) bool {
		t.Logf("Client %d: posting choice %v", c.id, c.choice)
		bs, e := json.Marshal(c.choice)
		if e != nil {
			t.Errorf("client %d: Failed to encode choice: %v", c.id, e)
			return false
		}
		target := fmt.Sprintf("%s/api/assign", srv.URL)
		countCookies(c, target)
		r, e := c.client.Post(target, "application/json", bytes.NewReader(bs))
		if e != nil {
			t.Errorf("client %d: Request error: %v", c.id, e)
			return false
		}
		b, e := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if e != nil {
			t.Errorf("client %d: Read error on puzzle response body: %v", c.id, e)
			return false
		}

		if r.StatusCode != http.StatusBadRequest {
			t.Errorf("client %d: Bad assignment returned unexpected status: %d",
				c.id, r.StatusCode)
		} else {
			var err puzzle.Error
			e = json.Unmarshal(b, &err)
			if e != nil {
				t.Errorf("client %d: Unmarshal failed: %v", c.id, e)
				return false
			}
			if err.Condition != puzzle.DuplicateAssignmentCondition {
				t.Errorf("client %d: Got unexpected error: %v", c.id, err)
			}
		}
		return true
	}
	// helper - sleep interval milliseconds
	sleep := func(c *sessionClient) {
		sleeptime := time.Duration(c.interval) * time.Millisecond
		t.Logf("Client %d sleeps %s", c.id, sleeptime)
		time.Sleep(sleeptime)
	}

	// make clients
	clients := make([]*sessionClient, clientCount)
	for i := 0; i < clientCount; i++ {
		jar, e := cookiejar.New(nil)
		if e != nil {
			t.Fatalf("Failed to create cookie jar #%d: %v", i+1, e)
		}
		// try every key except the default "1-star"
		testKeys := []string{"2-star", "3-star", "4-star", "5-star", "6-star"}
		keyIndex := i % len(testKeys)
		pid := testKeys[keyIndex]
		puzzleVals := puzzleSummaries[pid].Values
		clients[i] = &sessionClient{
			id:       i + 1,
			client:   &http.Client{Jar: jar, CheckRedirect: redirectFn},
			PID:      pid,
			interval: (i*17)%100 + 100,
			vals:     puzzleVals,
			choice:   firstAssigned(puzzleVals),
		}
		// t.Logf("Client %d: %+v\n", clients[i].id, *clients[i])
	}

	// each client makes runCount sets of 3 calls: reset then assign then back
	// after runCount sets, the client reports back, and we wait for all clients
	ch := make(chan int, clientCount)
	start := time.Now()
	for i := 0; i < clientCount; i++ {
		go func(client *sessionClient) {
			for i := 0; i < runCount; i++ {
				sleep(client)
				if !setPuzzle(client, client.PID) {
					break
				}
				if !getState(client, "/state") {
					break
				}
				sleep(client)
				if !getUpdate(client) {
					break
				}
				sleep(client)
				if !getState(client, fmt.Sprintf("/back/")) {
					break
				}
			}
			ch <- client.id
		}(clients[i])
	}
	for i := 0; i < clientCount; i++ {
		id := <-ch
		diff := time.Now().Sub(start)
		t.Logf("Client %d finished in %v\n", id, diff)
	}
	// the number of sessions is the number of different values of the different cookies
	sessionCount := 0
	for _, v := range cmap {
		sessionCount += len(v)
	}
	if sessionCount != clientCount {
		t.Errorf("Run produced %d (not %d) session cookies: %v", len(cmap), clientCount, cmap)
	}
}

func TestIssue1(t *testing.T) {
	rdcConnect(t, "TestIssue1")
	defer redisClose()

	// server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := sessionSelect(w, r)
		t.Logf("Session %v handling %s %s.", session.SID, r.Method, r.URL.Path)
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
			target := fmt.Sprintf("%s/reset/%d-star", srv.URL, i+j+1)
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
	rdcConnect(t, "TestIssue11")
	defer redisClose()

	// add puzzle and appropriate assignments for testing
	puzzleSummaries["test11"] = &puzzle.Summary{
		Geometry:   puzzle.SudokuGeometryName,
		SideLength: 4,
		Values: []int{
			1, 0, 3, 0,
			0, 3, 0, 1,
			3, 0, 1, 0,
			0, 1, 0, 3,
		}}
	defer func() { delete(puzzleSummaries, "test11") }()
	choices := []puzzle.Choice{{13, 2}, {10, 4}, {15, 4}}

	// server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// t.Logf("Handling %s %s...", r.Method, r.URL.Path)
		session := sessionSelect(w, r)
		session.rootHandler(w, r)
		// t.Logf("There are %d steps in session %q", session.Step, session.SID)
		// t.Logf("The puzzleID of session %q is %q", session.SID, session.PID)
	}))
	defer srv.Close()

	// client
	jar, e := cookiejar.New(nil)
	if e != nil {
		t.Fatalf("Failed to create cookie jar: %v", e)
	}
	c := http.Client{Jar: jar}

	// set the puzzle
	r, e := c.Get(srv.URL + "/reset/test11")
	if e != nil || r.StatusCode != http.StatusOK {
		t.Fatalf("Request error on /reset/test11: %v", e)
	}

	// do the assignments
	for i, choice := range choices {
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

	// we used to clear the in-memory session cache here, but now
	// that there isn't any we don't have to

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

	// now go back two steps
	r, e = c.Get(srv.URL + "/api/back/")
	if e != nil {
		t.Fatalf("Go Back 1 error: %v", e)
	}
	if r.StatusCode != http.StatusOK {
		t.Errorf("Go Back 1 status was %v not %v", r.StatusCode, http.StatusOK)
	}
	r, e = c.Get(srv.URL + "/api/back/")
	if e != nil {
		t.Fatalf("Go Back 2 error: %v", e)
	}
	if r.StatusCode != http.StatusOK {
		t.Errorf("Go Back 2 status was %v not %v", r.StatusCode, http.StatusOK)
	}

	// read the puzzle contents
	r, e = c.Get(srv.URL + "/api/state")
	if e != nil {
		t.Fatalf("State before request error: %v", e)
	}
	if r.StatusCode != http.StatusOK {
		t.Errorf("State before status was %v not %v", r.StatusCode, http.StatusOK)
	}
	stateBefore, e = ioutil.ReadAll(r.Body)
	r.Body.Close()
	if e != nil {
		t.Fatalf("Read error on state before: %v", e)
	}

	// again, no cache to clear

	// read the puzzle contents again, compare
	r, e = c.Get(srv.URL + "/api/state")
	if e != nil {
		t.Fatalf("State after request error: %v", e)
	}
	if r.StatusCode != http.StatusOK {
		t.Errorf("State after status was %v not %v", r.StatusCode, http.StatusOK)
	}
	stateAfter, e = ioutil.ReadAll(r.Body)
	r.Body.Close()
	if e != nil {
		t.Fatalf("Read error on state after: %v", e)
	}
	if string(stateAfter) != string(stateBefore) {
		t.Errorf("States don't match before (%+v) and after (%+v).", stateBefore, stateAfter)
	}
}
