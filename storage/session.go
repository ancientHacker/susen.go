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

// This package provides the persistence layer for the objects
// (sessions, puzzles, etc.) used in the Sūsen game layer.
//
// Unlike most support packages, this package does not catch
// runtime panics at the public entry layer and repackage them as
// errors.  That's because the game layer, as a user-facing
// service, has to do its own graceful handling of runtime
// errors, so to do it here would be redundant.
package storage

import (
	"encoding/json"
	"fmt"
	"github.com/ancientHacker/susen.go/Godeps/_workspace/src/github.com/garyburd/redigo/redis"
	"github.com/ancientHacker/susen.go/Godeps/_workspace/src/github.com/jackc/pgx"
	"github.com/ancientHacker/susen.go/dbprep"
	"github.com/ancientHacker/susen.go/puzzle"
	"strings"
	"time"
)

/*

Session interface

*/

// A Session holds a stored session's complete history, including
// all the puzzles associated with the session and any past
// choices made while working those puzzles.  Because sessions
// are actually views on the storage system, clients must look
// them up, not create them.  But if the client supplies a
// (non-empty) ID that's not in the system, a session is created
// for that ID.
type Session struct {
	Puzzle  *puzzle.Puzzle  // the active puzzle
	Info    *PuzzleInfo     // info about the active puzzle
	sid     string          // ID of this session
	info    *sessionInfo    // top level info about the session
	entries []*sessionEntry // all the puzzle entries in this session
	active  int             // the index of the active entry
}

// LoadSession: find a session given a session ID.  If there
// isn't already a session with that ID, a standard "sample"
// session is cloned and returned.
//
// Sessions cannot have empty IDs: providing an empty sessionId
// will panic.
func LoadSession(sessionId string) (s *Session) {
	if sessionId == "" {
		panic(fmt.Errorf("Session IDs cannot be null"))
	}
	s = &Session{sid: sessionId, active: -1}
	s.cacheLoadSession()
	if s.info == nil {
		s.databaseLoadSession()
		if s.info == nil {
			s.initializeFromSample()
			s.databaseInsertSession()
		}
		s.cacheInsertSession()
	}
	s.SelectPuzzle(s.info.ActivePID)
	return
}

// GetInactivePuzzles gets info about all the session puzzles
// other than the active one.
func (s *Session) GetInactivePuzzles() []*PuzzleInfo {
	infos := make([]*PuzzleInfo, 0, len(s.entries)-1)
	for i := range s.entries {
		if i != s.active {
			infos = append(infos, s.makePuzzleInfo(i))
		}
	}
	return infos
}

// SelectPuzzle: activate a specific puzzle for the session.  The
// puzzle is activated in the same state it was in when last
// active.  Activating the currently active puzzle is a no-op.
// Activating a puzzle not in the session selects the first
// session entry as the active entry.
//
// This function relies on some invariants: First, that puzzle
// signatures are uppercase; this is maintained by the puzzle
// module.  Second, that puzzle names are lowercase; this is
// maintained by the database routines in this module and the
// dbprep module (for initial data).
func (s *Session) SelectPuzzle(pid string) {
	// canonicalize the pid
	upid := strings.ToUpper(pid)
	lpid := strings.ToLower(pid)
	// find the puzzle being selected
	next := -1
	for i, se := range s.entries {
		if se.PuzzleId == upid {
			next = i
			break
		}
		if se.PuzzleName == lpid {
			next = i
			break
		}
	}
	if next == -1 {
		panic(fmt.Errorf("No such puzzle in this session: %s", pid))
	}
	if next == s.active {
		return
	}
	// activate the session entry
	s.active = next
	se := s.entries[next]
	s.info.ActivePID, s.info.Updated = se.PuzzleId, time.Now()
	s.cacheSaveInfo()
	s.databaseUpdateInfo()
	se.LastView = time.Now()
	s.cacheUpdateEntry(s.active)
	s.databaseUpdateEntry(s.active)
	// load the puzzle
	s.loadActivePuzzle()
}

/*

Session loading and saving

*/

// cacheLoadSession: find any existing session in the cache
// and load the Session object from it.  Returns any error
// encountered, and returns a non-nil session if one is found.
func (s *Session) cacheLoadSession() {
	s.cacheLoadInfo()
	if s.info == nil {
		return
	}
	s.cacheLoadEntries()
	if len(s.entries) == 0 {
		s.info = nil
	}
}

// databaseLoadSession: load the session from the database.
func (s *Session) databaseLoadSession() {
	s.databaseLoadInfo()
	if s.info == nil {
		return
	}
	s.databaseLoadEntries()
	if len(s.entries) == 0 {
		s.info = nil
	}
}

// cacheInsertSession: insert a new session into the cache.
func (s *Session) cacheInsertSession() {
	s.cacheSaveInfo()
	s.cacheSaveEntries()
}

// databaseInsertSession: insert a new session into the database.
func (s *Session) databaseInsertSession() {
	s.databaseInsertInfo()
	s.databaseInsertEntries()
}

/*

session info

*/

// A sessionInfo holds the basic data stored about each session.
// It is JSON serializable so it can go into the cache as well as
// the database.
type sessionInfo struct {
	Created   time.Time // when the session was created
	Updated   time.Time // when the session data (not puzzles) last changed
	ActivePID string    // the PuzzleId of the active puzzle
}

// key: returns the base of all session keys.
func (s *Session) key() string {
	return "SID:" + s.sid
}

// infoKey: returns the cache session info key.
func (s *Session) infoKey() string {
	return s.key() + ":Info"
}

// cacheLoadInfo: load the session info from the cache.
func (s *Session) cacheLoadInfo() {
	var bytes []byte
	body := func(tx redis.Conn) (err error) {
		bytes, err = redis.Bytes(tx.Do("GET", s.infoKey()))
		if err == redis.ErrNil {
			return nil
		}
		if err != nil {
			return fmt.Errorf("Cache error on lookup of info for session %q: %v", s.sid, err)
		}
		return
	}
	rdExecute(body)
	if len(bytes) == 0 {
		return
	}
	s.unmarshalInfo(bytes)
}

// databaseLoadInfo gets the session info from the DB, if there
// is such a session.
func (s *Session) databaseLoadInfo() {
	si := sessionInfo{}
	body := func(tx *pgx.Tx) (err error) {
		row := tx.QueryRow(
			"SELECT created, updated, active FROM sessions "+
				"WHERE sessionId = $1", s.sid)
		err = row.Scan(&si.Created, &si.Updated, &si.ActivePID)
		if err == pgx.ErrNoRows {
			return nil
		}
		if err != nil {
			return fmt.Errorf("Database failure loading info for session %q: %v", s.sid, err)
		}
		return
	}
	pgExecute(body)
	if !si.Created.IsZero() && !si.Updated.IsZero() {
		s.info = &si
	}
}

// cacheSaveInfo: save the current session info to the cache.
// This will insert a new info or replace any existing info for
// this session.
func (s *Session) cacheSaveInfo() {
	bytes := s.marshalInfo()
	body := func(tx redis.Conn) (err error) {
		_, err = tx.Do("SET", s.infoKey(), bytes)
		if err != nil {
			return fmt.Errorf("Cache error saving info for session %q: %v", s.sid, err)
		}
		return
	}
	rdExecute(body)
}

// databaseInsertInfo: add a new session to the database; this will
// fail if the session already exists.
func (s *Session) databaseInsertInfo() {
	body := func(tx *pgx.Tx) error {
		_, err := tx.Exec(
			"INSERT INTO sessions (sessionId, created, updated, active) "+
				"VALUES ($1, $2, $3, $4)",
			s.sid, s.info.Created, s.info.Updated, s.info.ActivePID)
		if err != nil {
			return fmt.Errorf("Database failure inserting session %q: %v", s.sid, err)
		}
		return err
	}
	pgExecute(body)
}

// databaseUpdateInfo: update an existing database session.
func (s *Session) databaseUpdateInfo() {
	body := func(tx *pgx.Tx) error {
		_, err := tx.Exec(
			"UPDATE sessions SET (created, updated, active) "+
				"= ($2, $3, $4) WHERE sessionId = $1",
			s.sid, s.info.Created, s.info.Updated, s.info.ActivePID)
		return err
	}
	pgExecute(body)
}

// marshalInfo: return JSON serialization of session info.
func (s *Session) marshalInfo() []byte {
	bytes, err := json.Marshal(s.info)
	if err != nil {
		panic(fmt.Errorf("Failed to marshal info for session %q: %v", s.sid, err))
	}
	return bytes
}

// unmarshalInfo: set session info from serialization.
func (s *Session) unmarshalInfo(bytes []byte) {
	var si *sessionInfo
	err := json.Unmarshal(bytes, &si)
	if err != nil {
		panic(fmt.Errorf("Failed to unmarshal info for session %q: %v", s.sid, err))
	}
	s.info = si
}

/*

session entries

*/

// sessionEntry is JSON serializable so it can go into the cache
// as well as the database
type sessionEntry struct {
	PuzzleId   string    // signature of the puzzle being worked
	PuzzleName string    // name of the puzzle being worked
	Choices    []int32   // flattened list of choices made for the active puzzle
	LastView   time.Time // when the puzzle was last viewed
}

// entryKey: this cache key holds a Redis array of serialized
// session entries.
func (s *Session) entryKey() string {
	return s.key() + ":Entries"
}

// cacheLoadEntry: load the specified session entry from the
// cache.  This will fail if the entry doesn't already exist.
func (s *Session) cacheLoadEntry(index int) {
	var bytes []byte
	body := func(tx redis.Conn) (err error) {
		bytes, err = redis.Bytes(tx.Do("LINDEX", s.entryKey(), index))
		if err == redis.ErrNil {
			return fmt.Errorf("No entry %d for session %q", index, s.sid)
		}
		if err != nil {
			return fmt.Errorf("Cache error on lookup of entry %d for session %q: %v",
				index, s.sid, err)
		}
		return
	}
	rdExecute(body)
	s.unmarshalEntry(index, bytes)
}

// cacheLoadEntries: load all the session entries from the cache,
// replacing any existing entries.
func (s *Session) cacheLoadEntries() {
	s.entries = nil
	var bytesSlice [][]byte
	body := func(tx redis.Conn) (err error) {
		bytesSlice, err = redis.ByteSlices(tx.Do("LRANGE", s.key(), 0, -1))
		if err == redis.ErrNil {
			return nil
		}
		if err != nil {
			return fmt.Errorf("Cache error on lookup of entries for session %q: %v", s.sid, err)
		}
		return
	}
	rdExecute(body)
	if len(bytesSlice) == 0 {
		return
	}

	s.entries = make([]*sessionEntry, len(bytesSlice))
	// don't leave s.entries is an incomplete state if the
	// unmarshaling fails unexpectedly
	defer func() {
		if e := recover(); e != nil {
			s.entries = nil
			panic(e)
		}
	}()
	for i, bytes := range bytesSlice {
		s.unmarshalEntry(i, bytes)
	}
}

// databaseLoadEntries: load all the session entries from the
// database, replacing any existing entries.
func (s *Session) databaseLoadEntries() {
	s.entries = nil
	body := func(tx *pgx.Tx) error {
		rows, err := tx.Query(
			"SELECT puzzleId, puzzleName, lastView, choicePairs FROM sessionEntries "+
				"WHERE sessionID = $1", s.sid)
		if err == pgx.ErrNoRows {
			return nil
		}
		if err != nil {
			return fmt.Errorf("Failed to fetch entries for session %q: %v", s.sid, err)
		}
		for i := 0; rows.Next(); i++ {
			se := sessionEntry{}
			err := rows.Scan(&se.PuzzleId, &se.PuzzleName, &se.LastView, &se.Choices)
			if err != nil {
				return fmt.Errorf("Failure loading entry %d for session %v: %v", i, s.sid, err)
			}
			s.entries = append(s.entries, &se)
		}
		return nil
	}
	pgExecute(body)
}

// cacheUpdateEntry: update a session entry already in the cache.
// This will fail if the entry isn't already in the cache.
func (s *Session) cacheUpdateEntry(index int) {
	bytes := s.marshalEntry(index)
	body := func(tx redis.Conn) (err error) {
		_, err = tx.Do("LSET", s.entryKey(), index, bytes)
		if err != nil {
			return fmt.Errorf("Cache error saving entry %d for session %q: %v",
				index, s.sid, err)
		}
		return
	}
	rdExecute(body)
}

// cacheSaveEntries: save all the entries for this session into
// the cache.  This will replace any existing entries.
func (s *Session) cacheSaveEntries() {
	bytesSlice := make([][]byte, len(s.entries))
	for i := range s.entries {
		bytesSlice[i] = s.marshalEntry(i)
	}
	body := func(tx redis.Conn) (err error) {
		// remove any existing entries
		tx.Send("DEL", s.entryKey())
		// add the new ones
		if len(bytesSlice) > 0 {
			_, err = tx.Do("RPUSH", redis.Args{}.Add(s.entryKey()).AddFlat(bytesSlice)...)
			if err != nil {
				err = fmt.Errorf("Failure saving entries for session %q: %v", s.sid, err)
			}
		}
		return
	}
	rdExecute(body)
}

// databaseUpdateEntry: update the session entry in the database.
func (s *Session) databaseUpdateEntry(index int) {
	se := s.entries[index]
	body := func(tx *pgx.Tx) error {
		_, err := tx.Exec(
			"UPDATE sessionEntries SET (puzzleName, choicePairs, lastView) "+
				"= ($1, $2, $3) WHERE sessionId = $4 and puzzleId = $5",
			se.PuzzleName, se.Choices, se.LastView, s.sid, se.PuzzleId)
		return err
	}
	pgExecute(body)
}

// databaseInsertEntries: insert all the entries for this session
// into the database.  This will fail if any of the entries
// already exist.
func (s *Session) databaseInsertEntries() {
	body := func(tx *pgx.Tx) error {
		for i, se := range s.entries {
			_, err := tx.Exec(
				"INSERT INTO sessionEntries "+
					"(sessionId, puzzleId, puzzleName, choicePairs, lastView) "+
					"VALUES ($1, $2, $3, $4, $5);",
				s.sid, se.PuzzleId, se.PuzzleName, se.Choices, se.LastView)
			if err != nil {
				return fmt.Errorf("Database error saving entry %d for session %q: %v",
					i, s.sid, err)
			}
		}
		return nil
	}
	pgExecute(body)
}

// marshalEntry: return JSON serialization of session entry.
func (s *Session) marshalEntry(index int) []byte {
	bytes, err := json.Marshal(s.entries[index])
	if err != nil {
		panic(fmt.Errorf("Failed to marshal entry %d for session %q: %v", index, s.sid, err))
	}
	return bytes
}

// unmarshalEntry: set session entry from serialization.
func (s *Session) unmarshalEntry(index int, bytes []byte) {
	var se *sessionEntry
	err := json.Unmarshal(bytes, &se)
	if err != nil {
		panic(fmt.Errorf("Failed to unmarshal entry %d for session %q: %v", index, s.sid, err))
	}
	s.entries[index] = se
}

/*

Sample session

All new sessions are initialized from this sample session.

*/

var sampleSession = &Session{sid: dbprep.SampleSessionName, active: -1}

// loadSampleSession loads the sample session from the database
// so it will stay in memory for use in initializing new
// sessions.  We don't bother saving it to the cache because it
// only gets loaded once per run, not per request.
func loadSampleSession() *Session {
	if sampleSession.info == nil {
		sampleSession.databaseLoadSession()
		if sampleSession.info == nil {
			panic(fmt.Errorf("Sample session not found in database"))
		}
	}
	return sampleSession
}

// initializeFromSample initializes a new session from the sample
// session so it has some associated puzzles.
func (s *Session) initializeFromSample() {
	ss := loadSampleSession()
	now := time.Now()
	s.info = &sessionInfo{Created: now, Updated: now, ActivePID: ss.info.ActivePID}
	s.entries = make([]*sessionEntry, len(ss.entries))
	for i := range s.entries {
		s.entries[i] = &sessionEntry{
			PuzzleId:   ss.entries[i].PuzzleId,
			PuzzleName: ss.entries[i].PuzzleName,
			Choices:    append([]int32(nil), ss.entries[i].Choices...),
			LastView:   ss.entries[i].LastView,
		}
	}
}
