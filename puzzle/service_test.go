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

helper puzzle type: gives errors doing json encoding of state.

*/

type unencodable int

func (u unencodable) MarshalJSON() ([]byte, error) {
	return []byte(`"unencodable"`), fmt.Errorf("unencodable")
}

var badError = Error{Message: "unencodable error", Values: ErrorData{unencodable(0)}}

type badEncoderPuzzle string

func (b badEncoderPuzzle) State() State {
	return State{SudokuGeometryCode, 0, []int{}, nil}
}

func (b badEncoderPuzzle) Squares() []Square {
	return nil
}

func (b badEncoderPuzzle) Solutions() []Solution {
	return nil
}

func (b badEncoderPuzzle) Assign(choice Choice) (Update, error) {
	return Update{}, badError
}

func (b badEncoderPuzzle) Copy() Puzzle {
	return b
}

func newBadEncoder(values []int) (Puzzle, error) {
	return badEncoderPuzzle(fmt.Sprint(values)), nil
}

var badGeometry = GeometryDescriptor{[]string{"bad"}, 254, newBadEncoder}

func newReallyBadEncoder(values []int) (Puzzle, error) {
	return nil, badError
}

var reallyBadGeometry = GeometryDescriptor{[]string{"really bad"}, 255, newReallyBadEncoder}

func init() {
	if e := RegisterGeometry(&badGeometry); e != nil {
		panic(fmt.Errorf("Couldn't register bad Geometry: %v", e))
	}
	if e := RegisterGeometry(&reallyBadGeometry); e != nil {
		panic(fmt.Errorf("Couldn't register really bad Geometry: %v", e))
	}
}

/*

GET handlers

*/

func TestPuzzleGetHandlers(t *testing.T) {
	tests := [][]int{
		rotation4Puzzle1PartialAssign1Values,
		rotation4Puzzle1Complete1,
		empty4PuzzleValues,
		oneStarValues,
		sixStarValues,
	}
	for i, vs := range tests {
		p, e := New(append([]int{SudokuGeometryCode}, vs...))
		if e != nil {
			t.Fatalf("test %d: Creation of puzzle failed: %v", i, e)
		}

		handlers := []func(Puzzle, http.ResponseWriter, *http.Request) error{
			StateHandler,
			SquaresHandler,
			SolutionsHandler,
		}
		ostate, istate := State{}, p.State()
		osquares, isquares := []Square{}, p.Squares()
		osolns, isolns := []Solution{}, p.Solutions()
		outputs := []interface{}{&ostate, &osquares, &osolns}
		inputs := []interface{}{&istate, &isquares, &isolns}
		for j, handler := range handlers {
			handlerFunc := func(w http.ResponseWriter, r *http.Request) {
				err := handler(p, w, r)
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
			t.Logf("%q\n", r.Status)
			t.Logf("%v\n", r.Header)
			b, e := ioutil.ReadAll(r.Body)
			r.Body.Close()
			if e != nil {
				t.Fatalf("test %d: Read error on puzzle response body: %v", i, e)
			}
			t.Logf("%s\n", b)

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
	var p Puzzle

	handlers := []func(Puzzle, http.ResponseWriter, *http.Request) error{
		StateHandler,
		SquaresHandler,
		SolutionsHandler,
	}
	for _, handler := range handlers {
		handlerFunc := func(w http.ResponseWriter, r *http.Request) {
			err := handler(p, w, r)
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
		t.Logf("%q\n", r.Status)
		r.Body.Close()
		if r.StatusCode != http.StatusNotFound {
			t.Errorf("Response status was %d (expected %d)", r.StatusCode, http.StatusNotFound)
		}
	}
}

/*

POST handlers

*/

type newHandlerTestcase struct {
	geometry int
	values   []int
}

func TestNewHandler(t *testing.T) {
	testcases := []newHandlerTestcase{
		{SudokuGeometryCode, empty4PuzzleValues},
		{SudokuGeometryCode, rotation4Puzzle1PartialAssign1Values},
		{SudokuGeometryCode, rotation4Puzzle1Complete1},
	}
	for i, tc := range testcases {
		codedValues := append([]int{tc.geometry}, tc.values...)
		pe, err := New(codedValues)
		if err != nil {
			t.Fatalf("case %d: Failed to create puzzle: %v", i, err)
		}

		bytes, err := json.Marshal(codedValues)
		if err != nil {
			t.Fatalf("case %d: Failed to encode geometry + values: %v", i, err)
		}

		handlerFunc := func(w http.ResponseWriter, r *http.Request) {
			p, e := NewHandler(w, r)
			if e != nil {
				t.Fatalf("Failed to create puzzle in handler: %v", e)
			}
			if !reflect.DeepEqual(p.State(), pe.State()) {
				t.Errorf("Created puzzle has state %v, expected %v", p.State(), pe.State())
			}
			ps, pes := p.Squares(), pe.Squares()
			if !reflect.DeepEqual(ps, pes) {
				t.Errorf("test %d: Unexpected squares:", i)
				for i := range ps {
					if !reflect.DeepEqual(ps[i], pes[i]) {
						t.Errorf("Square %d: is %+v, expected %+v",
							ps[i].Index, ps[i], pes[i])
					}
				}
			}
		}
		ts := httptest.NewServer(http.HandlerFunc(handlerFunc))
		defer ts.Close()

		t.Logf("%s\n", bytes)
		r, e := http.Post(ts.URL, "application/json", strings.NewReader(string(bytes)))
		if e != nil {
			t.Fatalf("case %d: Request error: %v", i, e)
		}
		t.Logf("%q\n", r.Status)
		t.Logf("%v\n", r.Header)
		if r.StatusCode != http.StatusOK {
			t.Errorf("case %d: Status was %v, expected %v", i, r.StatusCode, http.StatusOK)
		}
		b, e := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if e != nil {
			t.Fatalf("test %d: Read error on state: %v", i, e)
		}
		t.Logf("%s\n", b)

		var state State
		e = json.Unmarshal(b, &state)
		if e != nil {
			t.Fatalf("test %d: Unmarshal failed: %v", i, e)
		}
		if !reflect.DeepEqual(state, pe.State()) {
			t.Errorf("test %d: State was %+v, expected %+v:", i, state, pe.State())
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
		{"bad input", `"string not []int"`, DecodeAttribute},
		{"unknown geometry", "[-1, 1, 2, 3]", GeometryAttribute},
		{"values incompatible", "[0, 1, 2, 3]", PuzzleSizeAttribute},
	}

	for _, tc := range testcases {
		handlerFunc := func(w http.ResponseWriter, r *http.Request) {
			p, e := NewHandler(w, r)
			if e == nil {
				t.Errorf("Test %s: Successfully created puzzle: %v", tc.name, p.State())
			}
		}
		ts := httptest.NewServer(http.HandlerFunc(handlerFunc))
		defer ts.Close()

		r, e := http.Post(ts.URL, "application/json", strings.NewReader(tc.data))
		if e != nil {
			t.Fatalf("Request error: %v", e)
		}
		t.Logf("%q\n", r.Status)
		t.Logf("%v\n", r.Header)
		if r.StatusCode != http.StatusBadRequest {
			t.Errorf("Test %s: HTTP Status was %v, expected %v",
				tc.name, r.StatusCode, http.StatusBadRequest)
		}
		b, e := ioutil.ReadAll(r.Body)
		r.Body.Close()
		var err Error
		e = json.Unmarshal(b, &err)
		if e != nil {
			t.Errorf("Test %s: response decode error: %v", tc.name, e)
		}
		t.Logf("%v", err)
		if err.Attribute != tc.attribute {
			t.Errorf("Test %s: Attribute was %v, expected %v",
				tc.name, err.Attribute, tc.attribute)
		}
	}
}

func TestAssignHandler(t *testing.T) {
	choices := []Choice{{13, 2}, {10, 4}, {15, 4}}
	p1, err := New(append([]int{SudokuGeometryCode}, rotation4Puzzle1PartialValues...))
	if err != nil {
		t.Fatalf("Failed to create initial puzzle1: %v", err)
	}
	p2, err := New(append([]int{SudokuGeometryCode}, rotation4Puzzle1PartialValues...))
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
			up1, err := AssignHandler(p1, w, r)
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

		t.Logf("%s\n", bytes)
		r, e := http.Post(ts.URL, "application/json", strings.NewReader(string(bytes)))
		if e != nil {
			t.Fatalf("case %d: Request error: %v", i, e)
		}
		t.Logf("%q\n", r.Status)
		t.Logf("%v\n", r.Header)
		if r.StatusCode != http.StatusOK {
			t.Errorf("case %d: Status was %v, expected %v", i, r.StatusCode, http.StatusOK)
		}
		b, e := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if e != nil {
			t.Fatalf("test %d: Read error on state: %v", i, e)
		}
		t.Logf("%s\n", b)

		var update Update
		e = json.Unmarshal(b, &update)
		if e != nil {
			t.Fatalf("test %d: Unmarshal failed: %v", i, e)
		}
		if !reflect.DeepEqual(update, up2) {
			t.Errorf("test %d: Update was %+v, expected %+v:", i, update, up2)
		}
	}
}

func TestAssignHandlerErrors(t *testing.T) {
	p, err := New(append([]int{SudokuGeometryCode}, rotation4Puzzle1PartialValues...))
	if err != nil {
		t.Fatalf("Failed to create initial puzzle: %v", err)
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		_, err := AssignHandler(p, w, r)
		if err == nil {
			t.Errorf("Successful assignment!")
		}
		t.Logf("Server err: %v", err)
	}
	ts := httptest.NewServer(http.HandlerFunc(handler))
	defer ts.Close()

	bytes, err := json.Marshal([]int{1, 2, 3})
	if err != nil {
		t.Fatalf("Failed to encode []int{1, 2, 3}: %v", err)
	}
	t.Logf("%s\n", bytes)
	r, e := http.Post(ts.URL, "application/json", strings.NewReader(string(bytes)))
	if e != nil {
		t.Fatalf("Request error: %v", e)
	}
	t.Logf("%q\n", r.Status)
	t.Logf("%v\n", r.Header)
	if r.StatusCode != http.StatusBadRequest {
		t.Errorf("Status was %v, expected %v", r.StatusCode, http.StatusBadRequest)
	}
	b, e := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if e != nil {
		t.Fatalf("Read error on result: %v", e)
	}
	t.Logf("%s\n", b)

	bytes, err = json.Marshal(Choice{14, 2})
	if err != nil {
		t.Fatalf("Failed to encode Choice{14, 2}: %v", err)
	}
	t.Logf("%s\n", bytes)
	r, e = http.Post(ts.URL, "application/json", strings.NewReader(string(bytes)))
	if e != nil {
		t.Fatalf("Request error: %v", e)
	}
	t.Logf("%q\n", r.Status)
	t.Logf("%v\n", r.Header)
	if r.StatusCode != http.StatusBadRequest {
		t.Errorf("Status was %v, expected %v", r.StatusCode, http.StatusBadRequest)
	}
	b, e = ioutil.ReadAll(r.Body)
	r.Body.Close()
	if e != nil {
		t.Fatalf("Read error on result: %v", e)
	}
	t.Logf("%s\n", b)
}
