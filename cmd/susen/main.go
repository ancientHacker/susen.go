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
	"flag"
	"fmt"
	"github.com/ancientHacker/susen.go/client"
	"github.com/ancientHacker/susen.go/puzzle"
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

// flags
var (
	debugLog = flag.Bool("d", true, "debugging info in log")
)

func main() {
	// parse flags, if anything left over it's a usage problem
	flag.Parse()
	if flag.NArg() > 0 {
		flag.PrintDefaults()
		os.Exit(2)
	}
	if *debugLog {
		log.Printf("-d specified: debug messages to log")
	}

	// client initialization
	if err := client.VerifyResources(); err != nil {
		log.Printf("Error during client initialization: %v", err)
		shutdown(startupFailureShutdown)
	}
	// storage initialization
	if cacheId, databaseId, err := storage.Connect(); err != nil {
		log.Printf("Error during storage initialization: %v", err)
		shutdown(startupFailureShutdown)
	} else {
		log.Printf("Connected to cache at %q", cacheId)
		log.Printf("Connected to database at %q", databaseId)
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

sessions

*/

type session struct {
	sid string           // session ID
	ss  *storage.Session // underlying storage session
}

// working puzzle state
func (s *session) puzzle() *puzzle.Puzzle {
	return s.ss.Puzzle
}

// ID of working puzzle
func (s *session) pid() string {
	return s.ss.Info.PuzzleId
}

// name of working puzzle
func (s *session) name() string {
	return s.ss.Info.Name
}

// step count of working puzzle
func (s *session) step() int {
	return len(s.ss.Info.Choices) + 1
}

/*

request handlers

*/

// endpoint regular expressions
var (
	apiEndpointPattern    = "^/+api/?"
	solverEndpointPattern = "^/+solver/?"
	homeEndpointPattern   = "^/+home/?"
	selectEndpointPattern = "^/+(reset|select)/?"
	apiEndpointRegexp     = regexp.MustCompile("^/+api/+([a-z]+)/*$")
	selectEndpointRegexp  = regexp.MustCompile("^/+(reset|select)/+([a-zA-Z0-9-]+)/*$")
)

func serveHttp(w http.ResponseWriter, r *http.Request) {
	// session selection
	var s *session

	// runtime error handling
	defer func() {
		if err := recover(); err != nil {
			if s == nil || s.sid == "" {
				log.Printf("Error getting session cookie: %v", err)
			} else if s.ss != nil {
				log.Printf("Error in session %s:%q step %d: %v", s.sid, s.name(), s.step(), err)
			} else {
				log.Printf("Error in session %s: %v", s.sid, err)
			}
			errorHandler(err, w, r)
		}
	}()

	s = &session{sid: getCookie(w, r)}
	if client.StaticHandler(w, r) {
		return
	}
	s.load(w, r)
	s.rootHandler(w, r)
}

func (s *session) rootHandler(w http.ResponseWriter, r *http.Request) {
	if test, _ := regexp.MatchString(apiEndpointPattern, r.URL.Path); test {
		s.apiHandler(w, r)
	} else if test, _ = regexp.MatchString(solverEndpointPattern, r.URL.Path); test {
		s.solverHandler(w, r)
	} else if test, _ = regexp.MatchString(homeEndpointPattern, r.URL.Path); test {
		s.homeHandler(w, r)
	} else if test, _ = regexp.MatchString(selectEndpointPattern, r.URL.Path); test {
		http.Redirect(w, r, "/solver/", http.StatusFound)
		log.Printf("Redirected to home page on request for %q", r.URL.Path)
	} else {
		http.Redirect(w, r, "/home/", http.StatusFound)
		log.Printf("Redirected to home page on request for %q", r.URL.Path)
	}
}

func (s *session) apiHandler(w http.ResponseWriter, r *http.Request) {
	sendState := func() {
		s.puzzle().StateHandler(w, r)
		log.Printf("Returned current state for %s:%q step %d", s.sid, s.name(), s.step())
	}
	sendSummary := func() {
		s.puzzle().SummaryHandler(w, r)
		log.Printf("Returned current summary for %s:%q step %d", s.sid, s.name(), s.step())
	}
	sendNotAllowed := func() {
		http.Error(w, apiEndpointUnknown(r.URL.Path), http.StatusMethodNotAllowed)
		log.Printf("Endpoint %q cannot accept %s: returned a MethodNotAllowed error.",
			r.URL.Path, r.Method)
	}
	sendNotFound := func() {
		http.Error(w, apiEndpointUnknown(r.URL.Path), http.StatusNotFound)
		log.Printf("Uknown endpoint %q: returned a NotFound error.", r.URL.Path)
	}

	matches := apiEndpointRegexp.FindStringSubmatch(r.URL.Path)
	if matches == nil {
		sendNotFound()
		return
	}
	switch strings.ToLower(matches[1]) {
	case "reset":
		if r.Method == "GET" {
			s.ss.RemoveAllSteps()
			sendState()
		} else {
			sendNotAllowed()
		}
	case "back":
		if r.Method == "GET" {
			if s.step() > 1 {
				s.ss.RemoveStep()
				log.Printf("Reverted session %s:%q to step %d.", s.sid, s.name(), s.step()-1)
			}
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
			choice, update, err := s.puzzle().AssignHandler(w, r)
			if update == nil {
				log.Printf("Assign of %+v at %s:%q step %d failed: %v",
					*choice, s.sid, s.name(), s.step(), err)
			} else {
				if len(update.Errors) > 0 {
					log.Printf("Assign of %+v at %s:%q step %d made puzzle unsolvable.",
						choice, s.sid, s.name(), s.step())
				} else {
					log.Printf("Assign of %+v at %v:%q step %d left puzzle solvable",
						choice, s.sid, s.name(), s.step())
				}
				s.ss.AddStep(*choice)
				if err != nil {
					log.Printf("WARNING: Result of assign at %v:%q step %d failed to encode!",
						s.sid, s.name(), s.step())
				}
			}
		} else {
			sendNotAllowed()
		}
	default:
		sendNotFound()
	}
}

func (s *session) solverHandler(w http.ResponseWriter, r *http.Request) {
	summary, err := s.puzzle().Summary()
	if err != nil {
		panic(fmt.Errorf("Failed to create summary for puzzle: %v", err))
	}
	body := client.SolverPage(s.sid, s.ss.Info, summary.Values)
	hs := w.Header()
	hs.Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(body))
	log.Printf("Returned solver page for %s:%q step %d.", s.sid, s.name(), s.step())
}

func (s *session) homeHandler(w http.ResponseWriter, r *http.Request) {
	infos := s.ss.GetInactivePuzzles()
	sort.Sort(storage.ByLatestView(infos))
	body := client.HomePage(s.sid, s.ss.Info, infos)
	hs := w.Header()
	hs.Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(body))
	log.Printf("Returned home page for %s:%q step %d.",
		s.sid, s.name(), s.step())
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
	cookieNameBase   = "susenID"
	cookieAgeBase    = "susenDate"
	cookiePath       = "/"
	cookieMaxAge     = 3600 * 24 * 365 // 1 year
	cookieRefreshAge = 3600 * 24 * 1   // 1 day
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

	// helpers for getting and setting cookies
	getCookies := func() (id string, age bool) {
		idName, ageName := cookieNameBase+"-"+proto, cookieAgeBase+"-"+proto
		if sc, e := r.Cookie(idName); e == nil && sc.Value != "" {
			id = sc.Value
		}
		if sc, e := r.Cookie(ageName); e == nil && sc.Value != "" {
			age = true
		}
		return
	}
	setCookies := func(id string) {
		idName, ageName := cookieNameBase+"-"+proto, cookieAgeBase+"-"+proto
		http.SetCookie(w, &http.Cookie{
			Name: idName, Value: id, Path: cookiePath, MaxAge: cookieMaxAge})
		now := time.Now().Format(time.RFC822)
		http.SetCookie(w, &http.Cookie{
			Name: ageName, Value: now, Path: cookiePath, MaxAge: cookieRefreshAge})
	}

	// check for an existing cookie whose name matches the protocol
	id, age := getCookies()
	if id != "" {
		// refresh both cookies if the date cookie has expired
		if !age {
			setCookies(id)
		}
		return id
	}

	// no session cookie: start a new session with a new ID
	// poor man's UUID for the session in local mode: time since startup.
	sid := strconv.FormatInt(int64(time.Now().Sub(startTime)), 36)
	// if we're on Heroku infrastructure, we use the request ID
	if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
		sid = requestID
	}
	log.Printf("No session cookie found, created new session ID %q", sid)
	setCookies(sid)
	return sid
}

// load: load the session for the current connection.
func (s *session) load(w http.ResponseWriter, r *http.Request) {
	// get the stored session
	s.ss = storage.LoadSession(s.sid)

	// reset the session if requested
	matches := selectEndpointRegexp.FindStringSubmatch(r.URL.Path)
	if *debugLog {
		log.Printf("Session Load: URI %q matches: %v", r.URL.Path, matches)
	}
	if matches != nil {
		if len(matches[2]) > 0 {
			s.ss.SelectPuzzle(matches[2])
			log.Printf("Selected session %v puzzle %q at step %d.", s.sid, s.name(), s.step())
		}
		if matches[1] == "reset" {
			s.ss.RemoveAllSteps()
			log.Printf("Reset session %v puzzle %q to step %d", s.sid, s.name(), s.step())
		}
	}
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
