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

package client

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

/*

resource locations

*/

// testing setup: change default directory since we run from this
// module's directory which is a child of the top.  This applies
// to all the tests in this module.
func init() {
	defaultStaticDirectory = filepath.Join("..", "static")
	defaultTemplateDirectory = filepath.Join("..", "static", "tmpl")
}

func TestVerifyResources(t *testing.T) {
	defer func(td, sd string) {
		os.Unsetenv(defaultTemplateDirectoryEnvVar)
		os.Unsetenv(defaultStaticDirectoryEnvVar)
		defaultTemplateDirectory = td
		defaultStaticDirectory = sd
	}(defaultTemplateDirectory, defaultStaticDirectory)

	// should succeed with initalized defaults from above
	if err := VerifyResources(); err != nil {
		t.Errorf("Couldn't verify resources in %q & %q: %v",
			findStaticDirectory(), findTemplateDirectory(), err)
	}

	// should fail with either or both being a non-existent directory
	defaultStaticDirectory = "/no-such-directory"
	if err := VerifyResources(); err == nil {
		t.Errorf("Incorrectly verified resources in %q & %q",
			findStaticDirectory(), findTemplateDirectory())
	}
	defaultStaticDirectory = defaultTemplateDirectory
	defaultTemplateDirectory = "/no-such-directory"
	if err := VerifyResources(); err == nil {
		t.Errorf("Incorrectly verified resources in %q & %q",
			findStaticDirectory(), findTemplateDirectory())
	}
	defaultStaticDirectory = "/no-such-directory"
	defaultTemplateDirectory = "/no-such-directory"
	if err := VerifyResources(); err == nil {
		t.Errorf("Incorrectly verified resources in %q & %q",
			findStaticDirectory(), findTemplateDirectory())
	}

	// should succeed with overrides
	os.Setenv(defaultTemplateDirectoryEnvVar, "testdata")
	os.Setenv(defaultStaticDirectoryEnvVar, "testdata")
	if err := VerifyResources(); err != nil {
		t.Errorf("Couldn't verify resources in %q & %q: %v",
			findStaticDirectory(), findTemplateDirectory(), err)
	}

	// should fail with just one being an existing non-directory
	os.Setenv(defaultTemplateDirectoryEnvVar, "settings_test.go")
	if err := VerifyResources(); err == nil {
		t.Errorf("Incorrectly verified resources in %q & %q",
			findStaticDirectory(), findTemplateDirectory())
	}
	os.Setenv(defaultTemplateDirectoryEnvVar, "testdata")
	os.Setenv(defaultStaticDirectoryEnvVar, "settings_test.go")
	if err := VerifyResources(); err == nil {
		t.Errorf("Incorrectly verified resources in %q & %q",
			findStaticDirectory(), findTemplateDirectory())
	}
}

/*

template lookup

*/

func TestTemplateLookup(t *testing.T) {
	defer func() {
		loadedTemplates = make(map[string]*template.Template)
	}()

	tmpl1, err := loadPageTemplate("error")
	if err != nil {
		t.Fatalf("Failed to load error template: %v", err)
	}
	tmpl2, err := loadPageTemplate("error")
	if err != nil || tmpl2 != tmpl1 {
		t.Errorf("Second load of error template didn't use cache! (%v, %v)", tmpl2, tmpl1)
	}
	tmpl1, err = loadPageTemplate("solver")
	if err != nil {
		t.Fatalf("Failed to load solver template: %v", err)
	}
	tmpl2, err = loadPageTemplate("solver")
	if err != nil || tmpl2 != tmpl1 {
		t.Errorf("Second load of solver template didn't use cache! (%v, %v)", tmpl2, tmpl1)
	}
}

func TestTemplateEnvVarOverride(t *testing.T) {
	defer func() {
		loadedTemplates = make(map[string]*template.Template)
		os.Unsetenv(defaultTemplateDirectoryEnvVar)
	}()

	// first check that we fail with the wrong directory
	os.Setenv(defaultTemplateDirectoryEnvVar, filepath.Join("nosuchdir"))
	_, err := loadPageTemplate("error")
	if err == nil {
		t.Fatalf("Load with OS env directory %v", os.Getenv(defaultTemplateDirectoryEnvVar))
	}
	// now reset to the testdata directory and try a test load
	os.Setenv(defaultTemplateDirectoryEnvVar, "testdata")
	_, err = loadPageTemplate("test")
	if err != nil {
		t.Fatalf("Failed to load test template: %v", err)
	}
	// now unset the environment to use the default
	os.Unsetenv(defaultTemplateDirectoryEnvVar)
	_, err = loadPageTemplate("error")
	if err != nil {
		t.Fatalf("Failed to load error template: %v", err)
	}
}

/*

static lookup

*/

// helper used in two test functions below
func CoreStaticLookup(t *testing.T, shouldPass bool) {
	handlerFunc := func(w http.ResponseWriter, r *http.Request) {
		if StaticHandler(w, r) {
			return
		}
		http.Error(w, "No such static resource", http.StatusNotFound)
	}
	ts := httptest.NewServer(http.HandlerFunc(handlerFunc))
	defer ts.Close()

	for k, v := range staticResourcePaths {
		r, e := http.Get(ts.URL + k)
		if e != nil {
			t.Fatalf("Request failure on existing key %q", k)
		}
		if (r.StatusCode == http.StatusOK) != shouldPass {
			t.Errorf("Bad status on %q: %v %v", k, r.StatusCode, r.Status)
		}
		b, e := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if e != nil {
			t.Fatalf("Couldn't read body: %v", e)
		}
		if (sameAsStaticFile(b, v)) != shouldPass {
			t.Errorf("Got unexpected body for %q:\n%v", k, string(b))
		}
	}

	k := "/static/special/robots.txt"
	r, e := http.Get(ts.URL + k)
	if e != nil {
		t.Fatalf("Request failure on missing key %q", k)
	}
	if r.StatusCode != http.StatusNotFound {
		t.Errorf("Bad status on %q: %v %v", k, r.StatusCode, r.Status)
	}
}

func TestStaticLookup(t *testing.T) {
	log.SetOutput(tLogger{t})
	CoreStaticLookup(t, true)
}

func TestStaticEnvVarOverride(t *testing.T) {
	log.SetOutput(tLogger{t})
	defer func() {
		os.Unsetenv(defaultStaticDirectoryEnvVar)
	}()

	// first check that we fail with the wrong directory
	os.Setenv(defaultStaticDirectoryEnvVar, filepath.Join("nosuchdir"))
	CoreStaticLookup(t, false)

	// now reset to the testdata directory and try a test load
	os.Setenv(defaultStaticDirectoryEnvVar, "testdata")
	priorStaticPaths := staticResourcePaths
	staticResourcePaths = map[string]string{"/test": "testStatic.html"}
	CoreStaticLookup(t, true)
	staticResourcePaths = priorStaticPaths

	// now unset the environment to use the default
	os.Unsetenv(defaultStaticDirectoryEnvVar)
	CoreStaticLookup(t, true)
}

/*

helpers

*/

func sameAsStaticFile(body []byte, fname string) bool {
	path := filepath.Join(findStaticDirectory(), fname)
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		panic(err)
	}
	if fi.Size() != int64(len(body)) {
		return false
	}
	buf := make([]byte, len(body))
	n, err := f.Read(buf)
	if n != len(body) || err != nil {
		panic(fmt.Errorf("Read of %v bytes failed: %v read, %v", len(body), n, err))
	}
	return string(buf) == string(body)
}

/*

log helper for tests

*/

type tLogger struct {
	t *testing.T
}

func (t tLogger) Write(p []byte) (n int, err error) {
	n = len(p)
	t.t.Log(string(p[:n-1]))
	return
}
