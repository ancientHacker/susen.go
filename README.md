# susen &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;[![Build Status](https://travis-ci.org/ancientHacker/susen.go.svg)](https://travis-ci.org/ancientHacker/susen.go)

susen is a simple Sudoku game built as a learning experience in
Sudoku, golang, and web services.  The name susen is an (English)
contraction of "sudoku sensei" (sudoku teacher).

## Usage

To try susen out on your local system, set up a go 1.5 (or higher) environment, and do:

	go get -u github.com/ancientHacker/susen.go/
	cd $GOPATH/src/github.com/ancientHacker/susen.go
	$GOPATH/bin/susen

Then open your browser to [localhost:8080](http://localhost:8080) and you're there.

## CI/CD

Thanks to the wonderful people at Travis and Heroku, susen
has automated CI/CD on the master branch.  You can run the
latest build at [susen-staging.herokuapp.com](https://susen-staging.herokuapp.com).

## License

Copyright © 2013-2015 Daniel C Brotsky.  Licensed under GPLv2.
See the license file for details.
