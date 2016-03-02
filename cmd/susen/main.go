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
	"fmt"
	"github.com/ancientHacker/susen.go/client"
	"github.com/ancientHacker/susen.go/storage"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

func main() {
	// client initialization
	if err := client.VerifyResources(); err != nil {
		log.Printf("Error during client initialization: %v", err)
		shutdown(startupFailureShutdown)
	}
	// storage initialization
	if err := storage.Connect(); err != nil {
		log.Printf("Error during storage initialization: %v", err)
		shutdown(startupFailureShutdown)
	}

	// port sensing
	port := os.Getenv("PORT")
	if port == "" {
		// running locally in dev mode
		port = "localhost:8080"
	} else {
		// running as a true server
		port = ":" + port
	}

	// catch signals
	shutdownOnSignal()

	// serve
	log.Printf("Listening on %s...", port)
	http.HandleFunc("/", serveHttp)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Printf("Listener failure: %v", err)
		shutdown(listenerFailureShutdown)
	}
}

/*

request handlers

*/

var apiEndpointRegexp = regexp.MustCompile("^/+api/+([a-z]+)/*$")

type userSession struct {
	storage.Session
}

func serveHttp(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			errorHandler(err, w, r)
		}
	}()

	if client.StaticHandler(w, r) {
		return
	}
	session := sessionSelect(w, r)
	session.rootHandler(w, r)
}

func (session *userSession) apiHandler(w http.ResponseWriter, r *http.Request) {
	sendState := func() {
		session.Puzzle.StateHandler(w, r)
		log.Printf("Returned current state for %s:%q step %d", session.SID, session.PID, session.Step)
	}
	sendSummary := func() {
		session.Puzzle.SummaryHandler(w, r)
		log.Printf("Returned current summary for %s:%q step %d", session.SID, session.PID, session.Step)
	}
	sendNotAllowed := func() {
		http.Error(w, apiEndpointUnknown(r.URL.Path), http.StatusMethodNotAllowed)
		log.Printf("Endpoint %q cannot accept %s: returned a MethodNotAllowed error.", r.URL.Path, r.Method)
	}
	sendNotFound := func() {
		http.Error(w, apiEndpointUnknown(r.URL.Path), http.StatusNotFound)
		log.Printf("Endpoint %q unknown: returned a NotFound error.", r.URL.Path)
	}

	matches := apiEndpointRegexp.FindStringSubmatch(r.URL.Path)
	if matches == nil {
		http.Error(w, apiEndpointUnknown(r.URL.Path), http.StatusNotFound)
		log.Printf("Unknown endpoint %q: returned a NotFound error.", r.URL.Path)
		return
	}
	switch matches[1] {
	case "reset":
		if r.Method == "GET" {
			session.StartPuzzle("")
			sendState()
		} else {
			sendNotAllowed()
		}
	case "back":
		if r.Method == "GET" {
			session.RemoveStep()
			sendState()
		} else {
			sendNotAllowed()
		}
	case "state":
		if r.Method == "GET" {
			sendState()
		} else {
			sendNotAllowed()
		}
	case "summary":
		if r.Method == "GET" {
			sendSummary()
		} else {
			sendNotAllowed()
		}
	case "assign":
		if r.Method == "POST" {
			update, e := session.Puzzle.AssignHandler(w, r)
			if e != nil {
				log.Printf("Assign to %s:%q step %d failed: %v", session.SID, session.PID, session.Step, e)
			} else {
				session.AddStep()
				if update.Errors != nil {
					log.Printf("Assign to %s:%q gave errors; step %d is unsolvable.",
						session.SID, session.PID, session.Step)
				}
			}
		} else {
			sendNotAllowed()
		}
	default:
		sendNotFound()
	}
}

func (session *userSession) solverHandler(w http.ResponseWriter, r *http.Request) {
	body := client.SolverPage(session.SID, session.PID, session.Summary)
	hs := w.Header()
	hs.Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(body))
	log.Printf("Returned solver page for %s:%q step %d.", session.SID, session.PID, session.Step)
}

func (session *userSession) homeHandler(w http.ResponseWriter, r *http.Request) {
	var others []string
	for k := range storage.CommonPuzzles() {
		if k != session.PID {
			others = append(others, k)
		}
	}
	sort.Strings(others)
	body := client.HomePage(session.SID, session.PID, others)
	hs := w.Header()
	hs.Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(body))
	log.Printf("Returned home page for %s:%q step %d.",
		session.SID, session.PID, session.Step)
}

func (session *userSession) rootHandler(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasPrefix(r.URL.Path, "/reset/"):
		http.Redirect(w, r, "/solver/", http.StatusFound)
		log.Printf("Redirected to solver page for %s:%q step %d.",
			session.SID, session.PID, session.Step)
	case strings.HasPrefix(r.URL.Path, "/api/"):
		session.apiHandler(w, r)
	case strings.HasPrefix(r.URL.Path, "/solver/"):
		session.solverHandler(w, r)
	case strings.HasPrefix(r.URL.Path, "/home/"):
		session.homeHandler(w, r)
	default:
		http.Redirect(w, r, "/home/", http.StatusFound)
		log.Printf("Redirected to home page for %s:%q step %d.",
			session.SID, session.PID, session.Step)
	}
}

func errorHandler(err interface{}, w http.ResponseWriter, r *http.Request) {
	var body string
	switch err.(type) {
	case error:
		body = client.ErrorPage(err.(error))
	default:
		body = client.ErrorPage(fmt.Errorf("%v", err))
	}
	hs := w.Header()
	hs.Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(body))
	log.Printf("Returned server error page for %s of %q.", r.Method, r.URL.Path)
}

/*

session handling

*/

// We use a cookie to associate sessions with clients (by storing
// the session ID in the cookie).  These are the values shared
// among all our cookies.
const (
	cookieNameBase = "susenID"
	cookiePath     = "/"
	cookieMaxAge   = 3600 * 24 * 365 * 10 // 10 years
)

var (
	startTime = time.Now() // instance start-up time
)

// getCookie gets the session cookie, or sets a new one.  It
// returns the session ID associated with the cookie.
//
// The logic around session cookies is that we have one cookie
// name per connection protocol, so that opening both a secure
// and an insecure session from the same browser gives you two
// sessions.  We need to do this because we often have one
// instance serving both protocols, with the protocol termination
// done at a load-balancer.
func getCookie(w http.ResponseWriter, r *http.Request) string {
	// Issue #1: Heroku-transported protocols are specified in a header
	proto := "httpx" // absent other indicators, protocol is unknown
	if herokuProtocol := r.Header.Get("X-Forwarded-Proto"); herokuProtocol != "" {
		proto = herokuProtocol
	}

	// check for an existing cookie whose name matches the protocol
	cookieName := cookieNameBase + "-" + proto
	if sc, e := r.Cookie(cookieName); e == nil && sc.Value != "" {
		return sc.Value
	}

	// no session cookie: start a new session with a new ID
	// poor man's UUID for the session in local mode: time since startup.
	sid := strconv.FormatInt(int64(time.Now().Sub(startTime)), 36)
	// if we're on an infrastructure, we use the request ID
	if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
		sid = requestID
	}
	log.Printf("No session cookie found, created new session ID %q", sid)
	sc := &http.Cookie{Name: cookieName, Value: sid, Path: cookiePath, MaxAge: cookieMaxAge}
	http.SetCookie(w, sc)
	return sid
}

// sessionSelect: find or create the session for the current connection.
func sessionSelect(w http.ResponseWriter, r *http.Request) *userSession {
	// check to see if this is a force reset of the session
	forceReset, resetID := false, ""
	if strings.HasPrefix(r.URL.Path, "/reset/") {
		forceReset = true
		resetID = r.URL.Path[len("/reset/"):]
	}
	id := getCookie(w, r)
	// create an in-memory session with this cookie
	session := &userSession{storage.Session{SID: id, Created: time.Now().Format(time.RFC3339)}}
	// load session from storage if possible, otherwise just initialize it
	if session.Lookup() {
		log.Printf("Found session %v, puzzle %q, on step %d.", session.SID, session.PID, session.Step)
		if forceReset {
			session.StartPuzzle(resetID)
		} else {
			session.LoadStep()
		}
	} else if forceReset {
		session.StartPuzzle(resetID)
	} else {
		session.StartPuzzle(storage.DefaultPuzzleID())
	}
	return session
}

/*

coordinate shutdown across goroutines and top-level server

*/

type shutdownCause int

const (
	unknownShutdown = iota
	runtimeFailureShutdown
	startupFailureShutdown
	caughtSignalShutdown
	listenerFailureShutdown
)

// for testing, allow alternate forms of shutdown
var alternateShutdown func(reason shutdownCause)

// shutdown: process exit with logging.
func shutdown(reason shutdownCause) {
	// close down the storage connections
	storage.Close()

	// for testing: run alternateShutdown instead, if defined
	if alternateShutdown != nil {
		alternateShutdown(reason)
		panic(reason) // shouldn't get here
	}

	// log reason for shutdown and exit
	switch reason {
	case unknownShutdown:
		log.Fatal("Exiting: normal shutdown.")
	case startupFailureShutdown:
		log.Fatal("Exiting: initialization failure.")
	case runtimeFailureShutdown:
		log.Fatal("Exiting: runtime failure.")
	case caughtSignalShutdown:
		log.Fatal("Exiting: caught signal.")
	case listenerFailureShutdown:
		log.Fatal("Exiting: web server failed.")
	default:
		log.Fatal("Exiting: unknown cause.")
	}
}

// shutdownOnSignal: catch signals and exit.
func shutdownOnSignal() {
	// based on example in os.signal godoc
	c := make(chan os.Signal, 1)
	signal.Notify(c) // die on all signals

	go func() {
		s := <-c
		log.Printf("Received OS-level signal: %v", s)
		shutdown(caughtSignalShutdown)
	}()
}

/*

various low-level utilities

*/

// apiEndpointUnknown: a pre-serialized JSON Error used when
// someone calls a non-existent API endpoint.
func apiEndpointUnknown(endpoint string) string {
	return `{"scope": "1", "structure": "1", "condition": "1", "values": ["No such endpoint"], ` +
		`"message": "No such endpoint: ` + endpoint + `"}`
}
