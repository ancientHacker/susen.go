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
	"os"
	"sync"
)

func Connect() (cacheId, databaseId string, err error) {
	// make sure the database is initialized
	if err = dbprep.EnsureData(); err != nil {
		err = fmt.Errorf("Couldn't initialize database: %v", err)
		return
	}

	rdInit()
	rdMutex.Lock()
	defer rdMutex.Unlock()
	cacheId, err = rdConnect()
	if err != nil {
		return
	}

	pgInit()
	databaseId, err = pgConnect()
	if err != nil {
		return
	}
	return
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
	rdMutex sync.Mutex // prevent concurrent connection use
)

// rdInit - look up Redis info from the environment
func rdInit() {
	url := os.Getenv("REDISTOGO_URL")
	if url == "" {
		rdUrl = "redis://localhost:6379/"
	} else {
		rdUrl = url
	}
}

// rdConnect: connect to the given Redis URL.  Returns the
// connection id, if successful, an error otherwise.
func rdConnect() (string, error) {
	conn, err := redis.DialURL(rdUrl)
	if err != nil {
		err = fmt.Errorf("Couldn't connect to cache at %q: %v", rdUrl, err)
		return "", err
	}
	rdc = conn
	return rdUrl, nil
}

// rdClose: close the given Redis connection.
func rdClose() {
	if rdc != nil {
		rdc.Close()
		rdc = nil
	}
}

// rdExecute: execute the body inside a Redis "transaction"
// (i.e., with the Redis mutex and connection).  Meant to be used
// inside a handler, because errors in execution will panic back
// to package entry level.
func rdExecute(body func(tx redis.Conn) error) {
	// wrap the body against runtime and database failures
	wrapper := func(tx redis.Conn) (err error) {
		defer func() {
			if r := recover(); r != nil {
				if e, ok := r.(error); ok {
					err = e
				} else {
					err = fmt.Errorf("Caught panic during rdExecute: %v", err)
				}
			}
		}()
		// Because Redis connections can go away without warning,
		// we ping to make sure the connection is alive, and try
		// to reconnect if not.
		if _, err := rdc.Do("PING"); err != nil {
			rdClose()
			_, err = rdConnect()
			if err != nil {
				err = fmt.Errorf("Failed to reconnect to cache at %q", rdUrl)
				return err
			}
		}
		// connection is good; run the body
		return body(tx)
	}
	// grab the mutex and execute the body
	rdMutex.Lock()
	defer func(err error) {
		rdMutex.Unlock()
		if err != nil {
			panic(err)
		}
	}(wrapper(rdc))
}

/*

persistence using Postgres

*/

// Postgres connection data
var (
	pgConn *pgx.Conn // open database, if any
	pgUrl  string    // URL for the open connection
)

// pgInit - look up Redis info from the environment
func pgInit() {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		pgUrl = "postgres://localhost/susen?sslmode=disable"
	} else {
		pgUrl = url
	}
}

// pgConnect: Open the Postgres database.  Returns any error
// encountered during the open.
func pgConnect() (string, error) {
	cfg, err := pgx.ParseURI(pgUrl)
	if err != nil {
		err = fmt.Errorf("Parse failure on Postgres URI %q: %v", pgUrl, err)
		return "", err
	}
	conn, err := pgx.Connect(cfg)
	if err != nil {
		err = fmt.Errorf("Couldn't connect to db at %q: %v", pgUrl, err)
		return "", err
	}
	pgConn = conn
	return pgUrl, nil
}

// pgClose: close the given Postgres connection.
func pgClose() {
	if pgConn != nil {
		pgConn.Close()
		pgConn = nil
	}
}

// pgExecute: execute the body inside a single transaction.
// Meant to be used inside a handler, because errors in execution
// will panic back to the package entry level.  If the body errs
// out, then the transaction is rolled back, otherwise it's
// committed.
func pgExecute(body func(tx *pgx.Tx) error) {
	// wrap the body against runtime and database failures
	wrapper := func(tx *pgx.Tx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				if e, ok := r.(error); ok {
					err = e
				} else {
					err = fmt.Errorf("Caught panic during pgExecute: %v", r)
				}
			}
		}()
		return body(tx)
	}
	// get the transaction
	tx, err := pgConn.Begin()
	if err != nil {
		panic(fmt.Errorf("Can't open a transaction against database: %v", err))
	}
	// execute the body in the transaction
	defer func(err error) {
		if err != nil {
			tx.Rollback()
			panic(err)
		}
		tx.Commit()
	}(wrapper(tx))
}
