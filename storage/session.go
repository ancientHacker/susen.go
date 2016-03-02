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
	"encoding/json"
	"github.com/ancientHacker/susen.go/Godeps/_workspace/src/github.com/garyburd/redigo/redis"
	"github.com/ancientHacker/susen.go/puzzle"
	"log"
	"time"
)

// A Session tracks the user's current step in the solution of
// his current puzzle.  Behind the scenes, we persist all the
// prior steps the user has taken in this solution, so he can go
// back (undo) prior choices.
type Session struct {
	// these elements are persisted as part of the session
	SID     string // session ID
	PID     string // ID of puzzle being solved
	Step    int    // current step
	Created string // RFC3339 time when the session was created
	Saved   string // RFC3339 time when the session was last saved

	// these elements are presisted in the steps, serialized as JSON
	Summary *puzzle.Summary `redis:"-"` // summary upon arriving at current step
	Puzzle  *puzzle.Puzzle  `redis:"-"` // puzzle for current step
}

/*

session manipulation

*/

// StartPuzzle: set the puzzle ID for the current session and
// clear any existing solver steps for that puzzle ID.  If the
// given puzzle ID is empty, try using the session's current
// puzzle ID.  If the given puzzle ID is the special value
// "default" (or unknown), use the default puzzle ID.
func (session *Session) StartPuzzle(pid string) {
	// change to the given pid, making sure it's valid
	if pid == "" {
		pid = session.PID
	} else if pid == "default" {
		pid = defaultPuzzleID
	}
	session.Summary = puzzleSummaries[pid]
	if session.Summary != nil {
		session.PID = pid
	} else {
		session.PID, session.Summary = defaultPuzzleID, puzzleSummaries[defaultPuzzleID]
	}

	// make the puzzle for the summary
	p, e := puzzle.New(session.Summary)
	if e != nil {
		log.Printf("Failed to create puzzle %q: %v", pid, e)
		panic(e)
	}
	session.Puzzle = p

	// update the cache
	session.Saved = time.Now().Format(time.RFC3339)
	session.Step = 1
	bytes := session.marshalStep()
	body := func() (err error) {
		rdc.Send("HMSET", redis.Args{}.Add(session.key()).AddFlat(session)...)
		rdc.Send("DEL", session.stepsKey())
		_, err = rdc.Do("RPUSH", session.stepsKey(), bytes)
		if err != nil {
			log.Printf("Redis error on save of session %q after reset: %v", session.SID, err)
		}
		return
	}
	rdExecute(body)
	log.Printf("Reset session %v to start solving puzzle %q.", session.SID, session.PID)
}

// AddStep: add a new current step with the current puzzle.
func (session *Session) AddStep() {
	summary, err := session.Puzzle.Summary()
	if err != nil {
		log.Printf("Failed to get summary of %s:%q step %d: %v",
			session.SID, session.PID, session.Step, err)
		panic(err)
	}
	session.Summary = summary

	// update the cache
	session.Saved = time.Now().Format(time.RFC3339)
	session.Step++
	bytes := session.marshalStep()
	body := func() (err error) {
		rdc.Send("HMSET", redis.Args{}.Add(session.key()).AddFlat(session)...)
		_, err = rdc.Do("RPUSH", session.stepsKey(), bytes)
		if err != nil {
			log.Printf("Redis error on save of %s:%q step %d: %v", session.SID, session.PID, session.Step, err)
		}
		return
	}
	rdExecute(body)
	log.Printf("Added session %v:%v step %d.", session.SID, session.PID, session.Step)
}

// RemoveStep: remove the last step and restore the prior step's
// puzzle.
func (session *Session) RemoveStep() {
	if session.Step <= 1 {
		// nothing to do
		return
	}

	// load the puzzle from the cache
	var bytes []byte
	session.Saved = time.Now().Format(time.RFC3339)
	session.Step--
	session.Summary = nil // free the current step's summary
	body := func() (err error) {
		rdc.Send("HMSET", redis.Args{}.Add(session.key()).AddFlat(session)...)
		rdc.Send("LTRIM", session.stepsKey(), 0, -2)
		bytes, err = redis.Bytes(rdc.Do("LINDEX", session.stepsKey(), -1))
		if err != nil {
			log.Printf("Error on remove to %s:%q step %d: %v",
				session.SID, session.PID, session.Step, err)
		}
		return
	}
	rdExecute(body)
	session.unmarshalStep(bytes)
	log.Printf("Reverted session %v:%v to step %d.", session.SID, session.PID, session.Step)
}

// Lookup: lookup a session for an ID
func (session *Session) Lookup() (found bool) {
	body := func() error {
		vals, err := redis.Values(rdc.Do("HGETALL", session.key()))
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
	rdExecute(body)
	return
}

// LoadStep: load the current step from the saved summary
func (session *Session) LoadStep() {
	var bytes []byte
	body := func() (err error) {
		bytes, err = redis.Bytes(rdc.Do("LINDEX", session.stepsKey(), -1))
		if err != nil {
			log.Printf("Error on load of %s:%q step %d: %v", session.SID, session.PID, session.Step, err)
		}
		return
	}
	rdExecute(body)
	session.unmarshalStep(bytes)
}

/*

serialization of puzzle state into and out of the cache

*/

// marshalStep - get JSON for the current step
func (session *Session) marshalStep() []byte {
	bytes, err := json.Marshal(session.Summary)
	if err != nil {
		log.Printf("Failed to marshal summary of %s:%q step %d (%+v) as JSON: %v",
			session.SID, session.PID, session.Step, *session.Summary, err)
		panic(err)
	}
	return bytes
}

// unmarshalStep - get puzzle for the saved step
func (session *Session) unmarshalStep(bytes []byte) {
	var summary *puzzle.Summary
	err := json.Unmarshal(bytes, &summary)
	if err != nil {
		log.Printf("Failed to unmarshal saved JSON of %s:%q step %d: %v",
			session.SID, session.PID, session.Step, err)
		panic(err)
	}
	session.Summary = summary
	session.Puzzle, err = puzzle.New(session.Summary)
	if err != nil {
		log.Printf("Failed to create puzzle for %s:%q step %d (%+v): %v",
			session.SID, session.PID, session.Step, *session.Summary, err)
		panic(err)
	}
}

/*

session key generation

*/

// key - returns the session key
func (session *Session) key() string {
	return rdEnv + ":SID:" + session.SID
}

// stepsKey - returns the key for the session's step array
func (session *Session) stepsKey() string {
	return session.key() + ":Steps"
}
