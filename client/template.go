package client

import (
	"bytes"
	"fmt"
	"github.com/ancientHacker/susen.go/puzzle"
	"html/template"
)

/*

solver pages

*/

// The solverPageTemplate contains the template for a solver
// page.  It's initialized when needed; see the definition of
// findSolverPageTemplate for template location details.
var solverPageTemplate *template.Template

// A templateSolverPage contains the values to file the solver
// page template.
type templateSolverPage struct {
	Title, CssFile, JsFile, TopHead string
	Puzzle                          templatePuzzle
}

// templatePuzzle is the structure expected by the puzzle grid
// section of the solver page template.
type templatePuzzle [][]templatePuzzleCell

// A templatePuzzleCell contains the cell's index, value, and CSS
// styling classes as expected by the puzzle grid section of the
// solver page template.
type templatePuzzleCell struct {
	Index                   int
	Value                   template.HTML
	Shade, HBorder, VBorder string
}

// solverPage executes the solverPageTemplate over the passed
// puzzle state, and returns the solver page content as a string.
func solverPage(state puzzle.State) string {
	var tp templatePuzzle
	var err error
	if state.Geometry == puzzle.SudokuGeometryCode {
		tp, err = sudokuTemplatePuzzle(state.Values)
	} else if state.Geometry == puzzle.DudokuGeometryCode {
		tp, err = dudokuTemplatePuzzle(state.Values)
	} else {
		err = fmt.Errorf("Can't generate puzzle grid for Geometry Code %v", state.Geometry)
	}
	if err != nil {
		return errorPage(err)
	}

	tsp := templateSolverPage{
		Title:   fmt.Sprintf("%s v%s", applicationName, applicationVersion),
		CssFile: "../css/puzzle.css",
		JsFile:  "../js/puzzle.js",
		TopHead: solverPageHead,
		Puzzle:  tp,
	}

	tmpl, err := loadPageTemplate("solver")
	if err != nil {
		return errorPage(fmt.Errorf("Couldn't load the %q template: %v", "solver", err))
	}
	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, tsp)
	if err != nil {
		return errorPage(err)
	}
	return buf.String()
}

/*

Sudoku puzzle templates

*/

// sudokuTemplatePuzzle takes the values of a puzzle and returns
// the appropriate templatePuzzle.  Errors mean the given values
// have the wrong shape to be a sudokuPuzzle.
func sudokuTemplatePuzzle(vals []int) (templatePuzzle, error) {
	slen, ok := findIntSquareRoot(len(vals))
	if !ok {
		return nil, fmt.Errorf("Puzzle square count is %v: not a square.", len(vals))
	}
	tlen, ok := findIntSquareRoot(slen)
	if !ok {
		return nil, fmt.Errorf("Puzzle side length is %v: not a square.", slen)
	}
	rows := make(templatePuzzle, slen)
	for i := 0; i < slen; i++ {
		rows[i] = make([]templatePuzzleCell, slen)
		// is this top, bottom, or middle row of quad
		hborder := "middle"
		if i%tlen == 0 {
			hborder = "top"
		} else if i%tlen == tlen-1 {
			hborder = "bottom"
		}
		for j := 0; j < slen; j++ {
			index := i*slen + j
			value := template.HTML("&nbsp;")
			if val := vals[index]; val > 0 {
				value = template.HTML(fmt.Sprint(val))
			}
			quad := i/tlen + j/tlen
			// even quad or odd quad shading
			shade := "lighter"
			if quad%2 == 0 {
				shade = "darker"
			}
			// is this left, center, or right column of quad
			vborder := "center"
			if j%tlen == 0 {
				vborder = "left"
			} else if j%tlen == tlen-1 {
				vborder = "right"
			}
			rows[i][j] = templatePuzzleCell{
				Index:   index + 1,
				Value:   value,
				Shade:   shade,
				HBorder: hborder,
				VBorder: vborder,
			}
		}
	}
	return rows, nil
}

// Find the integer square root of val, if it exists.
func findIntSquareRoot(val int) (int, bool) {
	var i int
	for i = 1; i*i <= val; i++ {
		if i*i == val {
			return i, true
		}
	}
	return i - 1, false
}

/*

Dudoku puzzle templates

*/

// dudokuTemplatePuzzle takes the values of a puzzle and returns
// the appropriate templatePuzzle.  Errors mean the given values
// have the wrong shape to be a dudokuPuzzle.
func dudokuTemplatePuzzle(vals []int) (templatePuzzle, error) {
	slen, ok := findIntSquareRoot(len(vals))
	if !ok {
		return nil, fmt.Errorf("Puzzle square count is %v: not a square.", len(vals))
	}
	htlen, vtlen, ok := findDivisors(slen)
	if !ok {
		return nil, fmt.Errorf("Puzzle side length is %v: not the product of consecutive integers.", slen)
	}
	rows := make(templatePuzzle, slen)
	for i := 0; i < slen; i++ {
		rows[i] = make([]templatePuzzleCell, slen)
		// is this top, bottom, or middle row of quad
		hborder := "middle"
		if i%vtlen == 0 {
			hborder = "top"
		} else if i%vtlen == vtlen-1 {
			hborder = "bottom"
		}
		for j := 0; j < slen; j++ {
			index := i*slen + j
			value := template.HTML("&nbsp;")
			if val := vals[index]; val > 0 {
				value = template.HTML(fmt.Sprint(val))
			}
			quad := i/vtlen + j/htlen
			// even quad or odd quad shading
			shade := "lighter"
			if quad%2 == 0 {
				shade = "darker"
			}
			// is this left, center, or right column of quad
			vborder := "center"
			if j%htlen == 0 {
				vborder = "left"
			} else if j%htlen == htlen-1 {
				vborder = "right"
			}
			rows[i][j] = templatePuzzleCell{
				Index:   index + 1,
				Value:   value,
				Shade:   shade,
				HBorder: hborder,
				VBorder: vborder,
			}
		}
	}
	return rows, nil
}

// findDivisors: find consecutive ints that multiply to give an
// int, if they exist
func findDivisors(val int) (int, int, bool) {
	var low, high int
	for low, high = 1, 2; low*high <= val; low, high = high, high+1 {
		if low*high == val {
			return low, high, true
		}
	}
	return low - 1, low, false
}

/*

error pages

*/

// A templateErrorPage contains the values to fill the error page
// template.
type templateErrorPage struct {
	Title, TopHead, Message, ReportBugPage string
}

// return error page content
func errorPage(e error) string {
	tep := templateErrorPage{
		Title:         fmt.Sprintf("%s: %s", applicationName, "Error"),
		TopHead:       errorPageHead,
		Message:       e.Error(),
		ReportBugPage: "/report_bug.html",
	}

	tmpl, err := loadPageTemplate("error")
	if err != nil {
		return fmt.Sprintf("Couldn't load the %q template: %v", "error", err)
	}

	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, tep)
	if err != nil {
		return fmt.Sprintf("A templating error has occurred: %v", err)
	}
	return buf.String()
}
