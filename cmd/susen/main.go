package main

import (
	"github.com/ancientHacker/susen.go/puzzle"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	puzzleValues = [][]int{
		[]int{0,
			4, 0, 0, 0, 0, 3, 5, 0, 2,
			0, 0, 9, 5, 0, 6, 3, 4, 0,
			0, 0, 0, 0, 0, 0, 0, 0, 8,
			0, 0, 0, 0, 3, 4, 8, 6, 0,
			0, 0, 4, 6, 0, 5, 2, 0, 0,
			0, 2, 8, 7, 9, 0, 0, 0, 0,
			9, 0, 0, 0, 0, 0, 0, 0, 0,
			0, 8, 7, 3, 0, 2, 9, 0, 0,
			5, 0, 2, 9, 0, 0, 0, 0, 6,
		},
		[]int{0,
			0, 1, 0, 5, 0, 6, 0, 2, 0,
			0, 0, 0, 0, 0, 3, 0, 1, 8,
			0, 0, 0, 0, 7, 0, 0, 0, 6,
			0, 0, 5, 0, 0, 0, 0, 3, 0,
			0, 0, 8, 0, 9, 0, 7, 0, 0,
			0, 6, 0, 0, 0, 0, 4, 0, 0,
			5, 0, 0, 0, 4, 0, 0, 0, 0,
			6, 4, 0, 2, 0, 0, 0, 0, 0,
			0, 3, 0, 9, 0, 1, 0, 8, 0,
		},
		[]int{0,
			9, 0, 0, 4, 5, 0, 0, 0, 8,
			0, 2, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 1, 7, 2, 4, 0, 0,
			0, 7, 9, 0, 0, 0, 6, 8, 0,
			2, 0, 0, 0, 0, 0, 0, 0, 5,
			0, 4, 3, 0, 0, 0, 2, 7, 0,
			0, 0, 8, 3, 2, 5, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 6, 0,
			4, 0, 0, 0, 1, 6, 0, 0, 3,
		},
		[]int{0,
			9, 4, 8, 0, 5, 0, 2, 0, 0,
			0, 0, 7, 8, 0, 3, 0, 0, 1,
			0, 5, 0, 0, 7, 0, 0, 0, 0,
			0, 7, 0, 0, 0, 0, 3, 0, 0,
			2, 0, 0, 6, 0, 5, 0, 0, 4,
			0, 0, 5, 0, 0, 0, 0, 9, 0,
			0, 0, 0, 0, 6, 0, 0, 1, 0,
			3, 0, 0, 5, 0, 9, 7, 0, 0,
			0, 0, 6, 0, 1, 0, 4, 2, 3,
		},
		[]int{0,
			0, 0, 0, 0, 0, 0, 0, 0, 0,
			9, 0, 0, 5, 0, 7, 0, 3, 0,
			0, 0, 0, 1, 0, 0, 6, 0, 7,
			0, 4, 0, 0, 6, 0, 0, 8, 2,
			6, 7, 0, 0, 0, 0, 0, 1, 3,
			3, 8, 0, 0, 1, 0, 0, 9, 0,
			7, 0, 5, 0, 0, 8, 0, 0, 0,
			0, 2, 0, 3, 0, 9, 0, 0, 8,
			0, 0, 0, 0, 0, 0, 0, 0, 0,
		},
		[]int{0,
			2, 0, 0, 8, 0, 0, 0, 5, 0,
			0, 8, 5, 0, 0, 0, 0, 0, 0,
			0, 3, 6, 7, 5, 0, 0, 0, 1,
			0, 0, 3, 0, 4, 0, 0, 9, 8,
			0, 0, 0, 3, 0, 5, 0, 0, 0,
			4, 1, 0, 0, 6, 0, 7, 0, 0,
			5, 0, 0, 0, 0, 7, 1, 2, 0,
			0, 0, 0, 0, 0, 0, 5, 6, 0,
			0, 2, 0, 0, 0, 0, 0, 0, 4,
		},
	}
	defaultIndex = 0
	currentIndex = defaultIndex
	steps        = []puzzle.Puzzle{}
)

func reset(index int) {
	if index < 0 || index > len(puzzleValues) {
		index = defaultIndex
	}
	currentIndex = index
	p, e := puzzle.New(puzzleValues[index])
	if e != nil {
		log.Fatal(e)
	}
	steps = []puzzle.Puzzle{p}
	log.Printf("Initialized solution from puzzle %d.", index)
}

func addStep(next puzzle.Puzzle) {
	steps = append(steps, next)
	log.Printf("Added solution step %d.", len(steps))
}

func undoStep() {
	if len(steps) > 1 {
		steps[len(steps)-1] = nil // release current step
		steps = steps[:len(steps)-1]
		log.Printf("Reverted to solution step %d.", len(steps))
	}
}

func main() {
	http.Handle("/static/", http.StripPrefix("/", http.FileServer(http.Dir("."))))
	http.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received %s %s: /api handler.", r.Method, r.URL.Path)
		if len(steps) == 0 || strings.Contains(r.URL.Path, "/reset/") {
			re := regexp.MustCompile("/reset/([0-9]+)(/.*)?$")
			if matches := re.FindStringSubmatch(r.URL.Path); matches != nil {
				i, e := strconv.Atoi(matches[1])
				if e != nil {
					// can't happen!
					log.Printf("Atoi failure on %s in %s", matches[1], r.URL.Path)
					reset(defaultIndex)
				} else {
					reset(i - 1)
				}
			} else {
				reset(currentIndex)
			}
		}
		if strings.Contains(r.URL.Path, "/back/") {
			undoStep()
		}
		switch method := r.Method; method {
		case "GET":
			puzzle.SquaresHandler(steps[len(steps)-1], w, r)
			log.Printf("Returned current state.")
		case "POST":
			next := steps[len(steps)-1].Copy()
			_, e := puzzle.AssignHandler(next, w, r)
			if e != nil {
				log.Printf("Assign failed, returning error.")
			} else {
				log.Printf("Assign succeeded, returning update.")
				addStep(next)
			}
		default:
			log.Printf("%s unexpected; no action taken.", method)
		}
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received %s %s: root handler.", r.Method, r.URL.Path)
		if r.URL.Path == "/favicon.ico" {
			http.Error(w, "No custom icon.", http.StatusNotFound)
			return
		}
		if r.URL.Path == "/restart" || r.URL.Path == "/reset" {
			reset(defaultIndex)
		}
		http.Redirect(w, r, "/static/html/puzzle.html", http.StatusFound)
	})

	// Heroku environment port sensing
	port := os.Getenv("PORT")
	if port == "" {
		// running locally in dev mode
		port = "localhost:8080"
	} else {
		// running as a true server
		port = ":" + port
	}

	log.Printf("Listening on %s...", port)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal("Listener failure: ", err)
	}
}
