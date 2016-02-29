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

package puzzle

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

/*

helper puzzle type: gives errors doing json encoding of summary.

*/

type unencodable int

func (u unencodable) MarshalJSON() ([]byte, error) {
	return []byte(`"unencodable"`), fmt.Errorf("unencodable")
}

var badError = Error{Message: "unencodable error", Values: ErrorData{unencodable(0)}}

type badEncoderPuzzle Puzzle

func (b *badEncoderPuzzle) Summary() (*Summary, error) {
	return &Summary{nil, StandardGeometryName, 0, []int{}, nil}, nil
}

func (b *badEncoderPuzzle) State() (*Content, error) {
	return nil, nil
}

func (b *badEncoderPuzzle) Solutions() ([]Solution, error) {
	return nil, nil
}

func (b *badEncoderPuzzle) Assign(choice Choice) (*Content, error) {
	return nil, badError
}

func (b *badEncoderPuzzle) Copy() (*Puzzle, error) {
	return (*Puzzle)(b), nil
}

func newBadEncoder(values []int) (*Puzzle, error) {
	return (*Puzzle)(&badEncoderPuzzle{}), nil
}

func newReallyBadEncoder(values []int) (*Puzzle, error) {
	return nil, badError
}

func init() {
	knownGeometries["badgeometry"] = newBadEncoder
	knownGeometries["reallybadgeometry"] = newReallyBadEncoder
}

/*

GET handlers

*/

func TestPuzzleGetHandlers(t *testing.T) {
	tests := []*Summary{
		&Summary{nil, StandardGeometryName, 4, rotation4Puzzle1PartialAssign1Values, nil},
		&Summary{nil, StandardGeometryName, 4, rotation4Puzzle1Complete1, nil},
		&Summary{nil, StandardGeometryName, 4, empty4PuzzleValues, nil},
		&Summary{nil, StandardGeometryName, 9, oneStarValues, nil},
		&Summary{nil, StandardGeometryName, 9, sixStarValues, nil},
	}
	for i, test := range tests {
		p, e := New(test)
		if e != nil {
			t.Fatalf("test %d: Creation of puzzle failed: %v", i, e)
		}

		handlers := []func(http.ResponseWriter, *http.Request) error{
			p.SummaryHandler,
			p.StateHandler,
			p.SolutionsHandler,
		}
		osummary, isummary := Summary{}, *p.summary()
		ostate, istate := Content{}, *p.state()
		osolns, isolns := []Solution{}, p.allSolutions()
		outputs := []interface{}{&osummary, &ostate, &osolns}
		inputs := []interface{}{&isummary, &istate, &isolns}
		for j, handler := range handlers {
			handlerFunc := func(w http.ResponseWriter, r *http.Request) {
				err := handler(w, r)
				if err != nil {
					t.Errorf("%v failed: %v", handler, err)
				}
			}
			ts := httptest.NewServer(http.HandlerFunc(handlerFunc))
			defer ts.Close()

			r, e := http.Get(ts.URL)
			if e != nil {
				t.Fatalf("test %d: Request error: %v", i, e)
			}
			if r.StatusCode != http.StatusOK {
				t.Errorf("Incorrect status: %q\n", r.Status)
				t.Logf("Headers are: %v\n", r.Header)
			}
			b, e := ioutil.ReadAll(r.Body)
			r.Body.Close()
			if e != nil {
				t.Logf("Response body: %s\n", b)
				t.Fatalf("test %d: Read error on puzzle response body: %v", i, e)
			}

			e = json.Unmarshal(b, outputs[j])
			if e != nil {
				t.Fatalf("test %d: Unmarshal failed: %v", i, e)
			}
			if !reflect.DeepEqual(outputs[j], inputs[j]) {
				t.Errorf("test %d: Received %+v, expected %+v:", i, outputs[j], inputs[j])
			}
		}
	}
}

func TestGetHandlerErrors(t *testing.T) {
	var p *Puzzle

	handlers := []func(http.ResponseWriter, *http.Request) error{
		p.SummaryHandler,
		p.StateHandler,
		p.SolutionsHandler,
	}
	for _, handler := range handlers {
		handlerFunc := func(w http.ResponseWriter, r *http.Request) {
			err := handler(w, r)
			if err == nil {
				t.Errorf("%v didn't fail", handler)
			}
		}
		ts := httptest.NewServer(http.HandlerFunc(handlerFunc))
		defer ts.Close()

		r, e := http.Get(ts.URL)
		if e != nil {
			t.Fatalf("Request error: %v", e)
		}
		r.Body.Close()
		if r.StatusCode != http.StatusNotFound {
			t.Errorf("Response status was %d (expected %d)", r.StatusCode, http.StatusNotFound)
		}
	}
}

/*

POST handlers

*/

func TestNewHandler(t *testing.T) {
	testcases := []*Summary{
		&Summary{nil, StandardGeometryName, 4, empty4PuzzleValues, nil},
		&Summary{nil, StandardGeometryName, 4, rotation4Puzzle1PartialAssign1Values, nil},
		&Summary{nil, StandardGeometryName, 4, rotation4Puzzle1Complete1, nil},
	}
	for i, tc := range testcases {
		pe, err := New(tc)
		if err != nil {
			t.Fatalf("case %d: Failed to create puzzle: %v", i, err)
		}
		pesum, pestate := pe.summary(), pe.state()

		bytes, err := json.Marshal(tc)
		if err != nil {
			t.Fatalf("case %d: Failed to encode summary: %v", i, err)
		}

		handlerFunc := func(w http.ResponseWriter, r *http.Request) {
			p, e := NewHandler(w, r)
			if e != nil {
				t.Fatalf("Failed to create puzzle in handler: %v", e)
			}
			if psum := p.summary(); !reflect.DeepEqual(psum, pesum) {
				t.Errorf("test %d: Created puzzle has summary %+v, expected %+v", i, *psum, *pesum)
			}
			if pstate := p.state(); !reflect.DeepEqual(pstate, pestate) {
				t.Errorf("test %d: Created puzzle has state %+v, expected %+v", i, *pstate, *pestate)
			}
		}
		ts := httptest.NewServer(http.HandlerFunc(handlerFunc))
		defer ts.Close()

		r, e := http.Post(ts.URL, "application/json", strings.NewReader(string(bytes)))
		if e != nil {
			t.Logf("case %d body: %s\n", i, bytes)
			t.Fatalf("case %d: Request error: %v", i, e)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("case %d: Status was %v, expected %v", i, r.StatusCode, http.StatusOK)
			t.Logf("case %d headers: %v\n", i, r.Header)
		}
		b, e := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if e != nil {
			t.Logf("test %d body: %s\n", i, b)
			t.Fatalf("test %d: Read error on body: %v", i, e)
		}

		var state *Content
		e = json.Unmarshal(b, &state)
		if e != nil {
			t.Fatalf("test %d: Unmarshal failed: %v", i, e)
		}
		if !reflect.DeepEqual(state, pestate) {
			t.Errorf("test %d: Summary was %+v, expected %+v:", i, *state, *pestate)
		}
	}
}

type testNewHandlerErrorTestcase struct {
	name      string
	data      string
	attribute ErrorAttribute
}

func TestNewHandlerErrors(t *testing.T) {
	testcases := []testNewHandlerErrorTestcase{
		{"bad input", `"string not summary"`, DecodeAttribute},
		{"unknown geometry", `{"geometry":"nope","sidelen":4}`, GeometryAttribute},
		{"values incompatible", `{"geometry":"square","sidelen":4,"values":[1, 2, 3]}`, PuzzleSizeAttribute},
	}

	for _, tc := range testcases {
		handlerFunc := func(w http.ResponseWriter, r *http.Request) {
			p, e := NewHandler(w, r)
			if e == nil {
				t.Errorf("Test %s: Successfully created puzzle: %v", tc.name, p.summary())
			}
		}
		ts := httptest.NewServer(http.HandlerFunc(handlerFunc))
		defer ts.Close()

		r, e := http.Post(ts.URL, "application/json", strings.NewReader(tc.data))
		if e != nil {
			t.Fatalf("Request error: %v", e)
		}
		if r.StatusCode != http.StatusBadRequest {
			t.Errorf("Test %s: HTTP Status was %v, expected %v",
				tc.name, r.StatusCode, http.StatusBadRequest)
			t.Logf("Test %s headers: %v\n", tc.name, r.Header)
		}
		b, e := ioutil.ReadAll(r.Body)
		r.Body.Close()
		var err Error
		e = json.Unmarshal(b, &err)
		if e != nil {
			t.Errorf("Test %s: response decode error: %v", tc.name, e)
		}
		if err.Attribute != tc.attribute {
			t.Errorf("Test %s: Attribute was %v, expected %v",
				tc.name, err.Attribute, tc.attribute)
			t.Logf("Test %s Error: %+v", tc.name, err)
		}
	}
}

func TestAssignHandler(t *testing.T) {
	choices := []Choice{{13, 2}, {10, 4}, {15, 4}}
	p1, err := New(&Summary{Geometry: StandardGeometryName, SideLength: 4, Values: rotation4Puzzle1PartialValues})
	if err != nil {
		t.Fatalf("Failed to create initial puzzle1: %v", err)
	}
	p2, err := New(&Summary{Geometry: StandardGeometryName, SideLength: 4, Values: rotation4Puzzle1PartialValues})
	if err != nil {
		t.Fatalf("Failed to create initial puzzle2: %v", err)
	}

	for i, choice := range choices {
		bytes, err := json.Marshal(choice)
		if err != nil {
			t.Fatalf("Case %d: Failed to encode choice: %v", i, err)
		}
		up2, err := p2.Assign(choice)
		if err != nil {
			t.Fatalf("Case %d: Failed to assign choice to p1: %v", i, err)
		}

		handler := func(w http.ResponseWriter, r *http.Request) {
			up1, err := p1.AssignHandler(w, r)
			if err != nil {
				t.Fatalf("Case %d: Failed to assign choice to p1: %v", i, err)
			}
			if !reflect.DeepEqual(up1, up2) {
				t.Errorf("Case %d: Result of assign to p1 (%+v) differs from p2 (%+v)",
					i, up1, up2)
			}
		}
		ts := httptest.NewServer(http.HandlerFunc(handler))
		defer ts.Close()

		r, e := http.Post(ts.URL, "application/json", strings.NewReader(string(bytes)))
		if e != nil {
			t.Logf("case %d POST body: %s", i, bytes)
			t.Fatalf("case %d: Request error: %v", i, e)
		}
		if r.StatusCode != http.StatusOK {
			t.Errorf("case %d: Status was %v, expected %v", i, r.StatusCode, http.StatusOK)
			t.Logf("case %d headers: %v\n", i, r.Header)
		}
		b, e := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if e != nil {
			t.Logf("test %d response body: %s\n", i, b)
			t.Fatalf("test %d: Read error on summary: %v", i, e)
		}

		var update *Content
		e = json.Unmarshal(b, &update)
		if e != nil {
			t.Fatalf("test %d: Unmarshal failed: %v", i, e)
		}
		if !reflect.DeepEqual(update, up2) {
			t.Errorf("test %d: Content was %+v, expected %+v:", i, update, up2)
		}
	}
}

func TestAssignHandlerErrors(t *testing.T) {
	p, err := New(&Summary{Geometry: StandardGeometryName, SideLength: 4, Values: rotation4Puzzle1PartialValues})
	if err != nil {
		t.Fatalf("Failed to create initial puzzle: %v", err)
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		_, err := p.AssignHandler(w, r)
		if err == nil {
			t.Errorf("Successful assignment!")
		}
	}
	ts := httptest.NewServer(http.HandlerFunc(handler))
	defer ts.Close()

	bytes, err := json.Marshal([]int{1, 2, 3})
	if err != nil {
		t.Fatalf("Failed to encode []int{1, 2, 3}: %v", err)
	}
	r, e := http.Post(ts.URL, "application/json", strings.NewReader(string(bytes)))
	if e != nil {
		t.Logf("Post body: %s\n", bytes)
		t.Fatalf("Request error: %v", e)
	}
	if r.StatusCode != http.StatusBadRequest {
		t.Errorf("Status was %v, expected %v", r.StatusCode, http.StatusBadRequest)
		t.Logf("Headers are: %v\n", r.Header)
	}
	b, e := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if e != nil {
		t.Logf("Response body: %s\n", b)
		t.Fatalf("Read error on result: %v", e)
	}

	bytes, err = json.Marshal(Choice{14, 2})
	if err != nil {
		t.Fatalf("Failed to encode Choice{14, 2}: %v", err)
	}
	r, e = http.Post(ts.URL, "application/json", strings.NewReader(string(bytes)))
	if e != nil {
		t.Logf("Post body: %s\n", bytes)
		t.Fatalf("Request error: %v", e)
	}
	if r.StatusCode != http.StatusBadRequest {
		t.Errorf("Status was %v, expected %v", r.StatusCode, http.StatusBadRequest)
		t.Logf("Headers are: %v\n", r.Header)
	}
	b, e = ioutil.ReadAll(r.Body)
	r.Body.Close()
	if e != nil {
		t.Logf("Response body: %s\n", b)
		t.Fatalf("Read error on result: %v", e)
	}
}
