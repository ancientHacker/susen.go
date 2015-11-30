package main

import (
	"encoding/json"
	"github.com/ancientHacker/susen.go/Godeps/_workspace/src/github.com/garyburd/redigo/redis"
	"github.com/ancientHacker/susen.go/client"
	"github.com/ancientHacker/susen.go/puzzle"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	cookieNameBase = "susenID"
	cookiePath     = "/"
	cookieMaxAge   = 3600 * 24 * 7 * 52 // 1 year
	puzzleIDKey    = ":puzzleID"
	stepsKey       = ":steps"
)

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
	rdMutex         sync.Mutex
	rdUrl           string
	rdc             redis.Conn
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
	proto := "httpx" // absent other indicators, protocol is unknown

	// Issue #1: Heroku-transported protocols are specified in a header
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
		log.Fatalf("Failed to create puzzle %q: %v", puzzleID, e)
	}
	session.steps = []puzzle.Puzzle{p}
	session.redisInit()
	log.Printf("Initialized session %v from puzzle %q.", session.sessionID, session.puzzleID)
}

func (session *susenSession) addStep(next puzzle.Puzzle) {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()
	session.steps = append(session.steps, next)
	session.redisAddStep(next)
	log.Printf("Added session %v step %d.", session.sessionID, len(session.steps))
}

func (session *susenSession) undoStep() {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()
	if len(session.steps) > 1 {
		session.steps[len(session.steps)-1] = nil // release current step
		session.steps = session.steps[:len(session.steps)-1]
		session.redisUndoStep()
		log.Printf("Reverted session %v to step %d.", session.sessionID, len(session.steps))
	}
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
	case "POST":
		next := session.steps[len(session.steps)-1].Copy()
		_, e := puzzle.AssignHandler(next, w, r)
		if e != nil {
			log.Printf("Assign failed, returned error, no session change.")
		} else {
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

func (session *susenSession) rootHandler(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasPrefix(r.URL.Path, "/api/"):
		session.apiHandler(w, r)
		return
	case strings.HasPrefix(r.URL.Path, "/solver/"):
		session.solverHandler(w, r)
		return
	}
	http.Redirect(w, r, "/solver/", http.StatusFound)
}

func serveHttp(w http.ResponseWriter, r *http.Request) {
	if client.StaticHandler(w, r) {
		return
	}
	session := sessionSelect(w, r)
	session.rootHandler(w, r)
}

func main() {
	// establish redis connection
	url := redisUrl()
	if err := redisConnect(url); err != nil {
		log.Fatalf("Exiting: No redis server at %q: %v", url, err)
	}
	defer redisClose()

	// port sensing
	port := os.Getenv("PORT")
	if port == "" {
		// running locally in dev mode
		port = "localhost:8080"
	} else {
		// running as a true server
		port = ":" + port
	}

	// serve
	log.Printf("Listening on %s...", port)
	http.HandleFunc("/", serveHttp)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal("Listener failure: ", err)
	}
}

func redisUrl() string {
	url := os.Getenv("REDISTOGO_URL")
	if url == "" {
		url = "redis://localhost:6379/0"
	} else {
		url = url + "0"
	}
	return url
}

// redisConnect: connect to the given Redis URL.  Returns the
// connection, if successful, nil otherwise.
func redisConnect(url string) error {
	rdMutex.Lock()
	defer rdMutex.Unlock()

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
	rdMutex.Lock()
	defer rdMutex.Unlock()

	rdc.Close()
}

// redisPing: make sure the redis connection is up
// you MUST have the mutex when you call this!
func (session *susenSession) redisPing() {
	if _, err := rdc.Do("PING"); err != nil {
		log.Printf("Lost connection to redis: %v", err)
		rdc.Close()
		rdc, err = redis.DialURL(rdUrl)
		if err != nil {
			log.Fatalf("Reconnect failure: %v", err)
		} else {
			log.Printf("Reconnected to redis at %q", rdUrl)
		}
	}
}

// redisLookup: lookup a session for a sessionID
func (session *susenSession) redisLookup() bool {
	rdMutex.Lock()
	defer rdMutex.Unlock()
	session.redisPing()
	puzzleID, err := redis.String(rdc.Do("GET", session.sessionID+puzzleIDKey))
	if puzzleID != "" {
		session.puzzleID = puzzleID
		return true
	}
	if err != redis.ErrNil {
		log.Fatalf("Redis error on GET of session %q puzzleID: %v", session.sessionID, err)
	}
	return false
}

// redisInit: initialize the session in Redis
func (session *susenSession) redisInit() {
	rdMutex.Lock()
	defer rdMutex.Unlock()
	session.redisPing()
	_, err := rdc.Do("SET", session.sessionID+puzzleIDKey, session.puzzleID)
	if err != nil {
		log.Fatalf("Redis error on SET of session %q puzzleID to %q: %v",
			session.sessionID, session.puzzleID, err)
	}
	_, err = rdc.Do("DEL", session.sessionID+stepsKey)
	if err != nil {
		log.Fatalf("Redis error on DELETE of session %q steps: %v", session.sessionID, err)
	}
}

// redisAddStep: add to the session in Redis
func (session *susenSession) redisAddStep(p puzzle.Puzzle) {
	rdMutex.Lock()
	defer rdMutex.Unlock()
	session.redisPing()
	vals := p.State().Values
	bytes, err := json.Marshal(vals)
	if err != nil {
		log.Fatalf("Failed to marshal %v as JSON: %v", vals, err)
	}
	_, err = rdc.Do("RPUSH", session.sessionID+stepsKey, bytes)
	if err != nil {
		log.Fatalf("Redis error on RPUSH of session %q steps: %v", session.sessionID, err)
	}
}

// redisUndoStep: add to the session in Redis
func (session *susenSession) redisUndoStep() {
	rdMutex.Lock()
	defer rdMutex.Unlock()
	session.redisPing()
	_, err := rdc.Do("RPOP", session.sessionID+stepsKey)
	if err != nil {
		log.Fatalf("Redis error on RPOP of session %q steps: %v", session.sessionID, err)
	}
}

// redisLoad: load the session from Redis
func (session *susenSession) redisLoad() {
	rdMutex.Lock()
	defer rdMutex.Unlock()
	session.redisPing()

	// first load the first step
	vals, ok := puzzleValues[session.puzzleID]
	geo := vals[0] // save for steps, later
	if !ok {
		log.Fatalf("Failed to find values for puzzle %q", session.puzzleID)
	}
	p, e := puzzle.New(vals)
	if e != nil {
		log.Fatalf("Failed to create puzzle %q: %v", session.puzzleID, e)
	}
	session.steps = []puzzle.Puzzle{p}

	// now add any steps that were saved
	steps, err := redis.Strings(rdc.Do("LRANGE", session.sessionID+stepsKey, 0, -1))
	if err != nil {
		log.Fatalf("Redis error on LRANGE of session %q steps: %v", session.sessionID, err)
	}
	for _, step := range steps {
		var stepVals []int
		if e = json.Unmarshal([]byte(step), &stepVals); e != nil {
			log.Fatalf("JSON error on session %q step %v: %v", session.sessionID, step, e)
		}
		vals = append([]int{geo}, stepVals...)
		p, e := puzzle.New(vals)
		if e != nil {
			log.Fatalf("Failed to create puzzle from %v: %v", vals, e)
		}
		session.steps = append(session.steps, p)
	}
}
