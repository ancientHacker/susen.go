package main

import (
	"github.com/ancientHacker/susen.go/client"
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
const cookiePath = "/"

type susenSession struct {
	sessionID string
	puzzleID  string
	steps     []puzzle.Puzzle
}

var (
	puzzleValues = map[string][]int{
		"1-star": []int{0,
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
		"2-star": []int{0,
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
		"3-star": []int{0,
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
		"4-star": []int{0,
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
		"5-star": []int{0,
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
		"6-star": []int{0,
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
	defaultPuzzleID = "1-star"
	startTime       = time.Now()
	sessions        = make(map[string]*susenSession)
	sessionMutex    sync.RWMutex
)

// getCookie gets the session cookie, or sets a new one.  It
// returns the session ID associated with the cookie.
//
// The logic here was meant to be very simple, because it was
// designed for only one server instance (which is all we support
// right now), so each browser was given a cookie based on the
// time (to the nanosecond) of the first request we received from
// that browser.  Then the browser's notion of session cookie
// lifetime would control the extent of that session: if it
// thought it was in a different session it would not send the
// cookie.
//
// Unfortunately, this breaks down for Heroku-served instances,
// because the same server instance gets both HTTP and HTTP
// traffic, which look to the browser like different sessions
// even though they have the same endpoint.  Since HTTP cookies
// can be given to HTTPS connections to the same endpoint,
// browsers that start in HTTP and move to HTTPS will give the
// HTTP cookie to the HTTPS endpoint and thus be using the same
// puzzle as they had in HTTP, but they will have established a
// different local session and thus the client will think he can
// change puzzles etc. without affecting the HTTP session.
//
// The solution to this problem is to notice when we are running
// under Heroku and make sure that browser tabs which use
// different source protocols get different sessions, even if
// they try submitting an existing cookie from the other tab.
func getCookie(w http.ResponseWriter, r *http.Request) string {
	proto := "httpx" // absent other indicators, protocol is unknown

	// Issue #1: Heroku-transported protocols are specified in a header
	if herokuProtocol := r.Header.Get("X-Forwarded-Proto"); herokuProtocol != "" {
		proto = herokuProtocol
	}

	// check for an existing cookie whose value matches the protocol
	if sc, e := r.Cookie(cookieName); e == nil {
		if m, e := regexp.MatchString(proto+"-[0-9a-z]{3,}", sc.Value); e == nil && m {
			return sc.Value
		}
	}

	// no session cookie or not a valid session cookie,
	// start a new session with a new cookie
	sid := proto + "-" + strconv.FormatInt(int64(time.Now().Sub(startTime)), 36)
	sc := &http.Cookie{Name: cookieName, Value: sid, Path: cookiePath}
	http.SetCookie(w, sc)
	return sid
}

// since session selection can happen concurrently from
// simultaneous goroutines, it has to be interlocked
func sessionSelect(w http.ResponseWriter, r *http.Request) *susenSession {
	sessionID := getCookie(w, r)
	// look up the session for the cookie
	sessionMutex.RLock()
	session, ok := sessions[sessionID]
	sessionMutex.RUnlock()
	if ok && session != nil && len(session.steps) > 0 {
		return session
	}
	// initialize and save the new session
	session = &susenSession{sessionID: sessionID}
	session.reset(defaultPuzzleID)
	sessionMutex.Lock()
	sessions[sessionID] = session
	sessionMutex.Unlock()
	return session
}

func (session *susenSession) reset(puzzleID string) {
	vals, ok := puzzleValues[puzzleID]
	if ok {
		session.puzzleID = puzzleID
	} else {
		session.puzzleID, vals = defaultPuzzleID, puzzleValues[defaultPuzzleID]
	}
	p, e := puzzle.New(vals)
	if e != nil {
		log.Fatal(e)
	}
	session.steps = []puzzle.Puzzle{p}
	log.Printf("Initialized session %v from puzzle %q.", session.sessionID, session.puzzleID)
}

func (session *susenSession) addStep(next puzzle.Puzzle) {
	session.steps = append(session.steps, next)
	log.Printf("Added session %v step %d.", session.sessionID, len(session.steps))
}

func (session *susenSession) undoStep() {
	if len(session.steps) > 1 {
		session.steps[len(session.steps)-1] = nil // release current step
		session.steps = session.steps[:len(session.steps)-1]
		log.Printf("Reverted session %v to step %d.", session.sessionID, len(session.steps))
	} else {
		log.Printf("No steps to undo in session %v.", session.sessionID)
	}
}

func (session *susenSession) apiHandler(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.URL.Path, "/reset/") {
		session.reset(session.puzzleID)
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

func (session *susenSession) solverHandler(w http.ResponseWriter, r *http.Request) {
	curpuz := session.steps[len(session.steps)-1]
	state := curpuz.State()
	body := client.SolverPage(session.sessionID, session.puzzleID, state)
	hs := w.Header()
	hs.Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(body))
}

func main() {
	http.Handle("/static/", http.StripPrefix("/", http.FileServer(http.Dir("."))))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/favicon.ico" {
			log.Printf("Received site icon request.")
			http.ServeFile(w, r, "static/img/susen.ico")
			return
		}
		log.Printf("Handling %s %s...", r.Method, r.URL.Path)
		session := sessionSelect(w, r)
		switch {
		case strings.HasPrefix(r.URL.Path, "/reset/"):
			if len(r.URL.Path) > len("/reset/") {
				session.reset(r.URL.Path[len("/reset/"):])
			} else {
				session.reset(session.puzzleID)
			}
		case strings.HasPrefix(r.URL.Path, "/api/"):
			session.apiHandler(w, r)
			return
		case strings.HasPrefix(r.URL.Path, "/solver/"):
			session.solverHandler(w, r)
			return
		}
		http.Redirect(w, r, "/solver/", http.StatusFound)
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
