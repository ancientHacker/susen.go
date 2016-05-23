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
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

/*

Common client settings

*/

const (
	defaultTemplateDirectoryEnvVar = "TEMPLATE_DIRECTORY"
	defaultStaticDirectoryEnvVar   = "STATIC_DIRECTORY"
	templatePageSuffix             = "Page.tmpl.html"
	applicationNameEnvVar          = "HEROKU_APP_NAME"
	applicationVersionEnvVar       = "HEROKU_RELEASE_VERSION"
	applicationBuildEnvVar         = "HEROKU_SLUG_COMMIT"
	applicationInstanceEnvVar      = "HEROKU_DYNO_ID"
	applicationEnvEnvVar           = "APPLICATION_ENV"
)

var (
	brandName                = "Sūsen"
	iconPath                 = "/favicon.ico"
	reportBugPath            = "/bugreport.html"
	defaultStaticDirectory   = "static"
	defaultTemplateDirectory = filepath.Join(defaultStaticDirectory, "tmpl")
	staticResourcePaths      = map[string]string{
		iconPath:      filepath.Join("special", "susen.ico"),
		"/robots.txt": filepath.Join("special", "robots.txt"),
		reportBugPath: filepath.Join("special", "report_bug.html"),
	}
)

// VerifyResources - check that resources can be found, return
// error if not.
func VerifyResources() error {
	if fi, err := os.Stat(findStaticDirectory()); err != nil {
		return err
	} else if !fi.IsDir() {
		return fmt.Errorf("Static resource location %q not a directory.", findStaticDirectory())
	}
	if fi, err := os.Stat(findTemplateDirectory()); err != nil {
		return err
	} else if !fi.IsDir() {
		return fmt.Errorf("Template resource location %q not a directory.", findTemplateDirectory())
	}
	return nil
}

/*

handle static resources

*/

func findStaticDirectory() string {
	if dir := os.Getenv(defaultStaticDirectoryEnvVar); dir != "" {
		return dir
	}
	return defaultStaticDirectory
}

func StaticHandler(w http.ResponseWriter, r *http.Request) bool {
	path, ok := staticResourcePaths[r.URL.Path]
	if ok {
		log.Printf("Serving static resource for %q", r.URL.Path)
		fp := filepath.Join(findStaticDirectory(), path)
		http.ServeFile(w, r, fp)
	}
	return ok
}

/*

find and parse templates

*/

func findTemplateDirectory() string {
	if dir := os.Getenv(defaultTemplateDirectoryEnvVar); dir != "" {
		return dir
	}
	return defaultTemplateDirectory
}

// parsePageTemplate takes an empty template, finds the template
// Page file associated with the teamplate's name, and parses
// that file's content into the template.
func parsePageTemplate(tmpl *template.Template) (*template.Template, error) {
	name := tmpl.Name()
	fp := filepath.Join(findTemplateDirectory(), name+templatePageSuffix)
	text, err := ioutil.ReadFile(fp)
	if err != nil {
		return nil, err
	}
	tmpl, err = tmpl.Parse(string(text))
	if err != nil {
		return nil, err
	}
	return tmpl, nil
}
