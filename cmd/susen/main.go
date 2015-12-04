package main

import (
	"encoding/json"
	"github.com/ancientHacker/susen.go/Godeps/_workspace/src/github.com/garyburd/redigo/redis"
	"github.com/ancientHacker/susen.go/client"
	"github.com/ancientHacker/susen.go/puzzle"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"
)

func main() {
	// establish redis connection
	url := redisUrl()
	if err := redisConnect(url); err != nil {
		log.Fatalf("Exiting: No redis server at %q: %v", url, err)
	}

	// port sensing
	port := os.Getenv("PORT")
	if port == "" {
		// running locally in dev mode
		port = "localhost:8080"
	} else {
		// running as a true server
		port = ":" + port
	}

	// catch signals
	shutdownOnSignal()

	// serve
	log.Printf("Listening on %s...", port)
	http.HandleFunc("/", serveHttp)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Print("Listener failure: ", err)
		shutdown(listenerFailureShutdown)
	}
}

/*

request handlers

*/

func serveHttp(w http.ResponseWriter, r *http.Request) {
	if client.StaticHandler(w, r) {
		return
	}
	session := sessionSelect(w, r)
	session.rootHandler(w, r)
}

func (session *susenSession) apiHandler(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/reset/") {
		session.reset(session.puzzleID)
	}
	if strings.HasPrefix(r.URL.Path, "/api/back/") {
		session.undoStep()
	}
	switch method := r.Method; method {
	case "GET":
		puzzle.SquaresHandler(session.steps[len(session.steps)-1], w, r)
		log.Printf("Returned current squares for %s:%s step %d",
			session.sessionID, session.puzzleID, len(session.steps))
	case "POST":
		next := session.steps[len(session.steps)-1].Copy()
		update, e := puzzle.AssignHandler(next, w, r)
		if e != nil {
			log.Printf("Assign to %s:%s step %d failed, returned error.",
				session.sessionID, session.puzzleID, len(session.steps))
		} else {
			session.addStep(next)
			if update.Errors != nil {
				log.Printf("Assign to %s:%s gave errors; step %d is unsolvable.",
					session.sessionID, session.puzzleID, len(session.steps))
			}
		}
	default:
		log.Printf("%s of %s:%s step %d unexpected; no action taken.",
			method, session.sessionID, session.puzzleID, len(session.steps))
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
	log.Printf("Returned solver page for %s:%s step %d.",
		session.sessionID, session.puzzleID, len(session.steps))
}

func (session *susenSession) homeHandler(w http.ResponseWriter, r *http.Request) {
	body := client.HomePage(session.sessionID, session.puzzleID, nil)
	hs := w.Header()
	hs.Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(body))
	log.Printf("Returned home page for %s:%s step %d.",
		session.sessionID, session.puzzleID, len(session.steps))
}

func (session *susenSession) rootHandler(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasPrefix(r.URL.Path, "/reset/"):
		http.Redirect(w, r, "/solver/", http.StatusFound)
		log.Printf("Redirected to solver page for %s:%s step %d.",
			session.sessionID, session.puzzleID, len(session.steps))
	case strings.HasPrefix(r.URL.Path, "/api/"):
		session.apiHandler(w, r)
	case strings.HasPrefix(r.URL.Path, "/solver/"):
		session.solverHandler(w, r)
	case strings.HasPrefix(r.URL.Path, "/home/"):
		session.homeHandler(w, r)
	default:
		http.Redirect(w, r, "/home/", http.StatusFound)
		log.Printf("Redirected to home page for %s:%s step %d.",
			session.sessionID, session.puzzleID, len(session.steps))
	}
}

/*

session handling

*/

type susenSession struct {
	sessionID string
	puzzleID  string
	steps     []puzzle.Puzzle
}

const (
	cookieNameBase = "susenID"
	cookiePath     = "/"
	cookieMaxAge   = 3600 * 24 * 7 * 52 // 1 year
)

var (
	startTime    = time.Now()
	sessions     = make(map[string]*susenSession)
	sessionMutex sync.RWMutex
)

// getCookie gets the session cookie, or sets a new one.  It
// returns the session ID associated with the cookie.
//
// The logic around session cookies is that we have one cookie
// name per connection protocol, so that opening both a secure
// and an insecure session from the same browser gives you two
// sessions.  We need to do this because we often have one
// instance serving both protocols, with the protocol termination
// done at a load-balancer.
func getCookie(w http.ResponseWriter, r *http.Request) string {
	// Issue #1: Heroku-transported protocols are specified in a header
	proto := "httpx" // absent other indicators, protocol is unknown
	if herokuProtocol := r.Header.Get("X-Forwarded-Proto"); herokuProtocol != "" {
		proto = herokuProtocol
	}

	// check for an existing cookie whose name matches the protocol
	cookieName := cookieNameBase + "-" + proto
	if sc, e := r.Cookie(cookieName); e == nil && sc.Value != "" {
		return sc.Value
	}

	// no session cookie or not a valid session cookie,
	// start a new session with a new cookie
	sid := strconv.FormatInt(int64(time.Now().Sub(startTime)), 36)
	if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
		// use request ID, if present, for uniqueness
		sid = requestID
	}
	log.Printf("No session cookie found, creating new session ID %q", sid)
	sc := &http.Cookie{Name: cookieName, Value: sid, Path: cookiePath, MaxAge: cookieMaxAge}
	http.SetCookie(w, sc)
	return sid
}

// sessionSelect: find or create the session for the current connection.
func sessionSelect(w http.ResponseWriter, r *http.Request) *susenSession {
	// check to see if this is a force reset of the session
	forceReset, resetID := false, ""
	if strings.HasPrefix(r.URL.Path, "/reset/") {
		forceReset = true
		resetID = r.URL.Path[len("/reset/"):]
	}
	sessionID := getCookie(w, r)
	// look up in-memory session for the cookie, if there is one.
	sessionMutex.RLock()
	session, ok := sessions[sessionID]
	sessionMutex.RUnlock()
	if ok && session != nil {
		if forceReset {
			session.reset(resetID)
		}
		return session
	}
	// create and remember the in-memory session
	session = &susenSession{sessionID: sessionID, puzzleID: defaultPuzzleID}
	sessionMutex.Lock()
	sessions[sessionID] = session
	sessionMutex.Unlock()
	// initialize or reload it
	if forceReset {
		session.reset(resetID)
	} else if session.redisLookup() {
		session.redisLoad()
		log.Printf("Reloaded session %v, puzzle %q, through step %d.",
			session.sessionID, session.puzzleID, len(session.steps))
	} else {
		session.reset(defaultPuzzleID)
	}
	return session
}

// reset: reset the session from an explict puzzleID or its default
func (session *susenSession) reset(puzzleID string) {
	// reset to the given puzzleID, making sure it's valid
	if puzzleID == "" {
		puzzleID = session.puzzleID
	}
	vals, ok := puzzleValues[puzzleID]
	if ok {
		session.puzzleID = puzzleID
	} else {
		session.puzzleID, vals = defaultPuzzleID, puzzleValues[defaultPuzzleID]
	}

	// start with steps equal to the puzzle
	p, e := puzzle.New(vals)
	if e != nil {
		log.Printf("Failed to create puzzle %q: %v", puzzleID, e)
		shutdown(runtimeFailureShutdown)
	}
	session.steps = []puzzle.Puzzle{p}
	session.redisInit()
	log.Printf("Reset session %v from puzzle %q.", session.sessionID, session.puzzleID)
}

func (session *susenSession) addStep(next puzzle.Puzzle) {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()
	session.steps = append(session.steps, next)
	session.redisAddStep(next)
	log.Printf("Added session %v:%v step %d.",
		session.sessionID, session.puzzleID, len(session.steps))
}

func (session *susenSession) undoStep() {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()
	if len(session.steps) > 1 {
		session.steps[len(session.steps)-1] = nil // release current step
		session.steps = session.steps[:len(session.steps)-1]
		session.redisUndoStep()
		log.Printf("Reverted session %v:%v to step %d.",
			session.sessionID, session.puzzleID, len(session.steps))
	}
}

/*

persistence layer

*/

// puzzle data
var (
	defaultPuzzleID = "1-star"
	puzzleValues    = map[string][]int{
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
)

// keys for persisted session values
var (
	puzzleIDKey = ":puzzleID"
	stepsKey    = ":steps"
)

// current connection data
var (
	rdc     redis.Conn
	rdUrl   string
	rdMutex sync.Mutex
)

// redisUrl - where to find the database
func redisUrl() string {
	db := os.Getenv("REDISTOGO_DB")
	if db == "" {
		db = "0" // scratch database
	}
	url := os.Getenv("REDISTOGO_URL")
	if url == "" {
		url = "redis://localhost:6379/" + db
	} else {
		url = url + db
	}
	return url
}

// redisConnect: connect to the given Redis URL.  Returns the
// connection, if successful, nil otherwise.
func redisConnect(url string) error {
	conn, err := redis.DialURL(url)
	if err == nil {
		log.Printf("Connected to redis at %q", url)
		rdc, rdUrl = conn, url
		return nil
	}
	return err
}

// redisClose: close the given Redis connection.
func redisClose() {
	if rdc != nil {
		log.Print("Closed connection to redis.")
		rdc.Close()
		rdc = nil
	}
}

// redisExecute: execute the body with the redis connection.
// If the body returns an error, shutdown the server.
func redisExecute(body func() error) {
	// because redis connections can go away without warning, we
	// first ping to make sure the connection is alive, and
	// reconnect if not.
	redisPing := func() (err error) {
		_, err = rdc.Do("PING")
		if err != nil {
			log.Printf("PING failure with redis: %v", err)
			redisClose()
			err = redisConnect(rdUrl)
			if err != nil {
				log.Printf("Failed to reconnect to redis at %q", rdUrl)
			}
		}
		return
	}

	// grab the mutex and execute the body
	// if the ping or the body fails, shut down
	rdMutex.Lock()
	defer func(failed bool) {
		rdMutex.Unlock()
		if failed {
			shutdown(redisFailureShutdown)
		}
	}(redisPing() != nil || body() != nil)
}

// redisLookup: lookup a session for a sessionID
func (session *susenSession) redisLookup() (found bool) {
	body := func() error {
		puzzleID, err := redis.String(rdc.Do("GET", session.sessionID+puzzleIDKey))
		if puzzleID != "" {
			session.puzzleID = puzzleID
			found = true
			return nil
		}
		if err != redis.ErrNil {
			log.Printf("Redis error on GET of session %q puzzleID: %v",
				session.sessionID, err)
			return err
		}
		log.Printf("No redis saved state for session %q", session.sessionID)
		return nil
	}
	redisExecute(body)
	return
}

// redisInit: initialize the session in Redis
func (session *susenSession) redisInit() {
	body := func() (err error) {
		_, err = rdc.Do("SET", session.sessionID+puzzleIDKey, session.puzzleID)
		if err != nil {
			log.Printf("Redis error on SET of session %q puzzleID to %q: %v",
				session.sessionID, session.puzzleID, err)
			return err
		}
		_, err = rdc.Do("DEL", session.sessionID+stepsKey)
		if err != nil {
			log.Printf("Redis error on DELETE of session %q steps: %v",
				session.sessionID, err)
			return err
		}
		return nil
	}
	redisExecute(body)
}

// redisAddStep: add to the session in Redis
func (session *susenSession) redisAddStep(p puzzle.Puzzle) {
	vals := p.State().Values
	bytes, err := json.Marshal(vals)
	if err != nil {
		log.Printf("Failed to marshal %v as JSON: %v", vals, err)
		shutdown(runtimeFailureShutdown)
	}

	body := func() (err error) {
		_, err = rdc.Do("RPUSH", session.sessionID+stepsKey, bytes)
		if err != nil {
			log.Printf("Redis error on RPUSH of session %q steps: %v",
				session.sessionID, err)
		}
		return
	}
	redisExecute(body)
}

// redisUndoStep: add to the session in Redis
func (session *susenSession) redisUndoStep() {
	body := func() (err error) {
		_, err = rdc.Do("RPOP", session.sessionID+stepsKey)
		if err != nil {
			log.Printf("Redis error on RPOP of session %q steps: %v",
				session.sessionID, err)
		}
		return
	}
	redisExecute(body)
}

// redisLoad: load the session from Redis
func (session *susenSession) redisLoad() {
	// first load the first step
	vals, ok := puzzleValues[session.puzzleID]
	geo := vals[0] // save for steps, later
	if !ok {
		log.Printf("Failed to find values for puzzle %q", session.puzzleID)
		shutdown(runtimeFailureShutdown)
	}
	p, e := puzzle.New(vals)
	if e != nil {
		log.Printf("Failed to create puzzle %q: %v", session.puzzleID, e)
		shutdown(runtimeFailureShutdown)
	}
	session.steps = []puzzle.Puzzle{p}

	// now fetch any step values that were saved
	var steps []string
	body := func() (err error) {
		steps, err = redis.Strings(rdc.Do("LRANGE", session.sessionID+stepsKey, 0, -1))
		if err != nil {
			log.Printf("Redis error on LRANGE of session %q steps: %v",
				session.sessionID, err)
		}
		return
	}
	redisExecute(body)

	// add puzzle steps for each of the saved step values
	for _, step := range steps {
		var stepVals []int
		if e = json.Unmarshal([]byte(step), &stepVals); e != nil {
			log.Printf("JSON error on session %q step %v: %v", session.sessionID, step, e)
			shutdown(runtimeFailureShutdown)
		}
		vals = append([]int{geo}, stepVals...)
		p, e := puzzle.New(vals)
		if e != nil {
			log.Printf("Failed to create puzzle from %v: %v", vals, e)
			shutdown(runtimeFailureShutdown)
		}
		session.steps = append(session.steps, p)
	}
}

/*

coordinate shutdown across goroutines and top-level server

*/

type shutdownCause int

const (
	normalShutdown = iota
	runtimeFailureShutdown
	redisFailureShutdown
	caughtSignalShutdown
	listenerFailureShutdown
)

// shutdownLog: log process exit
func shutdown(reason shutdownCause) {
	// Get redis mutex and keep it to block other handlers from
	// running.  Close the redis connection.
	rdMutex.Lock()
	redisClose()

	// log reason for shutdown and exit
	switch reason {
	case normalShutdown:
		log.Fatal("Exiting: normal shutdown.")
	case runtimeFailureShutdown:
		log.Fatal("Exiting: runtime failure.")
	case caughtSignalShutdown:
		log.Fatal("Exiting: caught signal.")
	case listenerFailureShutdown:
		log.Fatal("Exiting: web server failed.")
	case redisFailureShutdown:
		log.Fatal("Exiting: redis failure.")
	default:
		log.Fatal("Exiting: unknown cause.")
	}
}

// shutdownOnSignal: catch signals and exit.
func shutdownOnSignal() {
	// based on example in os.signal godoc
	c := make(chan os.Signal, 1)
	signal.Notify(c) // die on all signals

	go func() {
		s := <-c
		log.Printf("Received OS-level signal: %v", s)
		shutdown(caughtSignalShutdown)
	}()
}
