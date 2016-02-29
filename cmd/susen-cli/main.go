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

// Command-line client for susen.go puzzle utilities
package main

import (
	"encoding/json"
	"fmt"
	"github.com/ancientHacker/susen.go/Godeps/_workspace/src/github.com/garyburd/redigo/redis"
	"github.com/ancientHacker/susen.go/puzzle"
	"io"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"
)

func main() {
	// establish redis connection
	redisInit()
	if err := redisConnect(); err != nil {
		shutdown(startupFailureShutdown)
	}
	defer redisClose()

	// catch signals
	shutdownOnSignal()

	// serve
	err := listener(os.Stdout, os.Stdin)
	if err != nil {
		log.Printf("CLI failure: %v", err)
		shutdown(listenerFailureShutdown)
	}
}

/*

CLI listener

*/

type request struct {
	inline  string
	command string
	args    []string
}

// listener reads lines and dispatches them to handlers
func listener(out *os.File, in *os.File) error {
	// if we are on a terminal, we do prompting
	// (see http://stackoverflow.com/questions/22744443/ for source)
	prompt := false
	if stat, _ := out.Stat(); (stat.Mode() & os.ModeCharDevice) != 0 {
		prompt = true
	}

	input := make([]byte, 4096)
	for {
		if prompt {
			fmt.Fprintf(out, "susen> ")
		}
		n, err := in.Read(input)
		switch err {
		case nil:
			r := &request{inline: strings.Trim(string(input[:n]), " \t\r\n")}
			args := strings.Split(r.inline, " ")
			r.command = strings.ToLower(args[0])
			switch r.command {
			case "":
				continue
			case "quit":
				fallthrough
			case "exit":
				return nil
			}
			for _, arg := range args[1:] {
				if len(arg) > 0 {
					r.args = append(r.args, strings.ToLower(arg))
				}
			}
			dispatchCommand(out, r)
		case io.EOF:
			// ignore any input before the EOF
			if prompt {
				fmt.Fprintf(out, " (EOF)\n")
			}
			return nil
		default:
			if prompt {
				fmt.Fprintf(out, " (read error)\n")
			}
			return err
		}
	}
}

// command dispatching
type commandInfo struct {
	command     string
	argInfo     string
	description string
	handler     func(*susenSession, *os.File, *request)
}

// the command dispatch info is sorted for easy usage printing,
// and then hashed for rapid dispatching
var (
	dispatchInfo  []commandInfo
	dispatchTable map[string]*commandInfo
)

func init() {
	dispatchInfo = []commandInfo{
		{"assign", "index value", "assign a value to a square", assignHandler},
		{"session", "[sessionID]", "get/set session info", summaryHandler},
		{"back", "", "go back one solution step", backHandler},
		{"hints", "on|off", "show hints in puzzle state", hintsHandler},
		{"markdown", "on|off", "format output in Markdown", markdownHandler},
		{"reset", "[puzzleID]", "reset this or another puzzle", stateHandler},
		{"state", "", "show current puzzle state", stateHandler},
		{"summary", "", "show current session summary", summaryHandler},
	}
	dispatchTable = make(map[string]*commandInfo, len(dispatchInfo))
	for i := range dispatchInfo {
		dispatchTable[dispatchInfo[i].command] = &dispatchInfo[i]
	}
}

func dispatchCommand(w *os.File, r *request) {
	defer func() {
		if err := recover(); err != nil {
			errorHandler(err, w, r)
		}
	}()

	session := sessionSelect(w, r)
	ci := dispatchTable[r.command]
	if ci == nil {
		usageHandler(fmt.Sprintf("%q is not a known command", r.command), w, r)
	} else {
		ci.handler(session, w, r)
	}
}

/*

request handlers

*/

// client state
var (
	useMarkdown  = false
	showBindings = true
)

func markdownHandler(session *susenSession, w *os.File, r *request) {
	if len(r.args) > 0 {
		switch r.args[0] {
		case "on":
			useMarkdown = true
			stateHandler(session, w, r)
		case "off":
			useMarkdown = false
			stateHandler(session, w, r)
		default:
			usageHandler(fmt.Sprintf("argument to %s must be 'on' or 'off'", r.command), w, r)
		}
	} else {
		if useMarkdown {
			fmt.Fprintf(w, "Markdown is on\n")
		} else {
			fmt.Fprintf(w, "Markdown is off\n")
		}
	}
}

func hintsHandler(session *susenSession, w *os.File, r *request) {
	if len(r.args) > 0 {
		switch r.args[0] {
		case "on":
			showBindings = true
			stateHandler(session, w, r)
		case "off":
			showBindings = false
			stateHandler(session, w, r)
		default:
			usageHandler(fmt.Sprintf("argument to %s must be 'on' or 'off'", r.command), w, r)
		}
	} else {
		if showBindings {
			fmt.Fprintf(w, "Hints are on\n")
		} else {
			fmt.Fprintf(w, "Hints are off\n")
		}
	}
}

func backHandler(session *susenSession, w *os.File, r *request) {
	session.removeStep()
	stateHandler(session, w, r)
}

func assignHandler(session *susenSession, w *os.File, r *request) {
	var choice puzzle.Choice
	var err error

	if len(r.args) != 2 {
		usageHandler(fmt.Sprintf("%s requires two arguments", r.command), w, r)
		return
	}

	// compute the index
	idx := r.args[0]
	if row := int(idx[0] - 'a'); row < 0 || row >= session.summary.SideLength {
		usageHandler(fmt.Sprintf("%s index (%s) row is out of range", r.command, idx), w, r)
		return
	} else if col, err := strconv.Atoi(idx[1:]); err != nil {
		usageHandler(fmt.Sprintf("%s index (%s) column is not a number", r.command, idx), w, r)
		return
	} else if col < 1 || col > session.summary.SideLength {
		usageHandler(fmt.Sprintf("%s index (%s) column is out of range", r.command, idx), w, r)
		return
	} else {
		choice.Index = (session.summary.SideLength * row) + col
	}

	// read the value
	choice.Value, err = strconv.Atoi(r.args[1])
	if err != nil {
		usageHandler(fmt.Sprintf("%s value (%s)	must be a number", r.command, r.args[1]), w, r)
		return
	}

	update, e := session.puzzle.Assign(choice)
	if e != nil {
		fmt.Fprintf(w, "Assign failed: %v\n", e)
	} else {
		session.addStep()
		if update.Errors != nil {
			log.Printf("Assign to %s:%q gave errors; step %d is unsolvable.",
				session.SID, session.PID, session.Step)
			fmt.Fprintf(w, "Assign succeeded but made puzzle unsolvable:\n")
		} else {
			fmt.Fprintf(w, "Assign succeeded:\n")
		}
		stateHandler(session, w, r)
	}
}

func stateHandler(session *susenSession, w *os.File, r *request) {
	if useMarkdown {
		fmt.Fprintf(w, "%s%s", session.puzzle.ValuesMarkdown(showBindings), session.puzzle.ErrorsMarkdown())
	} else {
		fmt.Fprintf(w, "%s%s", session.puzzle.ValuesString(showBindings), session.puzzle.ErrorsString())
	}
}

func summaryHandler(session *susenSession, w *os.File, r *request) {
	fmt.Fprintf(w, "Session %q solving puzzle %q on solution step %d\n",
		session.SID, session.PID, session.Step)
	sum, err := session.puzzle.Summary()
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(w, "Geometry: %v; Side length: %v; ", sum.Geometry, sum.SideLength)
	filled, empty := 0, 0
	for _, val := range sum.Values {
		if val == 0 {
			empty++
		} else {
			filled++
		}
	}
	fmt.Fprintf(w, "Assigned squares: %d; Empty squares: %d\n", filled, empty)
}

func usageHandler(msg string, w *os.File, r *request) {
	fmt.Fprintf(w, "Error: %s\nUsage:\n", msg)
	for _, ci := range dispatchInfo {
		fmt.Fprintf(w, "    %8s %-11s\t%s\n", ci.command, ci.argInfo, ci.description)
	}
	fmt.Fprintf(w, "  and 'quit' or EOF to exit.\n")
}

func errorHandler(err interface{}, w *os.File, r *request) {
	fmt.Fprintf(w, "Panic executing %q: %v\n", r, err)
	log.Printf("Server error executing %q: %v\n", r, err)
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

// cookie for the command line
var defaultCookie string

var (
	startTime = time.Now() // instance start-up time
)

// getCookie gets the session cookie, or sets a new one.  It
// returns the session ID associated with the cookie.
func getCookie(w *os.File, r *request) string {
	// look to see if the user is specifying a cookie
	if r.command == "session" && len(r.args) > 0 {
		defaultCookie = r.args[0]
	}

	// look for an existing session cookie
	if len(defaultCookie) != 0 {
		return defaultCookie
	}

	// no session cookie: start a new session with a new ID
	// poor man's UUID for the session in local mode: time since startup.
	sid := strconv.FormatInt(int64(time.Now().Sub(startTime)), 36)
	log.Printf("No session cookie found, created new session ID %q", sid)
	defaultCookie = sid
	return sid
}

// sessionSelect: find or create the session for the current connection.
func sessionSelect(w *os.File, r *request) *susenSession {
	id := getCookie(w, r)
	// check to see if this is a force reset of the session
	forceReset, resetID := r.command == "reset", ""
	if forceReset && len(r.args) > 0 {
		resetID = r.args[0]
	}
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
		log.Printf("Failed to get summary of %s:%q step %d: %v", session.SID, session.PID, session.Step, err)
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

persistence layer

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
