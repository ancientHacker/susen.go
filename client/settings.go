package client

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

/*

Common client settings

*/

const (
	applicationName                = "Sūsen"
	applicationVersion             = "0.6"
	templatePageSuffix             = "Page.tmpl.html"
	defaultTemplateDirectoryEnvVar = "TEMPLATE_DIRECTORY"
	defaultStaticDirectoryEnvVar   = "STATIC_DIRECTORY"
	iconPath                       = "/favicon.ico"
	reportBugPath                  = "/bugreport.html"
)

var (
	defaultTemplateDirectory = filepath.Join("static", "tmpl")
	defaultStaticDirectory   = filepath.Join("static")
	staticResourcePaths      = map[string]string{
		iconPath:      filepath.Join("special", "susen.ico"),
		"/robots.txt": filepath.Join("special", "robots.txt"),
		reportBugPath: filepath.Join("special", "report_bug.html"),
	}
)

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
