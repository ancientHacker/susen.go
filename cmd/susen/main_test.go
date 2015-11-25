package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ancientHacker/susen.go/Godeps/_workspace/src/github.com/garyburd/redigo/redis"
	"github.com/ancientHacker/susen.go/puzzle"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

const (
	clientCount = 5
	runCount    = 3
)

type sessionClient struct {
	id       int           // which client this is
	client   *http.Client  // the http client, with cookies
	puzzleID string        // the puzzle this client works on
	interval int           // the interval, in msec, between calls
	vals     []int         // the expected values of the puzzle
	choice   puzzle.Choice // the first choice to try in this puzzle
}

func rdcConnect(t *testing.T) redis.Conn {
	rdc := redisConnect("redis://localhost:6379/0")
	if rdc == nil {
		t.Fatalf("Exiting: No local redis server available")
	}
	_, e := rdc.Do("FLUSHALL")
	if e != nil {
		t.Fatalf("Exiting: Failed to flush redis database: %v", e)
	}
	return rdc
}

func TestSessionSelect(t *testing.T) {
	rdc := rdcConnect(t)
	defer rdc.Close()

	// one server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := sessionSelect(rdc, w, r)
		t.Logf("Session %v handling %s %s.", session.sessionID, r.Method, r.URL.Path)
		session.rootHandler(w, r)
	}))
	defer srv.Close()

	// helper - select first assigned square as choice
	firstAssigned := func(pvals []int) puzzle.Choice {
		// first value is actually the geometry code, so 1-based indexing
		for i := 1; i < len(pvals); i++ {
			if v := pvals[i]; v != 0 {
				return puzzle.Choice{Index: i, Value: v}
			}
		}
		panic(fmt.Errorf("No assigned values!"))
	}
	// helper - log cookies
	logCookies := func(c *sessionClient, target string) {
		url, e := url.Parse(target)
		if e != nil {
			panic(e)
		}
		cookies := c.client.Jar.Cookies(url)
		if len(cookies) == 0 {
			t.Logf("Client %d: No target cookies.\n", c.id)
		} else if len(cookies) == 1 {
			t.Logf("Client %d: Target cookie: %v\n", c.id, *cookies[0])
		} else {
			t.Logf("Client %d: %d target cookies are:\n", c.id, len(cookies))
			for i, c := range cookies {
				t.Logf("\tcookie %d: %v\n", i, *c)
			}
		}
	}
	// helper - prevent redirects in a known way
	redirectCount := 0
	redirectFn := func(*http.Request, []*http.Request) error {
		redirectCount++
		return fmt.Errorf("%d", redirectCount)
	}
	// helper - make a call setting the current session puzzle, return false on error
	setPuzzle := func(c *sessionClient, puzzleID string) bool {
		target := fmt.Sprintf("%s/reset/%s", srv.URL, puzzleID)
		t.Logf("Client %d: getting %s", c.id, target)
		logCookies(c, target)
		r, e := c.client.Get(target)
		if e != nil && e.(*url.Error).Err.Error() != fmt.Sprintf("%d", redirectCount) {
			t.Errorf("client %d: Request error: %v", c.id, e)
			return false
		}
		t.Logf("client %d: %q\n", c.id, r.Status)
		t.Logf("client %d: %v\n", c.id, r.Header)
		if r.StatusCode != http.StatusFound {
			t.Errorf("client %d: Reset request did not return redirect status: %v",
				c.id, r.StatusCode)
			return false
		}
		if r.Header.Get("Location") != "/solver/" {
			t.Errorf("client %d: Reset request redirected to incorrect location: %v",
				c.id, r.Header.Get("Location"))
			return false
		}
		return true
	}
	// helper - make a squares-returning action call, return false on error
	getSquares := func(c *sessionClient, action string) bool {
		target := fmt.Sprintf("%s/api/%s", srv.URL, action)
		t.Logf("Client %d: getting %s", c.id, target)
		logCookies(c, target)
		r, e := c.client.Get(target)
		if e != nil {
			t.Errorf("client %d: Request error: %v", c.id, e)
			return false
		}
		t.Logf("client %d: %q\n", c.id, r.Status)
		t.Logf("client %d: %v\n", c.id, r.Header)
		b, e := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if e != nil {
			t.Errorf("client %d: Read error on puzzle response body: %v", c.id, e)
			return false
		}

		var s []puzzle.Square
		e = json.Unmarshal(b, &s)
		if e != nil {
			t.Errorf("client %d: Unmarshal failed: %v", c.id, e)
			return false
		}
		if len(s) != len(c.vals)-1 {
			t.Errorf("client %d: Got wrong number of squares: %d", c.id, len(s))
			return false
		}
		for i := 0; i < len(s); i++ {
			if s[i].Aval != c.vals[i+1] {
				t.Errorf("client %d: Square %d has value %d", c.id, s[i].Index, s[i].Aval)
				return false
			}
		}
		return true
	}
	// helper - make an update-returning action call, return false on error
	getUpdate := func(c *sessionClient) bool {
		t.Logf("Client %d: posting choice %v", c.id, c.choice)
		bs, e := json.Marshal(c.choice)
		if e != nil {
			t.Errorf("client %d: Failed to encode choice: %v", c.id, e)
			return false
		}
		target := fmt.Sprintf("%s/api/assign", srv.URL)
		t.Logf("Client %d: posting to %s", c.id, target)
		logCookies(c, target)
		r, e := c.client.Post(target, "application/json", bytes.NewReader(bs))
		if e != nil {
			t.Errorf("client %d: Request error: %v", c.id, e)
			return false
		}
		t.Logf("client %d: %q\n", c.id, r.Status)
		t.Logf("client %d: %v\n", c.id, r.Header)
		b, e := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if e != nil {
			t.Errorf("client %d: Read error on puzzle response body: %v", c.id, e)
			return false
		}

		if r.StatusCode != http.StatusBadRequest {
			t.Errorf("client %d: Bad assignment returned unexpected status: %d",
				c.id, r.StatusCode)
		} else {
			var err puzzle.Error
			e = json.Unmarshal(b, &err)
			if e != nil {
				t.Errorf("client %d: Unmarshal failed: %v", c.id, e)
				return false
			}
			if err.Condition != puzzle.DuplicateAssignmentCondition {
				t.Errorf("client %d: Got unexpected error: %v", c.id, err)
			}
		}
		return true
	}
	// helper - sleep interval milliseconds
	sleep := func(c *sessionClient) {
		sleeptime := time.Duration(c.interval) * time.Millisecond
		t.Logf("Client %d sleeps %s", c.id, sleeptime)
		time.Sleep(sleeptime)
	}

	// make clients
	clients := make([]*sessionClient, clientCount)
	for i := 0; i < clientCount; i++ {
		jar, e := cookiejar.New(nil)
		if e != nil {
			t.Fatalf("Failed to create cookie jar #%d: %v", i+1, e)
		}
		// try every key except the default "1-star"
		testKeys := []string{"2-star", "3-star", "4-star", "5-star", "6-star"}
		keyIndex := i % len(testKeys)
		puzzleID := testKeys[keyIndex]
		puzzleVals := puzzleValues[puzzleID]
		clients[i] = &sessionClient{
			id:       i + 1,
			client:   &http.Client{Jar: jar, CheckRedirect: redirectFn},
			puzzleID: puzzleID,
			interval: (i*17)%100 + 100,
			vals:     puzzleVals,
			choice:   firstAssigned(puzzleVals),
		}
		t.Logf("Client %d: %+v\n", clients[i].id, *clients[i])
	}

	// each client makes runCount sets of 3 calls: reset then assign then back
	// after runCount sets, the client reports back, and we wait for all clients
	ch := make(chan int, clientCount)
	start := time.Now()
	for i := 0; i < clientCount; i++ {
		go func(client *sessionClient) {
			for i := 0; i < runCount; i++ {
				sleep(client)
				if !setPuzzle(client, client.puzzleID) {
					break
				}
				if !getSquares(client, "/") {
					break
				}
				sleep(client)
				if !getUpdate(client) {
					break
				}
				sleep(client)
				if !getSquares(client, fmt.Sprintf("/back/")) {
					break
				}
			}
			ch <- client.id
		}(clients[i])
	}
	for i := 0; i < clientCount; i++ {
		id := <-ch
		diff := time.Now().Sub(start)
		t.Logf("Client %d finished in %v\n", id, diff)
	}
	if len(sessions) != clientCount {
		t.Errorf("At end of run, there were %d sessions: %v", len(sessions), sessions)
	}
}

func TestIssue1(t *testing.T) {
	rdc := rdcConnect(t)
	defer rdc.Close()

	// helper - log cookies
	logCookies := func(jar http.CookieJar, target string) {
		url, e := url.Parse(target)
		if e != nil {
			panic(e)
		}
		cookies := jar.Cookies(url)
		if len(cookies) == 0 {
			t.Logf("No target cookies.\n")
		} else if len(cookies) == 1 {
			t.Logf("Target cookie: %v\n", *cookies[0])
		} else {
			t.Logf("%d target cookies are:\n", len(cookies))
			for i, c := range cookies {
				t.Logf("\tcookie %d: %v\n", i, *c)
			}
		}
	}

	// server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := sessionSelect(rdc, w, r)
		t.Logf("Session %v handling %s %s.", session.sessionID, r.Method, r.URL.Path)
		http.Error(w, "This is a test", http.StatusOK)
	}))
	defer srv.Close()
	target := srv.URL

	// client
	jar, e := cookiejar.New(nil)
	if e != nil {
		t.Fatalf("Failed to create cookie jar: %v", e)
	}
	c := http.Client{Jar: jar}

	// for each heroku protocol indicator, do two pairs of
	// requests, one to get the cookie set, one to use it.  We
	// also handle the case where there is no heroku protocol
	// indicator, which is a bit of overkill, since no server
	// should get both Heroku and non-Heroku requests, but you
	// never know :).
	for i, herokuProtocol := range []string{"", "http", "https"} {
		for j, expectSetCookie := range []bool{true, false} {
			req, e := http.NewRequest("GET", target, nil)
			if e != nil {
				t.Fatalf("Failed to create request %d: %v", 2*i+j, e)
			}
			if herokuProtocol != "" {
				req.Header.Add("X-Forwarded-Proto", herokuProtocol)
			}
			t.Logf("Created request %d: herokuProtocol = %q", 2*i+j, herokuProtocol)
			logCookies(c.Jar, target)
			r, e := c.Do(req)
			if e != nil {
				t.Fatalf("Request error: %v", e)
			}
			t.Logf("request 1: %q\n", r.Status)
			t.Logf("request 1: %v\n", r.Header)
			r.Body.Close()
			if expectSetCookie {
				if h := r.Header.Get("Set-Cookie"); h == "" {
					t.Errorf("No Set-Cookie received on request %d.", 2*i+j)
				}
			} else {
				if h := r.Header.Get("Set-Cookie"); h != "" {
					t.Errorf("Set-Cookie received on request %d.", 2*i+j)
				}
			}
		}
	}

	// now make sure the protocol cookies are set for the next round
	for i, herokuProtocol := range []string{"", "http", "https"} {
		for j, expectSetCookie := range []bool{false, false} {
			req, e := http.NewRequest("GET", target, nil)
			if e != nil {
				t.Fatalf("Failed to create request %d: %v", 2*i+j, e)
			}
			if herokuProtocol != "" {
				req.Header.Add("X-Forwarded-Proto", herokuProtocol)
			}
			t.Logf("Created request %d: herokuProtocol = %q", 2*i+j, herokuProtocol)
			logCookies(c.Jar, target)
			r, e := c.Do(req)
			if e != nil {
				t.Fatalf("Request error: %v", e)
			}
			t.Logf("request 1: %q\n", r.Status)
			t.Logf("request 1: %v\n", r.Header)
			r.Body.Close()
			if expectSetCookie {
				if h := r.Header.Get("Set-Cookie"); h == "" {
					t.Errorf("No Set-Cookie received on request %d.", 2*i+j)
				}
			} else {
				if h := r.Header.Get("Set-Cookie"); h != "" {
					t.Errorf("Set-Cookie received on request %d.", 2*i+j)
				}
			}
		}
	}
}

func TestIssue11(t *testing.T) {
	rdc := rdcConnect(t)
	defer rdc.Close()

	// helper - log cookies
	logCookies := func(jar http.CookieJar, target string) {
		url, e := url.Parse(target)
		if e != nil {
			panic(e)
		}
		cookies := jar.Cookies(url)
		if len(cookies) == 0 {
			t.Logf("No target cookies.\n")
		} else if len(cookies) == 1 {
			t.Logf("Target cookie: %v\n", *cookies[0])
		} else {
			t.Logf("%d target cookies are:\n", len(cookies))
			for i, c := range cookies {
				t.Logf("\tcookie %d: %v\n", i, *c)
			}
		}
	}

	// add puzzle and appropriate assignments for testing
	puzzleValues["test11"] = []int{0,
		1, 0, 3, 0,
		0, 3, 0, 1,
		3, 0, 1, 0,
		0, 1, 0, 3,
	}
	defer func() { delete(puzzleValues, "test11") }()
	choices := []puzzle.Choice{{13, 2}, {10, 4}, {15, 4}}

	// server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Handling %s %s...", r.Method, r.URL.Path)
		session := sessionSelect(rdc, w, r)
		if strings.HasPrefix(r.URL.Path, "/ClearMemorySession/") {
			sessionMutex.Lock()
			delete(sessions, session.sessionID)
			sessionMutex.Unlock()
			t.Logf("Deleted session %q from memory", session.sessionID)
			http.Error(w, session.sessionID, http.StatusOK)
			return
		}
		session.rootHandler(w, r)
		t.Logf("There are %d steps in session %q", len(session.steps), session.sessionID)
		t.Logf("The puzzleID of session %q is %q", session.sessionID, session.puzzleID)
	}))
	defer srv.Close()

	// client
	jar, e := cookiejar.New(nil)
	if e != nil {
		t.Fatalf("Failed to create cookie jar: %v", e)
	}
	c := http.Client{Jar: jar}

	// set the puzzle
	r, e := c.Get(srv.URL + "/reset/test11")
	if e != nil || r.StatusCode != http.StatusOK {
		t.Fatalf("Request error on /reset/test11: %v", e)
	}
	logCookies(c.Jar, srv.URL)

	// do the assignments
	for i, choice := range choices {
		b, e := json.Marshal(choice)
		if e != nil {
			t.Fatalf("Case %d: Failed to encode choice: %v", i, e)
		}
		t.Logf("%s\n", b)
		r, e := c.Post(srv.URL+"/api/",
			"application/json", strings.NewReader(string(b)))
		if e != nil {
			t.Fatalf("assignment %d: Request error: %v", i, e)
		}
		t.Logf("%q\n", r.Status)
		t.Logf("%v\n", r.Header)
		if r.StatusCode != http.StatusOK {
			t.Errorf("case %d: Status was %v, expected %v", i, r.StatusCode, http.StatusOK)
		}
		body, e := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if e != nil {
			t.Fatalf("test %d: Read error on update: %v", i, e)
		}
		t.Logf("%s\n", string(body))
	}

	// read the puzzle contents
	logCookies(c.Jar, srv.URL)
	r, e = c.Get(srv.URL + "/api/")
	if e != nil {
		t.Fatalf("Squares before request error: %v", e)
	}
	t.Logf("%q\n", r.Status)
	t.Logf("%v\n", r.Header)
	if r.StatusCode != http.StatusOK {
		t.Errorf("Squares before status was %v not %v", r.StatusCode, http.StatusOK)
	}
	squaresBefore, e := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if e != nil {
		t.Fatalf("Read error on squares before: %v", e)
	}
	t.Logf("%s\n", string(squaresBefore))

	// clear the in-memory session cache
	logCookies(c.Jar, srv.URL)
	r, e = c.Get(srv.URL + "/ClearMemorySession/")
	if e != nil {
		t.Fatalf("Clear Memory Session Request error: %v", e)
	}
	t.Logf("%q\n", r.Status)
	t.Logf("%v\n", r.Header)
	if r.StatusCode != http.StatusOK {
		t.Errorf("Clear Memory Session status was %v not %v", r.StatusCode, http.StatusOK)
	}

	// read the puzzle contents again, compare
	logCookies(c.Jar, srv.URL)
	r, e = c.Get(srv.URL + "/api/")
	if e != nil {
		t.Fatalf("Squares after request error: %v", e)
	}
	t.Logf("%q\n", r.Status)
	t.Logf("%v\n", r.Header)
	if r.StatusCode != http.StatusOK {
		t.Errorf("Squares after status was %v not %v", r.StatusCode, http.StatusOK)
	}
	squaresAfter, e := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if e != nil {
		t.Fatalf("Read error on squares after: %v", e)
	}
	t.Logf("%s\n", string(squaresAfter))
	if string(squaresAfter) != string(squaresBefore) {
		t.Errorf("Squares don't match before and after.")
	}

	// now go back two steps
	logCookies(c.Jar, srv.URL)
	r, e = c.Get(srv.URL + "/api/back/")
	if e != nil {
		t.Fatalf("Go Back 1 error: %v", e)
	}
	t.Logf("%q\n", r.Status)
	t.Logf("%v\n", r.Header)
	if r.StatusCode != http.StatusOK {
		t.Errorf("Go Back 1 status was %v not %v", r.StatusCode, http.StatusOK)
	}
	r, e = c.Get(srv.URL + "/api/back/")
	if e != nil {
		t.Fatalf("Go Back 2 error: %v", e)
	}
	t.Logf("%q\n", r.Status)
	t.Logf("%v\n", r.Header)
	if r.StatusCode != http.StatusOK {
		t.Errorf("Go Back 2 status was %v not %v", r.StatusCode, http.StatusOK)
	}

	// read the puzzle contents
	logCookies(c.Jar, srv.URL)
	r, e = c.Get(srv.URL + "/api/")
	if e != nil {
		t.Fatalf("Squares before request error: %v", e)
	}
	t.Logf("%q\n", r.Status)
	t.Logf("%v\n", r.Header)
	if r.StatusCode != http.StatusOK {
		t.Errorf("Squares before status was %v not %v", r.StatusCode, http.StatusOK)
	}
	squaresBefore, e = ioutil.ReadAll(r.Body)
	r.Body.Close()
	if e != nil {
		t.Fatalf("Read error on squares before: %v", e)
	}
	t.Logf("%s\n", string(squaresBefore))

	// clear the in-memory session cache
	logCookies(c.Jar, srv.URL)
	r, e = c.Get(srv.URL + "/ClearMemorySession/")
	if e != nil {
		t.Fatalf("Clear Memory Session Request error: %v", e)
	}
	t.Logf("%q\n", r.Status)
	t.Logf("%v\n", r.Header)
	if r.StatusCode != http.StatusOK {
		t.Errorf("Clear Memory Session status was %v not %v", r.StatusCode, http.StatusOK)
	}

	// read the puzzle contents again, compare
	logCookies(c.Jar, srv.URL)
	r, e = c.Get(srv.URL + "/api/")
	if e != nil {
		t.Fatalf("Squares after request error: %v", e)
	}
	t.Logf("%q\n", r.Status)
	t.Logf("%v\n", r.Header)
	if r.StatusCode != http.StatusOK {
		t.Errorf("Squares after status was %v not %v", r.StatusCode, http.StatusOK)
	}
	squaresAfter, e = ioutil.ReadAll(r.Body)
	r.Body.Close()
	if e != nil {
		t.Fatalf("Read error on squares after: %v", e)
	}
	t.Logf("%s\n", string(squaresAfter))
	if string(squaresAfter) != string(squaresBefore) {
		t.Errorf("Squares don't match before and after.")
	}
}
