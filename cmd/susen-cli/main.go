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
	"bytes"
	"fmt"
	"github.com/ancientHacker/susen.go/puzzle"
	"github.com/ancientHacker/susen.go/storage"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

func main() {
	// log initialization
	log.SetOutput(os.Stderr)
	// storage initialization
	cacheId, databaseId, err := storage.Connect()
	if err != nil {
		log.Printf("Error during storage initialization: %v", err)
		os.Exit(1)
	}
	defer storage.Close()
	log.Printf("Connected to cache at %q", cacheId)
	log.Printf("Connected to database at %q", databaseId)

	// serve
	err = listener(os.Stdout, os.Stdin)
	if err != nil {
		log.Printf("CLI failure: %v", err)
		os.Exit(1)
	}
	os.Exit(0)
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

var (
	// client preferences
	useMarkdown  = false
	showBindings = true
)

func markdownHandler(s *session, w io.Writer, r *request) {
	// check the args
	if len(r.args) > 1 {
		usageHandler(fmt.Sprintf("%s takes at most 1 argument", r.command), w, r)
		return
	}
	// process the request
	if len(r.args) == 1 {
		switch r.args[0] {
		case "on":
			useMarkdown = true
		case "off":
			useMarkdown = false
		default:
			usageHandler(fmt.Sprintf("argument to %s must be 'on' or 'off'", r.command), w, r)
		}
	}
	// provide feedback
	if useMarkdown {
		fmt.Fprintf(w, "Markdown is on\n")
	} else {
		fmt.Fprintf(w, "Markdown is off\n")
	}
}

func hintsHandler(s *session, w io.Writer, r *request) {
	// check the args
	if len(r.args) > 1 {
		usageHandler(fmt.Sprintf("%s takes at most 1 argument", r.command), w, r)
		return
	}
	// process the request
	if len(r.args) == 1 {
		switch r.args[0] {
		case "on":
			showBindings = true
		case "off":
			showBindings = false
		default:
			usageHandler(fmt.Sprintf("argument to %s must be 'on' or 'off'", r.command), w, r)
		}
	}
	// provide feedback
	if showBindings {
		fmt.Fprintf(w, "Hints are on\n")
	} else {
		fmt.Fprintf(w, "Hints are off\n")
	}
}

func backHandler(s *session, w io.Writer, r *request) {
	// check the args
	if len(r.args) > 0 {
		usageHandler(fmt.Sprintf("%s takes no arguments", r.command), w, r)
		return
	}
	if s.step() > 1 {
		s.ss.RemoveStep()
		solveHandler(s, w, r)
	} else {
		fmt.Fprintf(w, "No choices to undo.\n")
	}
}

func assignHandler(s *session, w io.Writer, r *request) {
	// check the args
	if len(r.args) != 2 {
		usageHandler(fmt.Sprintf("%s requires two arguments", r.command), w, r)
		return
	}

	// read the index part of the choice
	var choice puzzle.Choice
	var err error
	idx := r.args[0]
	if row := int(idx[0] - 'a'); row < 0 || row >= s.ss.Info.SideLength {
		usageHandler(fmt.Sprintf("%s index (%s) row is out of range", r.command, idx), w, r)
		return
	} else if col, err := strconv.Atoi(idx[1:]); err != nil {
		usageHandler(fmt.Sprintf("%s index (%s) column is not a number", r.command, idx), w, r)
		return
	} else if col < 1 || col > s.ss.Info.SideLength {
		usageHandler(fmt.Sprintf("%s index (%s) column is out of range", r.command, idx), w, r)
		return
	} else {
		choice.Index = (s.ss.Info.SideLength * row) + col
	}

	// read the value part of the choice
	choice.Value, err = strconv.Atoi(r.args[1])
	if err != nil {
		usageHandler(fmt.Sprintf("%s value (%s)	must be a number", r.command, r.args[1]), w, r)
		return
	}

	// do the assignment
	update, e := s.puzzle().Assign(choice)
	if e != nil {
		log.Printf("Assign of %+v at %s:%q step %d failed: %v",
			choice, s.sid, s.name(), s.step(), e)
	} else {
		if update.Errors != nil {
			log.Printf("Assign of %+v at %s:%q step %d made puzzle unsolvable.",
				choice, s.sid, s.name(), s.step())
		} else {
			log.Printf("Assign of %+v at %s:%q step %d left puzzle solvable",
				choice, s.sid, s.name(), s.step())
		}
		s.ss.AddStep(choice)
	}

	// provide feedback
	r.args = nil
	solveHandler(s, w, r)
}

func solveHandler(s *session, w io.Writer, r *request) {
	// check the args
	if len(r.args) > 1 {
		usageHandler(fmt.Sprintf("%s takes at most 1 argument", r.command), w, r)
		return
	}
	// if the user specified a puzzle, switch to it
	if len(r.args) == 1 {
		s.ss.SelectPuzzle(r.args[0])
		log.Printf("Selected session %v puzzle %q at step %d.", s.sid, s.name(), s.step())
	}
	// if the user requested a reset, perform it
	if r.command == "reset" {
		s.ss.RemoveAllSteps()
		log.Printf("Reset session %v puzzle %q to step %d", s.sid, s.name(), s.step())
	}
	// output the puzzle
	if useMarkdown {
		fmt.Fprintf(w, "%s%s",
			s.puzzle().ValuesMarkdown(showBindings),
			s.puzzle().ErrorsMarkdown())
	} else {
		fmt.Fprintf(w, "%s%s",
			s.puzzle().ValuesString(showBindings),
			s.puzzle().ErrorsString())
	}
}

func homeHandler(s *session, w io.Writer, r *request) {
	// check the args
	if len(r.args) > 1 {
		usageHandler(fmt.Sprintf("%s takes no arguments", r.command), w, r)
		return
	}
	// output the current puzzle summary
	fmt.Fprintf(w, "Session %q with current puzzle:\n", s.sid)
	fmt.Fprintf(w, "  %s [%s, %dx%d] (id: %s)\n\tSolved: %d; Remaining: %d\n",
		s.ss.Info.Name,
		s.ss.Info.Geometry, s.ss.Info.SideLength, s.ss.Info.SideLength,
		s.ss.Info.PuzzleId,
		len(s.ss.Info.Choices), s.ss.Info.Remaining)
	fmt.Fprintf(w, "\nOther puzzles (being solved first, most recent first):\n")
	// output the rest of the session puzzles
	infos := s.ss.GetInactivePuzzles()
	sort.Sort(storage.ByLatestSolutionView(infos))
	for _, info := range infos {
		fmt.Fprintf(w, "  %s [%s, %dx%d] (id: %s)\n\tSolved: %d; Remaining: %d\n",
			info.Name,
			info.Geometry, info.SideLength, info.SideLength,
			info.PuzzleId,
			len(info.Choices), info.Remaining)
	}
}

func usageHandler(msg string, w io.Writer, r *request) {
	fmt.Fprintf(os.Stderr, "Error: %s\nUsage:\n", msg)
	for _, ci := range dispatchInfo {
		fmt.Fprintf(os.Stderr, "    %8s %-11s\t%s\n", ci.command, ci.argInfo, ci.description)
	}
	fmt.Fprintf(os.Stderr, "  and 'quit' or EOF to exit.\n")
}

func errorHandler(err interface{}, w io.Writer, r *request) {
	log.Printf("Panic executing %+q: %v", r, err)
}

/*

session handling

*/

// cookie for the command line
var defaultCookie string

var (
	startTime = time.Now() // instance start-up time
)

// getCookie gets the session cookie, or sets a new one.  It
// returns the session ID associated with the cookie.
func getCookie(w io.Writer, r *request) string {
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
func sessionSelect(w io.Writer, r *request) *session {
	id := getCookie(w, r)
	return &session{sid: id, ss: storage.LoadSession(id)}
}

/*

CLI listener

*/

// bufsize (input read size) is a variable not a constant to
// allow testing with different values
var bufsize = 4096

type request struct {
	inline  string
	command string
	args    []string
}

// listener reads lines and dispatches them to handlers
func listener(out io.Writer, in io.Reader) error {
	// if we are on a terminal, we do prompting
	// (see http://stackoverflow.com/questions/22744443/ for source)
	prompt := false
	if infile, ok := in.(*os.File); ok {
		if stat, _ := infile.Stat(); (stat.Mode() & os.ModeCharDevice) != 0 {
			prompt = true
		}
	}

	// buffer the input and process it line by line
	input := new(bytes.Buffer)
	buf, start := make([]byte, bufsize), 0
	for {
		if input.Len() == 0 {
			if prompt {
				if start != 0 {
					fmt.Fprintf(out, "\nsusen> %s", buf[0:start])
				} else {
					fmt.Fprintf(out, "susen> ")
				}
			}
			n, err := in.Read(buf[start:])
			switch err {
			case nil:
				input.Write(buf[0 : start+n])
				start = 0
			case io.EOF:
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
		line, err := input.ReadString('\n')
		if err != nil && len(line) != 0 {
			// there's a partial line left in the buffer
			start = copy(buf, []byte(line))
			continue
		}
		r := &request{inline: strings.Trim(line, " \t\r\n")}
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
	}
}

// command dispatching
type commandInfo struct {
	command     string
	argInfo     string
	description string
	handler     func(*session, io.Writer, *request)
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
		{"back", "", "go back one solution step", backHandler},
		{"hints", "on|off", "show hints in puzzle state", hintsHandler},
		{"home", "", "show current session summary", homeHandler},
		{"markdown", "on|off", "format output in Markdown", markdownHandler},
		{"reset", "[name]", "reset current or another puzzle", solveHandler},
		{"session", "[sessionID]", "get/set session info", homeHandler},
		{"solve", "[name]", "work on current or another puzzle", solveHandler},
	}
	dispatchTable = make(map[string]*commandInfo, len(dispatchInfo))
	for i := range dispatchInfo {
		dispatchTable[dispatchInfo[i].command] = &dispatchInfo[i]
	}
}

func dispatchCommand(w io.Writer, r *request) {
	defer func() {
		if err := recover(); err != nil {
			errorHandler(err, w, r)
		}
	}()

	s := sessionSelect(w, r)
	ci := dispatchTable[r.command]
	if ci == nil {
		usageHandler(fmt.Sprintf("%q is not a known command", r.command), w, r)
	} else {
		ci.handler(s, w, r)
	}
}
