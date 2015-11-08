package client

import (
	"bytes"
	"fmt"
	"github.com/ancientHacker/susen.go/puzzle"
	"html/template"
	"os"
	"path/filepath"
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
	if solverPageTemplate == nil {
		solverPageTemplate = template.Must(template.ParseFiles(findSolverPageTemplate()))
	}

	var tp templatePuzzle
	var e error
	if state.Geometry == puzzle.SudokuGeometryCode {
		tp, e = sudokuTemplatePuzzle(state.Values)
	} else if state.Geometry == puzzle.DudokuGeometryCode {
		tp, e = dudokuTemplatePuzzle(state.Values)
	} else {
		e = fmt.Errorf("Can't generate puzzle grid for Geometry Code %v", state.Geometry)
	}
	if e != nil {
		return errorPage(e)
	}

	tsp := templateSolverPage{
		Title:   "Sudoku on the Web",
		CssFile: "../css/puzzle.css",
		JsFile:  "../js/puzzle.js",
		TopHead: "Solver Page v0.5",
		Puzzle:  tp,
	}
	buf := new(bytes.Buffer)
	e = solverPageTemplate.Execute(buf, tsp)
	if e != nil {
		return errorPage(e)
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

var errorPageTemplate *template.Template

// return error page content
func errorPage(e error) string {
	if errorPageTemplate == nil {
		errorPageTemplate = template.Must(template.ParseFiles(findErrorPageTemplate()))
	}

	tep := templateErrorPage{
		Title:         "Sudoku on the Web: Error",
		TopHead:       "Error Page v0.5",
		Message:       e.Error(),
		ReportBugPage: "/report_bug.html",
	}
	buf := new(bytes.Buffer)
	err := errorPageTemplate.Execute(buf, tep)
	if err != nil {
		return fmt.Sprintf("A templating error has occurred: %v", err)
	}
	return buf.String()
}

/*

template location

*/

const defaultTemplateDirectoryEnvVar = "TEMPLATE_DIRECTORY"

var defaultTemplateDirectory = filepath.Join("static", "tmpl")

func findTemplateDirectory() string {
	if dir := os.Getenv(defaultTemplateDirectoryEnvVar); dir != "" {
		return dir
	}
	return defaultTemplateDirectory
}

// Find the solverPage template file
func findSolverPageTemplate() string {
	return filepath.Join(findTemplateDirectory(), "solverPage.tmpl.html")
}

// Find the errorPage template file
func findErrorPageTemplate() string {
	return filepath.Join(findTemplateDirectory(), "errorPage.tmpl.html")
}
