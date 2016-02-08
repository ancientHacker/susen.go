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
	"fmt"
)

/*

Errors

*/

// An Error describes a problem with a puzzle or a requested
// operation.  It can produce an error message in English, but
// its main function is to support localized error messaging by
// clients.  It tells the client "this thing failed to meet this
// condition", and provides supplemental details about the thing
// and the condition.
type Error struct {
	Scope     ErrorScope     `json:"scope"`
	Structure ErrorStructure `json:"structure,omitempty"`
	Condition ErrorCondition `json:"condition,omitempty"`
	Attribute ErrorAttribute `json:"attribute,omitempty"`
	Values    ErrorData      `json:"values,omitempty"`
	Message   string         `json:"message,omitempty"` // custom message
}

// An ErrorScope explains what type of thing the error is
// referring to.  In the case of client errors, this is either a
// client-supplied argument or some aspect of the resulting
// puzzle.  In the case of internal logic errors, this is where
// in the code the failure occurred.
type ErrorScope int

// Constants for the various error scopes.
const (
	UnknownScope ErrorScope = iota
	RequestScope
	ArgumentScope
	GeometryScope
	GroupScope
	SquareScope
	InternalScope
	MaxScope
)

// The ErrorStructure denotes whether the problem is in the
// overall Scope, an Attribute of the Scope, or the value of an
// Attribute of the Scope.
type ErrorStructure int

// Constants for the various structure codes.
const (
	UnknownStructure ErrorStructure = iota
	ScopeStructure
	AttributeStructure
	AttributeValueStructure
	MaxStructure
)

// The ErrorCondition is the predicate that the
// scope/attribute/value failed to satisfy.  There are a bunch of
// known, named predicates and then a "general" (arbitrary
// English string) predicate for runtime errors.
type ErrorCondition int

// Constants for the various error conditions
const (
	UnknownCondition ErrorCondition = iota
	GeneralCondition
	TooLargeCondition
	TooSmallCondition
	DuplicateAssignmentCondition
	NotInSetCondition
	NoPossibleValuesCondition
	NoGroupValueCondition
	DuplicateGroupValuesCondition
	UnknownGeometryCondition
	NonSquareCondition
	NonRectangularCondition
	InvalidPuzzleAssignmentCondition
	WrongPuzzleSizeCondition
	InvalidArgumentCondition
	MismatchedSummaryErrorsCondition
	MaxCondition
)

// An ErrorAttribute names the attribute that has a problem.
type ErrorAttribute int

// Constants for the various attribute codes.
const (
	UnknownAttribute ErrorAttribute = iota
	DecodeAttribute
	EncodeAttribute
	URLAttribute
	LocationAttribute
	NamedAttribute
	GeometryAttribute
	IndexAttribute
	ValueAttribute
	AssignedValueAttribute
	BoundValueAttribute
	RemovedValueAttribute
	RemovedValuesAttribute
	RetainedValuesAttribute
	PuzzleSizeAttribute
	SideLengthAttribute
	PuzzleAttribute
	SummaryAttribute
	MaxAttribute
)

// The ErrorData provides details about the thing that failed to
// meet the predicate (such as the value of an attribute) as well
// as the predicate itself (such as minimum required values).
//
// Every item in the slice of ErrorData is required to be
// JSON-serializable, so it can be returned to web clients.
// Sadly, there is no good way to express this condition in a way
// the compiler can check it, so we just have to rely on
// implementors to "do the right thing" and check the condition
// at runtime.
type ErrorData []interface{}

// Return an error string from an Error.  If the Error has a
// pre-canned message, this will use it, otherwise it will
// produce an appropriate (English, non-localized) message.
func (e Error) Error() string {
	es := e.Message
	if len(es) > 0 {
		return es
	}
	values := e.Values
	nextVal := func() interface{} {
		if len(values) == 0 {
			return "<unknown>"
		}
		val := values[0]
		values = values[1:]
		return val
	}
	switch e.Scope {
	case RequestScope:
		es = "Invalid request: "
	case ArgumentScope:
		es = "Invalid argument: "
	case GeometryScope:
		es = "Invalid geometry: "
	case GroupScope:
		es = fmt.Sprintf("Problem in %v: ", nextVal())
	case SquareScope:
		es = fmt.Sprintf("Problem in square %v: ", nextVal())
	case InternalScope:
		es = "Internal logic error: "
	default:
		es = "Unknown error: "
	}
	if e.Structure == AttributeStructure || e.Structure == AttributeValueStructure {
		switch e.Attribute {
		case DecodeAttribute:
			es += "JSON Decode error"
		case EncodeAttribute:
			es += "JSON Encode error"
		case URLAttribute:
			es += "Resource path"
		case NamedAttribute:
			es += fmt.Sprint(nextVal())
		case GeometryAttribute:
			es += "Geometry"
		case IndexAttribute:
			es += "Index"
		case ValueAttribute:
			es += "Value"
		case AssignedValueAttribute:
			es += "Assigned value"
		case BoundValueAttribute:
			es += "Bound value"
		case RemovedValueAttribute:
			es += "Removed value"
		case RemovedValuesAttribute:
			es += "Removed values"
		case RetainedValuesAttribute:
			es += "Retained values"
		case PuzzleSizeAttribute:
			es += "Puzzle size"
		case PuzzleAttribute:
			es += "Puzzle"
		case SummaryAttribute:
			es += "Summary"
		case SideLengthAttribute:
			es += "Side length"
		case LocationAttribute:
			es += fmt.Sprintf("In puzzle.%v", nextVal())
		default:
			es += "<Unknown attribute>"
		}
		if e.Structure == AttributeValueStructure {
			es += " (" + fmt.Sprint(nextVal()) + ")"
		}
		es += ": "
	}
	switch e.Condition {
	case GeneralCondition:
		es += fmt.Sprint(nextVal())
	case TooLargeCondition:
		es += fmt.Sprintf("Must be at most %v", nextVal())
	case TooSmallCondition:
		es += fmt.Sprintf("Must be at least %v", nextVal())
	case DuplicateAssignmentCondition:
		es += fmt.Sprintf("Square %v is already assigned value %v", nextVal(), nextVal())
	case NotInSetCondition:
		es += fmt.Sprintf("Must be in possible values %v", nextVal())
	case NoPossibleValuesCondition:
		es += fmt.Sprintf("No remaining possible values")
	case NoGroupValueCondition:
		es += fmt.Sprintf("No square can contain %v", nextVal())
	case DuplicateGroupValuesCondition:
		es += fmt.Sprintf("Multiple squares have or need value %v", nextVal())
	case UnknownGeometryCondition:
		es += fmt.Sprintf("Not a known geometry")
	case NonSquareCondition:
		es += fmt.Sprintf("Not a perfect square")
	case NonRectangularCondition:
		es += fmt.Sprintf("Not the product of consecutive integers")
	case InvalidPuzzleAssignmentCondition:
		es += fmt.Sprintf("Target puzzle has errors; no assignments are allowed")
	case WrongPuzzleSizeCondition:
		es += fmt.Sprintf("Doesn't match specified side length (%v)", values)
	case InvalidArgumentCondition:
		es += fmt.Sprintf("Required value was missing or invalid")
	case MismatchedSummaryErrorsCondition:
		es += fmt.Sprintf("Summary has errors but puzzle created from it does not")
	default:
		es += fmt.Sprintf("Supplemental data is %v", values)
	}
	return es
}
