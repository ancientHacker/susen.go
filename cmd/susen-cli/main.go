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
	"fmt"
	"github.com/ancientHacker/susen.go/puzzle"
	"github.com/ancientHacker/susen.go/storage"
	"io"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
)

func main() {
	// storage initialization
	if err := storage.Connect(); err != nil {
		log.Printf("Error during storage initialization: %v", err)
		shutdown(startupFailureShutdown)
	}
	defer storage.Close()

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
	handler     func(*userSession, *os.File, *request)
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

func markdownHandler(session *userSession, w *os.File, r *request) {
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

func hintsHandler(session *userSession, w *os.File, r *request) {
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

func backHandler(session *userSession, w *os.File, r *request) {
	session.RemoveStep()
	stateHandler(session, w, r)
}

func assignHandler(session *userSession, w *os.File, r *request) {
	var choice puzzle.Choice
	var err error

	if len(r.args) != 2 {
		usageHandler(fmt.Sprintf("%s requires two arguments", r.command), w, r)
		return
	}

	// compute the index
	idx := r.args[0]
	if row := int(idx[0] - 'a'); row < 0 || row >= session.Summary.SideLength {
		usageHandler(fmt.Sprintf("%s index (%s) row is out of range", r.command, idx), w, r)
		return
	} else if col, err := strconv.Atoi(idx[1:]); err != nil {
		usageHandler(fmt.Sprintf("%s index (%s) column is not a number", r.command, idx), w, r)
		return
	} else if col < 1 || col > session.Summary.SideLength {
		usageHandler(fmt.Sprintf("%s index (%s) column is out of range", r.command, idx), w, r)
		return
	} else {
		choice.Index = (session.Summary.SideLength * row) + col
	}

	// read the value
	choice.Value, err = strconv.Atoi(r.args[1])
	if err != nil {
		usageHandler(fmt.Sprintf("%s value (%s)	must be a number", r.command, r.args[1]), w, r)
		return
	}

	update, e := session.Puzzle.Assign(choice)
	if e != nil {
		fmt.Fprintf(w, "Assign failed: %v\n", e)
	} else {
		session.AddStep()
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

func stateHandler(session *userSession, w *os.File, r *request) {
	if useMarkdown {
		fmt.Fprintf(w, "%s%s", session.Puzzle.ValuesMarkdown(showBindings), session.Puzzle.ErrorsMarkdown())
	} else {
		fmt.Fprintf(w, "%s%s", session.Puzzle.ValuesString(showBindings), session.Puzzle.ErrorsString())
	}
}

func summaryHandler(session *userSession, w *os.File, r *request) {
	fmt.Fprintf(w, "Session %q solving puzzle %q on solution step %d\n",
		session.SID, session.PID, session.Step)
	sum, err := session.Puzzle.Summary()
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

type userSession struct {
	storage.Session
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
func sessionSelect(w *os.File, r *request) *userSession {
	id := getCookie(w, r)
	// check to see if this is a force reset of the session
	forceReset, resetID := r.command == "reset", ""
	if forceReset && len(r.args) > 0 {
		resetID = r.args[0]
	}
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
	redisFailureShutdown
	caughtSignalShutdown
	listenerFailureShutdown
)

// for testing, allow alternate forms of shutdown
var alternateShutdown func(reason shutdownCause)

// shutdown: process exit with logging.
func shutdown(reason shutdownCause) {
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
