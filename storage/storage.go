// susen.go - a web-based Sudoku game and teaching tool.
// Copyright (C) 2015-2016 Daniel C. Brotsky.
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program; if not, write to the Free Software Foundation, Inc.,
// 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA.
// Licensed under the LGPL v3.  See the LICENSE file for details

package storage

import (
	"fmt"
	"github.com/ancientHacker/susen.go/Godeps/_workspace/src/github.com/garyburd/redigo/redis"
	"github.com/ancientHacker/susen.go/Godeps/_workspace/src/github.com/jackc/pgx"
	"github.com/ancientHacker/susen.go/dbprep"
	"log"
	"os"
	"sync"
)

func Connect() error {
	// make sure the database is initialized
	if err := dbprep.EnsureData(); err != nil {
		return err
	}

	rdInit()
	rdMutex.Lock()
	defer rdMutex.Unlock()
	if err := rdConnect(); err != nil {
		return err
	}

	pgInit()
	if err := pgConnect(); err != nil {
		return err
	}
	return nil
}

func Close() {
	rdMutex.Lock()
	defer rdMutex.Unlock()
	pgClose()
	rdClose()
}

/*

cache using Redis

*/

// Redis connection data
var (
	rdc     redis.Conn // open connection, if any
	rdUrl   string     // URL for the open connection
	rdEnv   string     // environment key prefix
	rdMutex sync.Mutex // prevent concurrent connection use
)

// rdInit - look up Redis info from the environment
func rdInit() {
	url := os.Getenv("REDISTOGO_URL")
	env := os.Getenv("APPLICATION_ENV")
	if url == "" {
		rdUrl = "redis://localhost:6379/"
	} else {
		rdUrl = url
	}
	if env == "" {
		if url == "" {
			rdEnv = "local"
		} else {
			rdEnv = "dev"
		}
	} else {
		rdEnv = env
	}
}

// rdConnect: connect to the given Redis URL.  Returns the
// connection, if successful, nil otherwise.
func rdConnect() error {
	conn, err := redis.DialURL(rdUrl)
	if err == nil {
		log.Printf("Connected to Redis at %q (env: %q)", rdUrl, rdEnv)
		rdc = conn
		return nil
	}
	log.Printf("Can't connect to Redis server at %q", rdUrl)
	return err
}

// rdClose: close the given Redis connection.
func rdClose() {
	if rdc != nil {
		rdc.Close()
		log.Print("Closed connection to Redis.")
		rdc = nil
	}
}

// rdExecute: execute the body with the Redis connection.
// Meant to be used inside a handler, because errors in execution
// will panic back to the handler level.
func rdExecute(body func() error) {
	// wrap the body against runtime and database failures
	wrapper := func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				if e, ok := r.(error); ok {
					err = e
				} else {
					err = fmt.Errorf("%v", r)
				}
				log.Printf("Caught panic during rdExecute: %v", err)
			}
		}()
		// Because Redis connections can go away without warning,
		// we ping to make sure the connection is alive, and try
		// to reconnect if not.
		if _, err := rdc.Do("PING"); err != nil {
			log.Printf("PING failure with Redis: %v", err)
			rdClose()
			err = rdConnect()
			if err != nil {
				log.Printf("Failed to reconnect to Redis at %q", rdUrl)
				return err
			}
		}
		// connection is good; run the body
		return body()
	}
	// grab the mutex and execute the body
	rdMutex.Lock()
	defer func(err error) {
		rdMutex.Unlock()
		if err != nil {
			panic(err)
		}
	}(wrapper())
}

/*

persistence using Postgres

*/

// Postgres connection data
var (
	pgConn *pgx.Conn // open database, if any
	pgUrl  string    // URL for the open connection
	pgEnv  string    // environment key (present in all tables)
)

// pgInit - look up Redis info from the environment
func pgInit() {
	url := os.Getenv("DATABASE_URL")
	env := os.Getenv("APPLICATION_ENV")
	if url == "" {
		pgUrl = "postgres://localhost/susen?sslmode=disable"
	} else {
		pgUrl = url
	}
	if env == "" {
		if url == "" {
			pgEnv = "local"
		} else {
			pgEnv = "dev"
		}
	} else {
		pgEnv = env
	}
}

// pgConnect: Open the Postgres database.  Returns any error
// encountered during the open.
func pgConnect() error {
	cfg, err := pgx.ParseURI(pgUrl)
	if err != nil {
		log.Printf("Parse failure on Postgres URI %v: %v", pgUrl, err)
		return nil
	}
	conn, err := pgx.Connect(cfg)
	if err == nil {
		log.Printf("Connected to Postgres at %q (env: %q)", pgUrl, pgEnv)
		pgConn = conn
		return nil
	}
	log.Printf("Can't connect to Postgres server at %q", pgUrl)
	return err
}

// pgClose: close the given Postgres connection.
func pgClose() {
	if pgConn != nil {
		pgConn.Close()
		log.Print("Closed connection to Postgres.")
		pgConn = nil
	}
}

// pgExecute: execute the body with the Postgres connection.
// Meant to be used inside a handler, because errors in execution
// will panic back to the handler level.
func pgExecute(body func() error) {
	// wrap the body against runtime and database failures
	wrapper := func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				if e, ok := r.(error); ok {
					err = e
				} else {
					err = fmt.Errorf("%v", r)
				}
				log.Printf("Caught panic during pgExecute: %v", err)
			}
		}()
		return body()
	}
	// execute the body
	defer func(err error) {
		if err != nil {
			panic(err)
		}
	}(wrapper())
}
