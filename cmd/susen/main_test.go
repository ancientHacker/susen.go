package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ancientHacker/susen.go/puzzle"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

const (
	clientCount = 10
	runCount    = 20
)

type sessionClient struct {
	id       int           // which client this is
	client   *http.Client  // the http client, with cookies
	puzzleID int           // the puzzle this client works on
	interval int           // the interval, in msec, between calls
	vals     []int         // the expected values of the puzzle
	choice   puzzle.Choice // the first choice to try in this puzzle
}

func TestSessionSelect(t *testing.T) {
	// one server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := sessionSelect(w, r)
		t.Logf("Session %v handling %s %s.", session.id, r.Method, r.URL.Path)
		session.apiHandler(w, r)
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
	// helper - make a squares-returning action call, return false on fatal error
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
		/*
			for i := 0; i < len(s); i++ {
				if s[i].Aval != c.vals[i+1] {
					t.Errorf("client %d: Square %d has value %d", c.id, s[i].Index, s[i].Aval)
				}
			}
		*/
		return true
	}
	// helper - make an update-returning action call, return false on fatal error
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

	// ten clients
	clients := make([]*sessionClient, clientCount)
	for i := 0; i < clientCount; i++ {
		jar, e := cookiejar.New(nil)
		if e != nil {
			t.Fatalf("Failed to create cookie jar #%d: %v", i+1, e)
		}
		// puzzleID is 1 + puzzleValues index, but we avoid 0 (the defaultIndex)
		puzzleID := 2 + (i % (len(puzzleValues) - 2))
		puzzleVals := puzzleValues[puzzleID-1]
		clients[i] = &sessionClient{
			id:       i + 1,
			client:   &http.Client{Jar: jar},
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
				if !getSquares(client, fmt.Sprintf("/reset/%d", client.puzzleID)) {
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
