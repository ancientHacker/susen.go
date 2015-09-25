// Copyright 2015 Daniel C. Brotsky.  All rights reserved.

// Package puzzle provides a model for Sudoku puzzles and
// operations on them.  It supports both a golang interface and a
// web interface to the puzzles.
//
// In this package, Sudoku puzzles are made of squares which are
// either empty (represented with a 0 value) or have an assigned
// value between 1 and the side length of the puzzle (inclusive).
// The squares are designated by indices that start at 1 and
// increase left-to-right, top-to-bottom (English reading order).
//
// For each empty square in a puzzle, the implementation
// maintains a set of possible values the square can be assigned
// without conflicting with other squares.  Exactly which other
// squares might conflict depends on the puzzle's geometry, which
// determines which groups of squares are constrained to have the
// full range of possible values.
//
// All Sudoku geometries have a group for each row and column.
// The most common geometry additionally requires the side length
// of a puzzle to be a perfect square, and then each
// non-overlapping subtile of the overall puzzle is itself a
// group.  Some sudoku geometries instead use rectangular (e.g.,
// 2x3) subtiles, making the side length of the overall square
// the product of the subtile sides.  And some sudoku geometries
// add the diagonals of the overall square as groups.
//
// If a square in a group is the only possible location for a
// needed value, we say that the square is bound by the group,
// and the implementation tracks these bound squares.  If an
// assignment of some other value is made to that square, the
// puzzle will not be solvable, and is deemed invalid.  Invalid
// puzzles can also arise from assignments of the same value to
// multiple squares in a group.  The implementation will not
// perform operations on invalid puzzles.
package puzzle

import (
	"fmt"
)

// Puzzle is the interface to puzzle objects, whose
// implementation is opaque.  This module implements a RESTful
// wrapper form of this API, so it's easy to build web services
// over Puzzles.
//
// Wherever an error is returned from Update, it should contain
// an Error value, and all the implementations built into this
// module work that way.  However, since implementations of
// geometries can be registered by other modules, this module
// can't guarantee that an Error will always be returned.
// Clients should guard against this possibility, as do the
// RESTful wrappers built into this module.
type Puzzle interface {
	State() State
	Squares() []Square
	Solutions() []Solution
	Assign(choice Choice) (Update, error)
	Copy() Puzzle
}

// New either returns a Puzzle with the specified geometry and
// cell values or an error (if the geometry and cell values don't
// work well together).  The geometry code should be the first
// value of the input array and the remaining values are the cell
// values (for the cell with the matching index).  As usual,
// assigned cell values range from 1 to the side length of the
// puzzle; cell values of 0 mean an empty cell.
//
// When an error is returned from this function, it will always
// contain an Error value, even if the New implementation of the
// particular geometry is misbehaved and doesn't.
func New(geoAndValues []int) (Puzzle, error) {
	if len(geoAndValues) == 0 {
		return nil, Error{
			Scope:     ArgumentScope,
			Structure: ScopeStructure,
			Condition: EmptyArgumentCondition,
		}
	}
	g, ok := LookupGeometryByCode(geoAndValues[0])
	if !ok {
		return nil, Error{
			Scope:     GeometryScope,
			Structure: AttributeValueStructure,
			Attribute: GeometryAttribute,
			Condition: UnknownGeometryCondition,
			Values:    ErrorData{geoAndValues[0]},
		}
	}
	p, e := g.New(geoAndValues[1:])
	if e != nil {
		if err, ok := e.(Error); !ok {
			err = Error{
				Scope:     GeometryScope,
				Structure: ScopeStructure,
				Condition: GeneralCondition,
				Values:    ErrorData{e.Error()},
			}
			e = err
		}
		return nil, e
	}
	return p, nil
}

// The State of a puzzle gives its geometry, side length,
// cell values, and any known problems with the puzzle.
type State struct {
	Geometry  int     `json:"geometry"`
	SideLenth int     `json:"sidelen"`
	Values    []int   `json:"values"`
	Errors    []Error `json:"errors,omitempty"`
}

// A Square in a puzzle gives the square's index, assigned value
// (if any), bound value (if any, with sources), and possible
// values (if more than one).  Puzzle squares are numbered
// left-to-right, top-to-bottom, starting at 1, and the sequence
// of squares is returned in that order.
//
// Only required fields should be specified in a Square, so as to
// minimize the Square's JSON-encoded form (which is used for
// transmission of puzzle data from server to client).  If an
// Aval (user-assigned value) is specified, no other fields
// should be present.  The Pvals (possible values) field should
// only be present if there are multiple possible values; if the
// square has only one possible value it should be specified as
// the Aval or the Bval (bound value).  A Bsrc (bound value
// source) should only be present if a row, column, or tile
// requires that bound value be assigned to the Square.
type Square struct {
	Index int       `json:"index"`
	Aval  int       `json:"aval,omitempty"`
	Bval  int       `json:"bval,omitempty"`
	Bsrc  []GroupID `json:"bsrc,omitempty"`
	Pvals intset    `json:"pvals,omitempty"`
}

// A GroupID names a row, column, tile, diagonal, or other set of
// constrained squares, collectively called groups.  The
// numbering and cardinality for each type of group is 1-based
// and determined by the puzzle geometry.
type GroupID struct {
	Gtype string `json:"gtype"`
	Index int    `json:"index"`
}

// Group IDs implement Stringer
func (gid GroupID) String() string {
	if gid.Gtype == "" {
		return fmt.Sprintf("<group> %d", gid.Index)
	}
	return fmt.Sprintf("%s %d", gid.Gtype, gid.Index)
}

// GType (group type) constants.  These are human-readable but
// not localized.  Registered geometries may define other group
// types, so clients should consult the documentation for each
// such geometry.
const (
	GtypeRow      = "row"
	GtypeCol      = "column"
	GtypeTile     = "tile"
	GtypeDiagonal = "diagonal"
)

// A Choice assigns a value to a cell.  The cell is referred to
// by its index.
type Choice struct {
	Index int `json:"index"`
	Value int `json:"value"`
}

// An Update to a puzzle is the result of an assignment, giving
// any changed squares.  If there was a problem performing the
// assignment, or if performing the assignment produced errors in
// the underlying puzzle, they are reported here.  (Any puzzle
// errors will also be available in the puzzle's state.)
type Update struct {
	Squares []Square `json:"squares,omitempty"`
	Errors  []Error  `json:"conflict,omitempty"`
}

// A Solution is a filled-in puzzle (expressed as its values)
// plus the sequence of choices for empty squares that were made
// to get there.  Solutions tend to have far fewer choices than
// originally empty squares, because most of the empty squares in
// most puzzles have their values forced (bound) by puzzle
// structure.  These bound values are present only in the solved
// puzzle, not in the choice list.
type Solution struct {
	Values  []int    `json:"values"`
	Choices []Choice `json:"choices,omitempty"`
}

/*

Puzzle Geometries

*/

// A GeometryDescriptor is used to register a geometry: it gives
// the geometry some names and a code (all of which must be
// unique among those registered) and provides a constructor for
// puzzle objects with that geometry.  The order of the names
// doesn't matter.  The code allows for more compact puzzle
// storage, and is not expected to be used by humans.
//
// Best practice for registered geometries is to return an Error
// from New on failure; if a non-Error is returned then it's
// converted to a Geometry Error.
type GeometryDescriptor struct {
	Names []string
	Code  byte
	New   func([]int) (Puzzle, error)
}

// The registry of known geometries.  We use a linear list
// because we're not expecting a lot of geometries, so a linear
// lookup seems fine.  Registration is expected to be done at
// initialization time, but the world doesn't end if you add new
// geometries later.
var knownGeometries []*GeometryDescriptor

// LookupGeometryByName is how people look up geometries.  The
// return value will be nil if the name is not registered.
// There's also a boolean return value to tell you if we found a
// descriptor for the name, similar to a map lookup.
func LookupGeometryByName(name string) (*GeometryDescriptor, bool) {
	for _, rd := range knownGeometries {
		for _, n := range rd.Names {
			if n == name {
				return rd, true
			}
		}
	}
	return nil, false
}

// LookupGeometryByCode is how the programs look up geometries.
// The return value be nil if the code is not registered.
// There's also a boolean return value to tell you if we found a
// descriptor for the code, similar to a map lookup.
func LookupGeometryByCode(code int) (*GeometryDescriptor, bool) {
	for _, rd := range knownGeometries {
		if int(rd.Code) == code {
			return rd, true
		}
	}
	return nil, false
}

// RegisterGeometry is how you tell the module about new
// geometries.  It's used by both internal and external puzzle
// implementations.
func RegisterGeometry(gd *GeometryDescriptor) error {
	if gd == nil {
		return fmt.Errorf("Can't register a null geometry")
	}
	if len(gd.Names) == 0 {
		return fmt.Errorf("Can't register a geometry with no names")
	}
	if len(gd.Names[0]) == 0 {
		return fmt.Errorf("Can't register a geometry whose first name is empty.")
	}
	if rd, ok := LookupGeometryByCode(int(gd.Code)); ok != false {
		return fmt.Errorf("Geometry %q is already using code %d", rd.Names[0], gd.Code)
	}
	for _, gn := range gd.Names {
		if rd, ok := LookupGeometryByName(gn); ok != false {
			return fmt.Errorf("Geometry %q is already using name %q", rd.Names[0], gn)
		}
	}
	knownGeometries = append(knownGeometries, gd)
	return nil
}
