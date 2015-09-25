package puzzle

import (
	"fmt"
	"reflect"
	"testing"
)

/*

Test the error cases for New.  The non-error cases should be
tested by the various registered geometry implementations.

*/

func TestNewErrorCases(t *testing.T) {
	_, e := New([]int{})
	err, ok := e.(Error)
	if !ok || err.Condition != EmptyArgumentCondition {
		t.Errorf("Wrong error on nil input: %v", e)
	}
	_, e = New([]int{-1, 0, 0, 0, 0})
	err, ok = e.(Error)
	if !ok || err.Condition != UnknownGeometryCondition {
		t.Errorf("Wrong error on geometry -1: %v", e)
	}

	// restore known geometries after test
	defer func(gd []*GeometryDescriptor) {
		knownGeometries = gd
	}(knownGeometries)
	knownGeometries = []*GeometryDescriptor{
		&GeometryDescriptor{
			Names: []string{"test"},
			Code:  0,
			New:   func(_ []int) (Puzzle, error) { return nil, fmt.Errorf("test error") },
		},
	}
	_, e = New([]int{0, 0, 0, 0, 0})
	err, ok = e.(Error)
	if !ok || err.Scope != GeometryScope || err.Condition != GeneralCondition {
		t.Errorf("Wrong error on test geometry: %v", e)
	}
	if c, ok := err.Values[0].(string); !ok || c != "test error" {
		t.Errorf("Wrong message on test geometry error: %v", e)
	}
}

/*

Test the generated strings for GroupID values.

*/

func TestGroupString(t *testing.T) {
	// group IDs
	s := GroupID{GtypeRow, 1}.String()
	if s != "row 1" {
		t.Errorf("String for row 1 is wrong: %q", s)
	}
	s = GroupID{GtypeCol, 2}.String()
	if s != "column 2" {
		t.Errorf("String for column 2 is wrong: %q", s)
	}
	s = GroupID{"test", 4}.String()
	if s != "test 4" {
		t.Errorf("String for test 4 is wrong: %q", s)
	}
	s = GroupID{}.String()
	if s != "<group> 0" {
		t.Errorf("String for null GroupID is wrong: %q", s)
	}
}

/*

Test the geometry registration mechanism

*/

func TestGeometryRegistration(t *testing.T) {
	// restore known geometries after test
	defer func(gd []*GeometryDescriptor) {
		knownGeometries = gd
	}(knownGeometries)

	newfn := func([]int) (Puzzle, error) { return Puzzle(nil), nil }
	knownGeometries = []*GeometryDescriptor{
		&GeometryDescriptor{[]string{"zeroth", "0th", ""}, 0, newfn},
		&GeometryDescriptor{[]string{"first", "1st"}, 1, newfn},
		&GeometryDescriptor{[]string{"second", "2nd"}, 2, newfn},
	}
	errorGeometries := []*GeometryDescriptor{
		nil,
		&GeometryDescriptor{nil, 0, newfn},
		&GeometryDescriptor{[]string{"", "error1"}, 1, newfn},
		&GeometryDescriptor{[]string{"error2"}, 2, newfn},
		&GeometryDescriptor{[]string{"error3", "second"}, 3, newfn},
		&GeometryDescriptor{[]string{"2nd", "error4"}, 4, newfn},
	}
	goodGeometries := []*GeometryDescriptor{
		&GeometryDescriptor{[]string{"zeroth", "0th", ""}, 0, newfn},
		&GeometryDescriptor{[]string{"third", "3rd"}, 3, newfn},
		&GeometryDescriptor{[]string{"fourth"}, 4, newfn},
	}
	// errors first
	for _, s := range []string{"error-test", "other"} {
		if gd, ok := LookupGeometryByName(s); ok {
			t.Errorf("Lookup of geometry %q returned %+v, %v", s, gd, ok)
		}
	}
	for _, c := range []int{5, 7} {
		if gd, ok := LookupGeometryByCode(c); ok {
			t.Errorf("Lookup of geometry %d returned %+v, %v", c, gd, ok)
		}
	}
	for _, gd := range errorGeometries {
		if e := RegisterGeometry(gd); e == nil {
			t.Errorf("Successfully registered geometry %+v", gd)
		}
	}
	// successes next
	for i, s := range []string{"", "first", "2nd"} {
		gd, ok := LookupGeometryByName(s)
		if !ok || !reflect.DeepEqual(gd, knownGeometries[i]) {
			t.Errorf("Lookup of geometry %q returned %+v, %v", s, gd, ok)
		}
	}
	for _, rd := range knownGeometries {
		gd, ok := LookupGeometryByCode(int(rd.Code))
		if !ok || !reflect.DeepEqual(gd, rd) {
			t.Errorf("Lookup of geometry %d returned %+v, %v", rd.Code, gd, ok)
		}
	}
	knownGeometries = knownGeometries[1:]
	for _, gd := range goodGeometries {
		if e := RegisterGeometry(gd); e != nil {
			t.Errorf("Failed register geometry %+v: %v", *gd, e)
		}
	}
}
