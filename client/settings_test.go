package client

import (
	"html/template"
	"os"
	"path/filepath"
	"testing"
)

/*

template lookup

*/

// testing setup: change default directory since we run from this
// module's directory which is a child of the top.  This applies
// to all the tests in this module.
func init() {
	defaultTemplateDirectory = filepath.Join("..", "static", "tmpl")
}

func TestBasicLookup(t *testing.T) {
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

func TestEnvVarOverride(t *testing.T) {
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
	// now reset to the tests directory and try a test load
	os.Setenv(defaultTemplateDirectoryEnvVar, "tests")
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
