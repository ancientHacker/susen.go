# Sūsen (数千)

[![Build Status](https://travis-ci.org/ancientHacker/susen.go.svg)](https://travis-ci.org/ancientHacker/susen.go)

Sūsen is a simple Sudoku game built as a learning experience in
Sūdoku, golang, and web services.  The name Sūsen (数千), which
means _Thousands_, is a pun: it _reads_ the same as the (English)
contraction of Sūdoku no Sensei (数独の先生), which means _Sūdoku
teacher_, and it _refers_ to the thousands of tries one makes
when programming or solving a Sūdoku puzzle.  (The actual
Japanese "contraction" 数先 is not a word in standard Japanese,
and so would not be read as Sūsen but as Kazu-saki.  Its meaning
would be closer to _number destination_.)

## Usage

To give Sūsen a try on your local system, set up a go 1.4 (or higher) environment, and do:

	go get -u github.com/ancientHacker/susen.go/
	cd $GOPATH/src/github.com/ancientHacker/susen.go
	$GOPATH/bin/susen

Then open your browser to <http://localhost:8080> and you're there.

## CI/CD

Thanks to the wonderful people at Travis and Heroku, Sūsen
has automated CI/CD on the master branch.  You can run the
latest build at <https://susen-staging.herokuapp.com>.

## License

Copyright © 2013-2015 Daniel C Brotsky.  Licensed under GPLv2.
See the LICENSE.md file for details.
