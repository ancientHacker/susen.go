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
	"encoding/json"
	"fmt"
	"github.com/ancientHacker/susen.go/Godeps/_workspace/src/github.com/garyburd/redigo/redis"
	"github.com/ancientHacker/susen.go/client"
	"github.com/ancientHacker/susen.go/puzzle"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

func main() {
	// client initialization
	if err := client.VerifyResources(); err != nil {
		log.Printf("Error during client initialization: %v", err)
		shutdown(startupFailureShutdown)
	}
	// establish redis connection
	redisInit()
	if err := redisConnect(); err != nil {
		shutdown(startupFailureShutdown)
	}
	// no deferred close; this function never terminates!
	// for abnormal terminations, see shutdown

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

func (session *susenSession) apiHandler(w http.ResponseWriter, r *http.Request) {
	sendState := func() {
		session.puzzle.StateHandler(w, r)
		log.Printf("Returned current state for %s:%q step %d", session.SID, session.PID, session.Step)
	}
	sendSummary := func() {
		session.puzzle.SummaryHandler(w, r)
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
			session.startPuzzle("")
			sendState()
		} else {
			sendNotAllowed()
		}
	case "back":
		if r.Method == "GET" {
			session.removeStep()
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
			update, e := session.puzzle.AssignHandler(w, r)
			if e != nil {
				log.Printf("Assign to %s:%q step %d failed: %v", session.SID, session.PID, session.Step, e)
			} else {
				session.addStep()
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

func (session *susenSession) solverHandler(w http.ResponseWriter, r *http.Request) {
	body := client.SolverPage(session.SID, session.PID, session.summary)
	hs := w.Header()
	hs.Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(body))
	log.Printf("Returned solver page for %s:%q step %d.", session.SID, session.PID, session.Step)
}

func (session *susenSession) homeHandler(w http.ResponseWriter, r *http.Request) {
	var others []string
	for k := range puzzleSummaries {
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

func (session *susenSession) rootHandler(w http.ResponseWriter, r *http.Request) {
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

// A susenSession the user's current step in a puzzle solution.
// Behind the scenes, we persist all the prior steps the user has
// taken, so he can go back (undo) prior choices.
type susenSession struct {
	SID     string          // session ID
	PID     string          // ID of puzzle being solved
	Step    int             // current step
	summary *puzzle.Summary // summary upon arriving at current step
	puzzle  *puzzle.Puzzle  // puzzle for current step
	Created string          // RFC3339 time when the session was created
	Saved   string          // RFC3339 time when the session was last saved
}

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
func sessionSelect(w http.ResponseWriter, r *http.Request) *susenSession {
	// check to see if this is a force reset of the session
	forceReset, resetID := false, ""
	if strings.HasPrefix(r.URL.Path, "/reset/") {
		forceReset = true
		resetID = r.URL.Path[len("/reset/"):]
	}
	id := getCookie(w, r)
	// create an in-memory session with this cookie
	session := &susenSession{SID: id, Created: time.Now().Format(time.RFC3339)}
	// load session from storage if possible, otherwise just initialize it
	if session.redisLookup() {
		log.Printf("Found session %v, puzzle %q, on step %d.", session.SID, session.PID, session.Step)
		if forceReset {
			session.startPuzzle(resetID)
		} else {
			session.redisLoadStep()
		}
	} else if forceReset {
		session.startPuzzle(resetID)
	} else {
		session.startPuzzle(defaultPuzzleID)
	}
	return session
}

// startPuzzle: set the puzzle ID for the current session and
// clear any existing solver steps for that puzzle ID.  If the
// given puzzle ID is empty, try using the session's current
// puzzle ID.  If the given puzzle ID is the special value
// "default" (or unknown), use the default puzzle ID.
func (session *susenSession) startPuzzle(pid string) {
	// change to the given pid, making sure it's valid
	if pid == "" {
		pid = session.PID
	} else if pid == "default" {
		pid = defaultPuzzleID
	}
	session.summary = puzzleSummaries[pid]
	if session.summary != nil {
		session.PID = pid
	} else {
		session.PID, session.summary = defaultPuzzleID, puzzleSummaries[defaultPuzzleID]
	}

	// make the puzzle for the summary
	p, e := puzzle.New(session.summary)
	if e != nil {
		log.Printf("Failed to create puzzle %q: %v", pid, e)
		panic(e)
	}
	session.puzzle = p
	session.redisStartPuzzle()
	log.Printf("Reset session %v to start solving puzzle %q.", session.SID, session.PID)
}

// addStep: add a new current step with the current puzzle.
func (session *susenSession) addStep() {
	summary, err := session.puzzle.Summary()
	if err != nil {
		log.Printf("Failed to get summary of %s:%q step %d: %v",
			session.SID, session.PID, session.Step, err)
		panic(err)
	}
	session.summary = summary
	session.redisAddStep()
	log.Printf("Added session %v:%v step %d.", session.SID, session.PID, session.Step)
}

// removeStep: remove the last step and restore the prior step's
// puzzle.
func (session *susenSession) removeStep() {
	if session.Step > 1 {
		session.redisRemoveStep()
		log.Printf("Reverted session %v:%v to step %d.",
			session.SID, session.PID, session.Step)
	}
}

/*

session persistence layer

*/

// puzzle data
var (
	defaultPuzzleID = "standard-1"
	puzzleSummaries = map[string]*puzzle.Summary{
		"standard-1": &puzzle.Summary{
			Geometry:   puzzle.StandardGeometryName,
			SideLength: 9,
			Values: []int{
				4, 0, 0, 0, 0, 3, 5, 0, 2,
				0, 0, 9, 5, 0, 6, 3, 4, 0,
				0, 0, 0, 0, 0, 0, 0, 0, 8,
				0, 0, 0, 0, 3, 4, 8, 6, 0,
				0, 0, 4, 6, 0, 5, 2, 0, 0,
				0, 2, 8, 7, 9, 0, 0, 0, 0,
				9, 0, 0, 0, 0, 0, 0, 0, 0,
				0, 8, 7, 3, 0, 2, 9, 0, 0,
				5, 0, 2, 9, 0, 0, 0, 0, 6,
			}},
		"standard-2": &puzzle.Summary{
			Geometry:   puzzle.StandardGeometryName,
			SideLength: 9,
			Values: []int{
				0, 1, 0, 5, 0, 6, 0, 2, 0,
				0, 0, 0, 0, 0, 3, 0, 1, 8,
				0, 0, 0, 0, 7, 0, 0, 0, 6,
				0, 0, 5, 0, 0, 0, 0, 3, 0,
				0, 0, 8, 0, 9, 0, 7, 0, 0,
				0, 6, 0, 0, 0, 0, 4, 0, 0,
				5, 0, 0, 0, 4, 0, 0, 0, 0,
				6, 4, 0, 2, 0, 0, 0, 0, 0,
				0, 3, 0, 9, 0, 1, 0, 8, 0,
			}},
		"standard-3": &puzzle.Summary{
			Geometry:   puzzle.StandardGeometryName,
			SideLength: 9,
			Values: []int{
				9, 0, 0, 4, 5, 0, 0, 0, 8,
				0, 2, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 1, 7, 2, 4, 0, 0,
				0, 7, 9, 0, 0, 0, 6, 8, 0,
				2, 0, 0, 0, 0, 0, 0, 0, 5,
				0, 4, 3, 0, 0, 0, 2, 7, 0,
				0, 0, 8, 3, 2, 5, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 6, 0,
				4, 0, 0, 0, 1, 6, 0, 0, 3,
			}},
		"standard-4": &puzzle.Summary{
			Geometry:   puzzle.StandardGeometryName,
			SideLength: 9,
			Values: []int{
				9, 4, 8, 0, 5, 0, 2, 0, 0,
				0, 0, 7, 8, 0, 3, 0, 0, 1,
				0, 5, 0, 0, 7, 0, 0, 0, 0,
				0, 7, 0, 0, 0, 0, 3, 0, 0,
				2, 0, 0, 6, 0, 5, 0, 0, 4,
				0, 0, 5, 0, 0, 0, 0, 9, 0,
				0, 0, 0, 0, 6, 0, 0, 1, 0,
				3, 0, 0, 5, 0, 9, 7, 0, 0,
				0, 0, 6, 0, 1, 0, 4, 2, 3,
			}},
		"standard-5": &puzzle.Summary{
			Geometry:   puzzle.StandardGeometryName,
			SideLength: 9,
			Values: []int{
				0, 0, 0, 0, 0, 0, 0, 0, 0,
				9, 0, 0, 5, 0, 7, 0, 3, 0,
				0, 0, 0, 1, 0, 0, 6, 0, 7,
				0, 4, 0, 0, 6, 0, 0, 8, 2,
				6, 7, 0, 0, 0, 0, 0, 1, 3,
				3, 8, 0, 0, 1, 0, 0, 9, 0,
				7, 0, 5, 0, 0, 8, 0, 0, 0,
				0, 2, 0, 3, 0, 9, 0, 0, 8,
				0, 0, 0, 0, 0, 0, 0, 0, 0,
			}},
		"standard-6": &puzzle.Summary{
			Geometry:   puzzle.StandardGeometryName,
			SideLength: 9,
			Values: []int{
				2, 0, 0, 8, 0, 0, 0, 5, 0,
				0, 8, 5, 0, 0, 0, 0, 0, 0,
				0, 3, 6, 7, 5, 0, 0, 0, 1,
				0, 0, 3, 0, 4, 0, 0, 9, 8,
				0, 0, 0, 3, 0, 5, 0, 0, 0,
				4, 1, 0, 0, 6, 0, 7, 0, 0,
				5, 0, 0, 0, 0, 7, 1, 2, 0,
				0, 0, 0, 0, 0, 0, 5, 6, 0,
				0, 2, 0, 0, 0, 0, 0, 0, 4,
			}},
		"rectangular-1": &puzzle.Summary{
			Geometry:   puzzle.RectangularGeometryName,
			SideLength: 6,
			Values: []int{
				0, 4, 5, 1, 6, 0,
				3, 0, 0, 0, 0, 0,
				0, 5, 0, 6, 2, 1,
				1, 0, 2, 3, 4, 0,
				5, 0, 0, 2, 1, 6,
				6, 0, 0, 0, 0, 0,
			}},
		"rectangular-2": &puzzle.Summary{
			Geometry:   puzzle.RectangularGeometryName,
			SideLength: 6,
			Values: []int{
				0, 0, 0, 2, 6, 0,
				2, 0, 3, 0, 0, 0,
				0, 5, 0, 0, 0, 6,
				3, 2, 6, 0, 0, 1,
				0, 0, 4, 0, 0, 0,
				0, 0, 0, 5, 1, 4,
			}},
		"rectangular-3": &puzzle.Summary{
			Geometry:   puzzle.RectangularGeometryName,
			SideLength: 12,
			Values: []int{
				5, 7, 0, 6, 0, 0, 0, 0, 0, 1, 11, 12,
				11, 0, 0, 0, 0, 0, 10, 0, 0, 0, 0, 3,
				8, 0, 9, 0, 0, 0, 1, 0, 5, 7, 0, 0,
				0, 0, 4, 2, 10, 11, 0, 0, 12, 0, 0, 8,
				0, 0, 0, 0, 9, 6, 0, 1, 7, 0, 0, 0,
				0, 9, 7, 0, 0, 0, 0, 2, 11, 0, 0, 0,
				0, 0, 0, 8, 7, 0, 0, 0, 0, 11, 3, 0,
				0, 0, 0, 11, 3, 0, 2, 5, 0, 0, 0, 0,
				9, 0, 0, 3, 0, 0, 11, 8, 10, 6, 0, 0,
				0, 0, 3, 7, 0, 10, 0, 0, 0, 12, 0, 2,
				2, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 11,
				6, 11, 12, 0, 0, 0, 0, 0, 3, 0, 9, 4,
			}},
		"rectangular-4": &puzzle.Summary{
			Geometry:   puzzle.RectangularGeometryName,
			SideLength: 12,
			Values: []int{
				0, 11, 3, 0, 0, 0, 0, 0, 0, 6, 0, 0,
				0, 7, 0, 0, 12, 0, 4, 0, 0, 3, 10, 8,
				4, 6, 0, 0, 10, 11, 0, 0, 1, 0, 0, 7,
				0, 0, 8, 9, 2, 0, 0, 0, 5, 0, 0, 0,
				0, 0, 0, 0, 0, 9, 6, 0, 12, 8, 11, 0,
				0, 5, 0, 0, 3, 0, 0, 11, 0, 9, 0, 0,
				0, 0, 4, 0, 8, 0, 0, 9, 0, 0, 7, 0,
				0, 9, 7, 3, 0, 10, 12, 0, 0, 0, 0, 0,
				0, 0, 0, 11, 0, 0, 0, 1, 3, 12, 0, 0,
				3, 0, 0, 7, 0, 0, 8, 2, 0, 0, 4, 1,
				2, 8, 5, 0, 0, 12, 0, 4, 0, 0, 3, 0,
				0, 0, 9, 0, 0, 0, 0, 0, 0, 7, 12, 0,
			}},
	}
)

// keys for persisted session values
var (
	sessionKeyFormat = "session:"
	stepsKey         = "step:"
)

// current connection data
var (
	rdc     redis.Conn // open connection, if any
	rdUrl   string     // URL for the open connection
	rdEnv   string     // environment key prefix
	rdMutex sync.Mutex // prevent concurrent connection use
)

// redisInit - look up redis info from the environment
func redisInit() {
	url := os.Getenv("REDISTOGO_URL")
	db := os.Getenv("REDISTOGO_DB")
	env := os.Getenv("REDISTOGO_ENV")
	if db == "" {
		db = "0" // default database
	}
	if url == "" {
		rdUrl = "redis://localhost:6379/" + db
	} else {
		rdUrl = url + db
	}
	if env == "" {
		if url == "" {
			rdEnv = "local"
		} else {
			rdEnv = "dev"
		}
	} else {
		rdEnv = env
	}
}

// redisConnect: connect to the given Redis URL.  Returns the
// connection, if successful, nil otherwise.
func redisConnect() error {
	conn, err := redis.DialURL(rdUrl)
	if err == nil {
		log.Printf("Connected to redis at %q (env: %q)", rdUrl, rdEnv)
		rdc = conn
		return nil
	}
	log.Printf("Can't connect to redis server at %q", rdUrl)
	return err
}

// redisClose: close the given Redis connection.
func redisClose() {
	if rdc != nil {
		rdc.Close()
		log.Print("Closed connection to redis.")
		rdc = nil
	}
}

// redisExecute: execute the body with the redis connection.
// Meant to be used inside a handler, because errors in execution
// will panic back to the handler level.
func redisExecute(body func() error) {
	// wrap the body against runtime and database failures
	wrapper := func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				if e, ok := r.(error); ok {
					err = e
				} else {
					err = fmt.Errorf("%v", r)
				}
				log.Printf("Caught panic during redisExecute: %v", err)
			}
		}()
		// Because redis connections can go away without warning,
		// we ping to make sure the connection is alive, and try
		// to reconnect if not.
		if _, err := rdc.Do("PING"); err != nil {
			log.Printf("PING failure with redis: %v", err)
			redisClose()
			err = redisConnect()
			if err != nil {
				log.Printf("Failed to reconnect to redis at %q", rdUrl)
				return err
			}
		}
		// connection is good; run the body
		return body()
	}
	// grab the mutex and execute the body
	rdMutex.Lock()
	defer func(err error) {
		rdMutex.Unlock()
		if err != nil {
			panic(err)
		}
	}(wrapper())
}

// redisKey - returns the session key
func (session *susenSession) redisKey() string {
	return rdEnv + ":SID:" + session.SID
}

// redisStepsKey - returns the key for the session's step array
func (session *susenSession) redisStepsKey() string {
	return session.redisKey() + ":Steps"
}

// redisLookup: lookup a session for an ID
func (session *susenSession) redisLookup() (found bool) {
	body := func() error {
		vals, err := redis.Values(rdc.Do("HGETALL", session.redisKey()))
		if len(vals) > 0 {
			if err := redis.ScanStruct(vals, session); err != nil {
				log.Printf("Redis error on parse of saved session %q: %v", session.SID, err)
				return err
			}
			found = true
			return nil
		}
		if err != nil {
			log.Printf("Redis error on GET of session %q pid: %v", session.SID, err)
			return err
		}
		log.Printf("No redis saved summary for session %q", session.SID)
		return nil
	}
	redisExecute(body)
	return
}

// redisLoadStep: load the current step from the saved summary
func (session *susenSession) redisLoadStep() {
	var bytes []byte
	body := func() (err error) {
		bytes, err = redis.Bytes(rdc.Do("LINDEX", session.redisStepsKey(), -1))
		if err != nil {
			log.Printf("Error on load of %s:%q step %d: %v", session.SID, session.PID, session.Step, err)
		}
		return
	}
	redisExecute(body)
	session.redisUnmarshalStep(bytes)
}

// redisStartPuzzle: save a session that's just starting a puzzle
func (session *susenSession) redisStartPuzzle() {
	session.Saved = time.Now().Format(time.RFC3339)
	session.Step = 1
	bytes := session.redisMarshalStep()
	body := func() (err error) {
		rdc.Send("HMSET", redis.Args{}.Add(session.redisKey()).AddFlat(session)...)
		rdc.Send("DEL", session.redisStepsKey())
		_, err = rdc.Do("RPUSH", session.redisStepsKey(), bytes)
		if err != nil {
			log.Printf("Redis error on save of session %q after reset: %v", session.SID, err)
		}
		return
	}
	redisExecute(body)
}

// redisAddStep: add the current step to the saved summary
func (session *susenSession) redisAddStep() {
	session.Saved = time.Now().Format(time.RFC3339)
	session.Step++
	bytes := session.redisMarshalStep()
	body := func() (err error) {
		rdc.Send("HMSET", redis.Args{}.Add(session.redisKey()).AddFlat(session)...)
		_, err = rdc.Do("RPUSH", session.redisStepsKey(), bytes)
		if err != nil {
			log.Printf("Redis error on save of %s:%q step %d: %v", session.SID, session.PID, session.Step, err)
		}
		return
	}
	redisExecute(body)
}

// redisRemoveStep: remove the last step from the saved session
// and load the current step
func (session *susenSession) redisRemoveStep() {
	var bytes []byte
	session.Saved = time.Now().Format(time.RFC3339)
	session.Step--
	session.summary = nil // free the current step's summary
	body := func() (err error) {
		rdc.Send("HMSET", redis.Args{}.Add(session.redisKey()).AddFlat(session)...)
		rdc.Send("LTRIM", session.redisStepsKey(), 0, -2)
		bytes, err = redis.Bytes(rdc.Do("LINDEX", session.redisStepsKey(), -1))
		if err != nil {
			log.Printf("Error on remove to %s:%q step %d: %v", session.SID, session.PID, session.Step, err)
		}
		return
	}
	redisExecute(body)
	session.redisUnmarshalStep(bytes)
}

// redisMarshalStep - get JSON for the current step
func (session *susenSession) redisMarshalStep() []byte {
	bytes, err := json.Marshal(session.summary)
	if err != nil {
		log.Printf("Failed to marshal summary of %s:%q step %d (%+v) as JSON: %v",
			session.SID, session.PID, session.Step, *session.summary, err)
		panic(err)
	}
	return bytes
}

// redisUnmarshalStep - get puzzle for the saved step
func (session *susenSession) redisUnmarshalStep(bytes []byte) {
	var summary *puzzle.Summary
	err := json.Unmarshal(bytes, &summary)
	if err != nil {
		log.Printf("Failed to unmarshal saved JSON of %s:%q step %d: %v",
			session.SID, session.PID, session.Step, err)
		panic(err)
	}
	session.summary = summary
	session.puzzle, err = puzzle.New(session.summary)
	if err != nil {
		log.Printf("Failed to create puzzle for %s:%q step %d (%+v): %v",
			session.SID, session.PID, session.Step, *session.summary, err)
		panic(err)
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
	redisFailureShutdown
	caughtSignalShutdown
	listenerFailureShutdown
)

// for testing, allow alternate forms of shutdown
var alternateShutdown func(reason shutdownCause)

// shutdown: process exit with logging.
func shutdown(reason shutdownCause) {
	// Get redis mutex and keep it to block other handlers from
	// running.  Close the redis connection.
	rdMutex.Lock()
	redisClose()

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
	case redisFailureShutdown:
		log.Fatal("Exiting: redis failure.")
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
