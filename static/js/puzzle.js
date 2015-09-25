var hoverHints;
var selectHints;
var guessHints;
var puzzleContent = null;
var guessContent = null;
var puzzleSideLength = 9;
var squaresURL = "http://localhost:8080/api/squares/";
var assignURL = "http://localhost:8080/api/assign/";
var backURL = "http://localhost:8080/api/back/";
var resetURL = "http://localhost:8080/api/reset/";

function receivePuzzleSquares() {
    if (this.readyState == 4) {
	console.log("Got puzzle squares:", this.responseText);
        var squares = JSON.parse(this.responseText);
	fillPuzzle(squares);
	setFeedback("Puzzle received.");
	selectCell(null)
    }
}

var getPuzzleRequest = new XMLHttpRequest();
getPuzzleRequest.onreadystatechange = receivePuzzleSquares;

function receivePuzzleUpdate() {
    if (this.readyState == 4) {
	selectCell(null)
	console.log("Got puzzle update:", this.responseText);
        var result = JSON.parse(this.responseText);
	if (this.status == 200) {
	    updatePuzzle(result.squares);
	    if ("conflict" in result) {
		errors = result.conflict
		messages = ""
		for (i = 0; i < result.conflict.length; i++) {
		    messages += "<br />" + result.conflict[i].message
		}
		setFeedback("Assign produced errors; puzzle not solvable:" + messages);
	    } else {
		setFeedback("Assign successful; puzzle updated.");
	    }
	} else {
	    setFeedback("Assign failed: " + result.message);
	}
    }
}

var postAssignRequest = new XMLHttpRequest();
postAssignRequest.onreadystatechange = receivePuzzleUpdate;

function LoadPuzzle(url) {
    if (!url) {
	url = squaresURL;
    }
    console.log("GET request for", url);
    getPuzzleRequest.open("GET", url, true);
    getPuzzleRequest.send(null);
}

function AssignPuzzle(cell, val) {
    var choice = {index: cell, value: val};
    var body = JSON.stringify(choice);
    console.log("POST request to puzzle:", body);
    postAssignRequest.open("POST", assignURL, true);
    postAssignRequest.setRequestHeader("Content-type", "application/json");
    postAssignRequest.send(body);
}

function fillPuzzle(squares) {
    if (squares)
	puzzleContent = squares;
    else
	puzzleContent = null;
    refillPuzzle();
    fillGuess();
};

function updatePuzzle(squares) {
    if (squares && puzzleContent) {
	for (i = 0; i < squares.length; i++) {
	    if (squares[i].index > 0 && squares[i].index < puzzleContent.length) {
		puzzleContent[squares[i].index-1] = squares[i]
	    }
	}
    }
    refillPuzzle();
    fillGuess();
};

function refillPuzzle() {
    if (puzzleContent) {
	for (i = 0; i < puzzleContent.length; i++) {
	    var idstr = "c" + i;
	    var cell = document.getElementById(idstr);
	    if ('aval' in puzzleContent[i]) {
		cell.innerHTML = puzzleContent[i].aval;
		cell.setAttribute("hint", "none");
	    } else if ('bval' in puzzleContent[i]) {
		cell.innerHTML = "&nbsp;";
		cell.setAttribute("hint", "one");
	    } else {
		cell.innerHTML = "&nbsp;";
		if (puzzleContent[i].pvals.length == 1)
		    cell.setAttribute("hint", "one");
		else if (puzzleContent[i].pvals.length == 2)
		    cell.setAttribute("hint", "two");
		else
		    cell.setAttribute("hint", "many");
	    }
	    if (hoverHints) {
		cell.setAttribute("hover", cell.getAttribute("hint"));
	    }else
		cell.setAttribute("hover", "opaque");
	}
    }
};

function fillGuess(guesses, index) {
    if (guesses)
	guessContent = { "guesses": guesses, "max": puzzleSideLength, "index": index };
    else
	guessContent = null;
    refillGuess();
};

function refillGuess() {
    if (guessContent) {
	var max = guessContent.max
	var guesses = guessContent.guesses
	for (i = 1; i <= max; i++) {
	    var idstr = "guess" + i;
	    var button = document.getElementById(idstr);
	    if (guessHints) {
		if (guesses.indexOf(i) > -1)
		    button.setAttribute("guess", "yes");
		else
		    button.setAttribute("guess", "no");
	    } else
		button.setAttribute("guess", "maybe");
	}
	var guessbox = document.getElementById("guessbox");
	guessbox.className = "filled";
	document.addEventListener('keypress', keyGuess);
    } else {
	var guessbox = document.getElementById("guessbox");
	guessbox.className = "empty";
	document.removeEventListener('keypress', keyGuess);
    }
};

function setFeedback(message) {
    document.getElementById("guessFeedback").innerHTML = message
}

function selectCell(idx) {
    // deselect currently selected cell
    if (arguments.callee.selectedCell) {
	cell = arguments.callee.selectedCell
	cell.setAttribute("selected", "")
	if (hoverHints) {
	    cell.setAttribute("hover", cell.getAttribute("hint"))
	} else {
	    cell.setAttribute("hover", "opaque")
	}
	arguments.callee.selectedCell = null
    }
    // find and select sell with given index
    if (idx) {
	var idstr = "c" + idx;
	var cell = document.getElementById(idstr);
	if (cell) {
	    arguments.callee.selectedCell = cell
	    if (selectHints) {
		cell.setAttribute("selected", cell.getAttribute("hint"));
		cell.setAttribute("hover", cell.getAttribute("hint"))
	    } else {
		cell.setAttribute("selected", "opaque");
	    }
	}
    }
}

function clickGuess(guess) {
    if (guessContent.guesses.indexOf(guess) >= 0) {
	setFeedback("Submitting guess...");
	AssignPuzzle(guessContent.index + 1, guess);
    } else
	setFeedback("Guess not allowed!");
    event.stopPropagation();
}

function keyGuess(event) {
    if(event.keyCode >= '1'.charCodeAt(0) && event.keyCode <= '9'.charCodeAt(0)) {
	console.log("Key pressed: ", event.keyCode - '0'.charCodeAt(0));
	clickGuess(event.keyCode - '0'.charCodeAt(0))
    }
}

function clickCell(idx) {
    if (puzzleContent) {
	if ('aval' in puzzleContent[idx]) {
	    fillGuess()
	} else if ('bval' in puzzleContent[idx]) {
	    fillGuess([ puzzleContent[idx].bval ], idx)
	} else
	    fillGuess(puzzleContent[idx].pvals, idx);
    }
    setFeedback("Cell " + (idx + 1));
    selectCell(idx)
    event.stopPropagation();
};

function clickNowhere(event) {
    fillGuess();
    setFeedback("No cell selected");
    selectCell(null)
}

function setHoverHints(val) {
    if (val) hoverHints = true; else hoverHints = false;
    document.getElementById("hoverOn").checked = hoverHints;
    document.getElementById("hoverOff").checked = ! hoverHints;
    if (hoverHints) {
	// turning on hoverHints requires turning on selectHints
	// otherwise your selected color is not your hover color
	setSelectHints(hoverHints)
    } else {
	refillPuzzle();
    }
};

function setSelectHints(val) {
    if (val) selectHints = true; else selectHints = false;
    document.getElementById("selectOn").checked = selectHints;
    document.getElementById("selectOff").checked = ! selectHints;
    if (selectHints) {
	refillPuzzle();
    } else {
	// turning off selectHints requires turning off hoverHints
	// otherwise your selected color is not your hover color
	setHoverHints(selectHints)
    }
    refillPuzzle();
};

function setGuessHints(val) {
    if (val) guessHints = true; else guessHints = false;
    document.getElementById("guessOn").checked = guessHints;
    document.getElementById("guessOff").checked = ! guessHints;
    refillGuess();
};

function undoGuess() {
    LoadPuzzle(backURL);
}

function resetPuzzle() {
    LoadPuzzle(resetURL);
}

function newPuzzle(index) {
    if (index) {
	LoadPuzzle(resetURL + index);
    }
}

function initializePage() {
    LoadPuzzle();
    setHoverHints(hoverHints)
    setSelectHints(selectHints)
    setGuessHints(guessHints)
};
