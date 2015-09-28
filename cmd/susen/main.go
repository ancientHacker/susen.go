package main

import (
	"fmt"
	"github.com/ancientHacker/susen.go/puzzle"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const cookieName = "susenID"
const cookiePath = "/api"

type susenSession struct {
	id       time.Duration
	steps    []puzzle.Puzzle
	puzzleID int
}

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
	defaultPuzzleID = 0
	startTime       = time.Now()
	sessions        = make(map[time.Duration]*susenSession)
	sessionMutex    sync.RWMutex
)

// since session selection can happen concurrently from
// simultaneous goroutines, it has to be interlocked
func sessionSelect(w http.ResponseWriter, r *http.Request) *susenSession {
	var sessionID time.Duration
	sc, err := r.Cookie(cookieName)
	if err == nil {
		sessionID, err = time.ParseDuration(sc.Value + "ns")
	}
	if err != nil {
		// no session cookie or not a valid session cookie,
		// start a new session with a new cookie
		sessionID = time.Now().Sub(startTime)
		cookieVal := fmt.Sprint(int64(sessionID))
		sc := &http.Cookie{Name: cookieName, Value: cookieVal, Path: cookiePath}
		http.SetCookie(w, sc)
	}
	// we have a valid sessionID, make sure we have a valid session
	sessionMutex.RLock()
	session, ok := sessions[sessionID]
	sessionMutex.RUnlock()
	if ok && session != nil && len(session.steps) > 0 {
		return session
	}
	// initialize and save the new session
	session = &susenSession{id: sessionID}
	session.reset(defaultPuzzleID)
	sessionMutex.Lock()
	sessions[sessionID] = session
	sessionMutex.Unlock()
	return session
}

func (session *susenSession) reset(id int) {
	if id < 0 || id > len(puzzleValues) {
		id = defaultPuzzleID
	}
	session.puzzleID = id
	p, e := puzzle.New(puzzleValues[id])
	if e != nil {
		log.Fatal(e)
	}
	session.steps = []puzzle.Puzzle{p}
	log.Printf("Initialized session %v steps from puzzle %d.", session.id, id+1)
}

func (session *susenSession) addStep(next puzzle.Puzzle) {
	session.steps = append(session.steps, next)
	log.Printf("Added session %v step %d.", session.id, len(session.steps))
}

func (session *susenSession) undoStep() {
	if len(session.steps) > 1 {
		session.steps[len(session.steps)-1] = nil // release current step
		session.steps = session.steps[:len(session.steps)-1]
		log.Printf("Reverted session %v to step %d.", session, len(session.steps))
	} else {
		log.Printf("No steps to undo in session %v.", session)
	}
}

func (session *susenSession) apiHandler(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.URL.Path, "/reset/") {
		re := regexp.MustCompile("/reset/([0-9]+)(/.*)?$")
		if matches := re.FindStringSubmatch(r.URL.Path); matches != nil {
			i, e := strconv.Atoi(matches[1])
			if e != nil {
				// can't happen!
				log.Printf("Atoi failure on %s in %s", matches[1], r.URL.Path)
				session.reset(defaultPuzzleID)
			} else {
				session.reset(i - 1)
			}
		} else {
			session.reset(session.puzzleID)
		}
	}
	if strings.Contains(r.URL.Path, "/back/") {
		session.undoStep()
	}
	switch method := r.Method; method {
	case "GET":
		puzzle.SquaresHandler(session.steps[len(session.steps)-1], w, r)
		log.Printf("Returned current state.")
	case "POST":
		next := session.steps[len(session.steps)-1].Copy()
		_, e := puzzle.AssignHandler(next, w, r)
		if e != nil {
			log.Printf("Assign failed, returned error, no session change.")
		} else {
			log.Printf("Assign succeeded, returned update.")
			session.addStep(next)
		}
	default:
		log.Printf("%s unexpected; no action taken.", method)
	}
}

func main() {
	http.Handle("/static/", http.StripPrefix("/", http.FileServer(http.Dir("."))))
	http.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received %s %s - invoke /api/ handler in session.", r.Method, r.URL.Path)
		session := sessionSelect(w, r)
		session.apiHandler(w, r)
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received %s %s - handle external to session.", r.Method, r.URL.Path)
		if r.URL.Path == "/favicon.ico" {
			http.Error(w, "No custom icon.", http.StatusNotFound)
			return
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
