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

package puzzle

import (
	"encoding/json"
	"fmt"
	"net/http"
)

/*

Puzzle Creation

*/

// NewHandler is a POST handler that reads a JSON-encoded Summary
// value from the request body calls New on the argument values
// to create a new Puzzle.  The new Puzzle's State is sent as a
// 200 response, and the new puzzle itself is returned to the
// golang caller.  If the return value from New is an error, then
// the error is sent as a 400 response and also returned to the
// caller.
//
// If we can't decode the posted Summary, we send a 400 reponse and
// return the error to the caller.
//
// If we can't encode the response to the client (which should
// never happen), then the client gets an error response and the
// golang caller gets both the puzzle and the encoding Error (as
// a signal that the client didn't get the correct response).
func NewHandler(w http.ResponseWriter, r *http.Request) (*Puzzle, error) {
	dec := json.NewDecoder(r.Body)
	var summary Summary
	e := dec.Decode(&summary)
	if e != nil {
		return nil, writeError(requestDecodingError, ErrorData{e.Error()}, w, r)
	}
	p, e := New(&summary)
	if e != nil {
		err, ok := e.(Error)
		if !ok {
			return nil, writeError(errorFormatError, ErrorData{"NewHandler", e.Error()}, w, r)
		}
		err.Message = err.Error()
		return nil, writeJSON(err, http.StatusBadRequest, w, r)
	}
	return p, p.StateHandler(w, r)
}

/*

Puzzle Download Methods

*/

// SummaryHandler responds with the Puzzle's summary.  If we can't
// encode the response to the client successfully, we give both
// the client and the golang caller an Error response.
func (p *Puzzle) SummaryHandler(w http.ResponseWriter, r *http.Request) error {
	if !p.isValid() {
		return writeError(noPuzzleError, ErrorData{r.URL.Path, "No puzzle"}, w, r)
	}
	return writeJSON(p.summary(), http.StatusOK, w, r)
}

// StateHandler responds with the Puzzle's content.  If we can't
// encode the response to the client successfully, we give both
// the client and the golang caller an Error response.
func (p *Puzzle) StateHandler(w http.ResponseWriter, r *http.Request) error {
	if !p.isValid() {
		return writeError(noPuzzleError, ErrorData{r.URL.Path, "No puzzle"}, w, r)
	}
	return writeJSON(p.state(), http.StatusOK, w, r)
}

// SolutionsHandler responds with the Puzzle's solutions (or the
// Error produced by computing the puzzle's solutions).  If we
// can't encode the response to the client successfully, we give
// both the client and the golang caller an Error response.
func (p *Puzzle) SolutionsHandler(w http.ResponseWriter, r *http.Request) error {
	if !p.isValid() {
		return writeError(noPuzzleError, ErrorData{r.URL.Path, "No puzzle"}, w, r)
	}
	return writeJSON(p.allSolutions(), http.StatusOK, w, r)
}

/*

Puzzle Updates

*/

// AssignHandler is a POST handler that assigns a posted choice
// to a puzzle.  The poster and the caller both get the Content
// object returned from the assignment (or the error).
//
// If we can't decode the posted choice, we send a 400 reponse
// and return the error to the caller.
//
// If we can't encode the response to the client (which should
// never happen), then the client gets an error response and the
// golang caller gets both the update and the encoding Error (as
// a signal that the client didn't get the update).
func (p *Puzzle) AssignHandler(w http.ResponseWriter, r *http.Request) (*Content, error) {
	if !p.isValid() {
		return nil, writeError(noPuzzleError, ErrorData{r.URL.Path, "No puzzle"}, w, r)
	}
	dec := json.NewDecoder(r.Body)
	var choice Choice
	e := dec.Decode(&choice)
	if e != nil {
		return nil, writeError(requestDecodingError, ErrorData{e.Error()}, w, r)
	}
	update, e := p.Assign(choice)
	if e != nil {
		err, ok := e.(Error)
		if !ok {
			return nil, writeError(errorFormatError, ErrorData{"AssignHandler", e.Error()}, w, r)
		}
		err.Message = err.Error()
		return nil, writeJSON(err, http.StatusBadRequest, w, r)
	}
	return update, writeJSON(update, http.StatusOK, w, r)
}

/*

Utilities

*/

type handlerError int

const (
	requestDecodingError handlerError = iota
	responseEncodingError
	noPuzzleError
	errorFormatError
)

// writeError sends back a server error of the given type, sort
// of like http.Error, but it sends the JSON form of an
// appropriate Error.
func writeError(et handlerError, ed ErrorData,
	w http.ResponseWriter, r *http.Request) error {
	var err Error
	var status int
	switch et {
	case requestDecodingError:
		status = http.StatusBadRequest
		err = Error{
			Scope:     RequestScope,
			Structure: AttributeStructure,
			Attribute: DecodeAttribute,
			Condition: GeneralCondition,
			Values:    ed,
		}
	case responseEncodingError:
		status = http.StatusInternalServerError
		err = Error{
			Scope:     InternalScope,
			Structure: AttributeStructure,
			Attribute: EncodeAttribute,
			Condition: GeneralCondition,
			Values:    ed,
		}
	case noPuzzleError:
		status = http.StatusNotFound
		err = Error{
			Scope:     RequestScope,
			Structure: AttributeValueStructure,
			Attribute: URLAttribute,
			Condition: GeneralCondition,
			Values:    ed,
		}
	case errorFormatError:
		status = http.StatusInternalServerError
		err = Error{
			Scope:     InternalScope,
			Structure: AttributeStructure,
			Attribute: LocationAttribute,
			Condition: GeneralCondition,
			Values:    ed,
		}
	default:
		status = http.StatusInternalServerError
		err = Error{
			Scope:     InternalScope,
			Structure: AttributeStructure,
			Attribute: LocationAttribute,
			Condition: GeneralCondition,
			Values: ErrorData{
				"writeError",
				fmt.Sprintf("Unknown handler error type (%v)", et),
			},
		}
	}
	err.Message = err.Error()
	return writeJSON(err, status, w, r)
}

// writeJSON is called by handlers to encode and send the client
// response.  It returns an appropriate error status for the
// handler to return to its caller, as follows:
//
// 1. If writeJSON encounters an encoding error sending the
// response, it will create an Error object describing the
// failure, encode that Error as a 500-series response to the
// client, and return that Error to the handler.
//
// 2. If no encoding error occurs, but the handler is sending an
// Error object as the response to the client, writeJSON will
// return that same Error to the handler.
//
// 3. If no encoding error occurs, and the handler is sending a
// non-Error object as the response to the client, writeJSON will
// return nil to the handler.
func writeJSON(obj interface{}, status int, w http.ResponseWriter, r *http.Request) error {
	err, isErr := obj.(Error)
	bytes, e := json.Marshal(obj)
	if e != nil {
		if isErr && err.Scope == InternalScope && err.Attribute == EncodeAttribute {
			// We just failed to encode an Encoding error.  This
			// should never happen!!  If it did, it almost
			// certainly means that the JSON encoding system is
			// dead, so pseudo-encode the error by hand by
			// returning the Error's summary as a quoted string.
			status = http.StatusInternalServerError // probably was already!
			bytes = []byte(fmt.Sprintf("%q", err.Error()))
		} else {
			// generate, send, and return an encoding error
			return writeError(responseEncodingError, ErrorData{e.Error()}, w, r)
		}
	}
	hs := w.Header()
	hs.Add("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(bytes)
	if isErr {
		return err
	}
	return nil
}
