package client

import (
	"html/template"
	"os"
	"path/filepath"
)

/*

Common client settings

*/

const (
	applicationName                = "Sudoku on the Web"
	applicationVersion             = "0.6"
	solverPageHead                 = "Puzzle Solver"
	errorPageHead                  = "Error Encountered"
	templatePageSuffix             = "Page.tmpl.html"
	defaultTemplateDirectoryEnvVar = "TEMPLATE_DIRECTORY"
)

var (
	defaultTemplateDirectory = filepath.Join("static", "tmpl")
)

/*

find and parse templates

*/

func findTemplateDirectory() string {
	if dir := os.Getenv(defaultTemplateDirectoryEnvVar); dir != "" {
		return dir
	}
	return defaultTemplateDirectory
}

// loadedTemplates is the cache of already-parsed templates
var loadedTemplates = make(map[string]*template.Template)

// loadPageTemplate does what you would expect: give it the
// template name, and it will find and parse the template file
// and return the resulting template.
func loadPageTemplate(name string) (*template.Template, error) {
	if tmpl, ok := loadedTemplates[name]; ok {
		return tmpl, nil
	}
	fp := filepath.Join(findTemplateDirectory(), name+templatePageSuffix)
	tmpl, err := template.ParseFiles(fp)
	if err != nil {
		return nil, err
	}
	loadedTemplates[name] = tmpl
	return tmpl, nil
}
