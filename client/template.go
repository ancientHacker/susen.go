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

// A templateSolverPage contains the values to fill the solver
// page template.
type templateSolverPage struct {
	SessionID, PuzzleID       string
	Title, TopHead            string
	IconFile, CssFile, JsFile string
	Puzzle                    templatePuzzle
	ApplicationFooter         string
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

// add solver statics to the static list
func init() {
	staticResourcePaths["/solver.js"] = filepath.Join("solver", "puzzle.js")
	staticResourcePaths["/solver.css"] = filepath.Join("solver", "puzzle.css")
}

// SolverPage executes the solver page template over the passed
// session and puzzle info, and returns the solver page content as a
// string.
func SolverPage(sessionID string, puzzleID string, summary *puzzle.Summary) string {
	var tp templatePuzzle
	var err error
	if summary.Geometry == puzzle.StandardGeometryName {
		tp, err = standardTemplatePuzzle(summary.Values)
	} else if summary.Geometry == puzzle.RectangularGeometryName {
		tp, err = rectangularTemplatePuzzle(summary.Values)
	} else {
		err = fmt.Errorf("Can't generate puzzle grid for geometry %q", summary.Geometry)
	}
	if err != nil {
		return ErrorPage(err)
	}

	tsp := templateSolverPage{
		SessionID:         sessionID,
		PuzzleID:          puzzleID,
		Title:             fmt.Sprintf("%s: Solver", brandName),
		TopHead:           fmt.Sprintf("Puzzle Solver"),
		IconFile:          iconPath,
		CssFile:           "/solver.css",
		JsFile:            "/solver.js",
		Puzzle:            tp,
		ApplicationFooter: applicationFooter(),
	}

	tmpl, err := loadPageTemplate("solver")
	if err != nil {
		return ErrorPage(fmt.Errorf("Couldn't load the %q template: %v", "solver", err))
	}
	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, tsp)
	if err != nil {
		return ErrorPage(err)
	}
	return buf.String()
}

/*

Standard puzzle templates

*/

// standardTemplatePuzzle takes the values of a puzzle and returns
// the appropriate templatePuzzle.  Errors mean the given values
// have the wrong shape to be a standardPuzzle.
func standardTemplatePuzzle(vals []int) (templatePuzzle, error) {
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

Rectangular puzzle templates

*/

// rectangularTemplatePuzzle takes the values of a puzzle and returns
// the appropriate templatePuzzle.  Errors mean the given values
// have the wrong shape to be a rectangularPuzzle.
func rectangularTemplatePuzzle(vals []int) (templatePuzzle, error) {
	slen, ok := findIntSquareRoot(len(vals))
	if !ok {
		return nil, fmt.Errorf("Puzzle square count is %v: not a square.", len(vals))
	}
	vtlen, htlen, ok := findDivisors(slen)
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
	Title, TopHead, Message string
	IconFile, ReportBugPage string
	ApplicationFooter       string
}

// return error page content
func ErrorPage(e error) string {
	tep := templateErrorPage{
		Title:             fmt.Sprintf("%s: Error", brandName),
		TopHead:           fmt.Sprintf("Unexpected Server Error"),
		Message:           e.Error(),
		IconFile:          iconPath,
		ReportBugPage:     reportBugPath,
		ApplicationFooter: applicationFooter(),
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

/*

home page

*/

// The homePageTemplate contains the template for a home
// page.  It's initialized when needed; see the definition of
// findHomePageTemplate for template location details.
var homePageTemplate *template.Template

// A templateHomePage contains the values to file the home
// page template.
type templateHomePage struct {
	SessionID, PuzzleID       string
	Title, TopHead            string
	IconFile, CssFile, JsFile string
	PuzzleIDs                 []string
	ApplicationFooter         string
}

// add home statics to the static list
func init() {
	staticResourcePaths["/home.js"] = filepath.Join("home", "home.js")
	staticResourcePaths["/home.css"] = filepath.Join("home", "home.css")
}

// HomePage executes the home page template over the passed
// session and puzzle info, and returns the home page content as
// a string.  If there is an error, what's returned is the error
// page content as a string.
func HomePage(sessionID string, puzzleID string, puzzleIDs []string) string {
	tsp := templateHomePage{
		SessionID:         sessionID,
		PuzzleID:          puzzleID,
		Title:             fmt.Sprintf("%s: Home", brandName),
		TopHead:           fmt.Sprintf("%s", brandName),
		IconFile:          iconPath,
		CssFile:           "/home.css",
		JsFile:            "/home.js",
		PuzzleIDs:         puzzleIDs,
		ApplicationFooter: applicationFooter(),
	}

	tmpl, err := loadPageTemplate("home")
	if err != nil {
		return ErrorPage(fmt.Errorf("Couldn't load the %q template: %v", "home", err))
	}
	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, tsp)
	if err != nil {
		return ErrorPage(err)
	}
	return buf.String()
}

/*

application footer

*/

// applicationFooter - the application footer that shows at the
// bottom of all pages.
func applicationFooter() string {
	appName := os.Getenv(applicationNameEnvVar)
	appEnv := os.Getenv(applicationEnvEnvVar)
	appVersion := os.Getenv(applicationVersionEnvVar)
	appInstance := os.Getenv(applicationInstanceEnvVar)
	appBuild := os.Getenv(applicationBuildEnvVar)

	if appName == "" {
		appName = brandName
	}

	if appEnv == "" {
		appEnv = "local"
	}

	if appVersion != "" {
		appVersion = " " + appVersion
	}
	if len(appBuild) >= 7 {
		appBuild = appBuild[:7]
	}

	if appInstance != "" {
		appInstance = " (dyno " + appInstance + ")"
	}

	switch appEnv {
	case "local":
		return "[" + appName + " local]"
	case "dev":
		return "[" + appName + " CI/CD]"
	case "stg":
		return "[" + appName + appVersion + " <" + appBuild + ">]"
	case "prd":
		return "[" + appName + appVersion + " <" + appBuild + ">" + appInstance + "]"
	}
	return "[" + appName + " <??>]"
}
