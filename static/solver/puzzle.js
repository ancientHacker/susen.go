var hoverHints;
var selectHints;
var guessHints;
var puzzleID;
var puzzleContent = null;
var guessContent = null;
var puzzleSideLength = 0;
var squaresURL = "/api/squares/";
var assignURL = "/api/assign/";
var backURL = "/api/back/";
var resetURL = "/api/reset/";

function receivePuzzleSquares() {
    if (this.readyState == 4) {
	// console.log("Got puzzle squares:", this.responseText);
        var squares = JSON.parse(this.responseText);
	fillPuzzle(squares);
	setFeedback("Puzzle received.");
    }
}

var getPuzzleRequest = new XMLHttpRequest();
getPuzzleRequest.onreadystatechange = receivePuzzleSquares;

function receivePuzzleUpdate() {
    if (this.readyState == 4) {
	// console.log("Got puzzle update:", this.responseText);
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

function fillPuzzle(squares) {
    selectCell(null)
    if (squares)
	puzzleContent = squares;
    else
	puzzleContent = null;
    refillPuzzle();
};

function AssignPuzzle(cell, val) {
    var choice = {index: cell, value: val};
    var body = JSON.stringify(choice);
    console.log("POST request to puzzle:", body);
    postAssignRequest.open("POST", assignURL, true);
    postAssignRequest.setRequestHeader("Content-type", "application/json");
    postAssignRequest.send(body);
}

function updatePuzzle(squares) {
    selectCell(null)
    if (squares && puzzleContent) {
	for (i = 0; i < squares.length; i++) {
	    if (squares[i].index > 0 && squares[i].index < puzzleContent.length) {
		puzzleContent[squares[i].index-1] = squares[i]
	    }
	}
    }
    refillPuzzle();
};

function refillPuzzle() {
    if (puzzleContent) {
	for (pcIdx = 0; pcIdx < puzzleContent.length; pcIdx++) {
	    var idstr = "c" + (pcIdx + 1);
	    var cell = document.getElementById(idstr);
	    if ('aval' in puzzleContent[pcIdx]) {
		cell.innerHTML = puzzleContent[pcIdx].aval;
		cell.setAttribute("hint", "none");
	    } else if ('bval' in puzzleContent[pcIdx]) {
		cell.innerHTML = "&nbsp;";
		cell.setAttribute("hint", "one");
	    } else {
		cell.innerHTML = "&nbsp;";
		if (puzzleContent[pcIdx].pvals.length == 1)
		    cell.setAttribute("hint", "one");
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

function fillGuess(guesses, idx, bval, bsrc) {
    if (guesses)
	guessContent = {
	    "guesses": guesses,
	    "max": puzzleSideLength,
	    "index" : idx,
	    "bval": bval,
	    "bsrc": bsrc
	};
    else
	guessContent = null;
    refillGuess();
};

function refillGuess() {
    if (guessContent) {
	var max = guessContent.max
	var guesses = guessContent.guesses
	for (val = 1; val <= max; val++) {
	    var idstr = "guess" + val;
	    var button = document.getElementById(idstr);
	    if (guessHints) {
		if (guesses.indexOf(val) > -1)
		    button.setAttribute("guess", "yes");
		else
		    button.setAttribute("guess", "no");
	    } else
		button.setAttribute("guess", "maybe");
	}
	var guessbox = document.getElementById("guessbox");
	guessbox.className = "filled";
	var whybox = document.getElementById("why")
	if (guessHints && guessContent.bsrc) {
	    whybox.setAttribute("show", "yes")
	} else {
	    whybox.setAttribute("show", "no")
	}
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
    if (idx == -1) {
	// reselect currently selected cell
	idx = arguments.callee.selectedIdx
    }
    // deselect currently selected cell
    if (arguments.callee.selectedIdx && idx != arguments.callee.selectedIdx) {
	var idstr = "c" + arguments.callee.selectedIdx;
	var cell = document.getElementById(idstr);
	cell.setAttribute("selected", "")
	if (hoverHints) {
	    cell.setAttribute("hover", cell.getAttribute("hint"))
	} else {
	    cell.setAttribute("hover", "opaque")
	}
	arguments.callee.selectedIdx = null
	fillGuess();
	setFeedback("No cell selected.");
    }
    // find and select sell with given index
    if (idx) {
	arguments.callee.selectedIdx = idx
	var idstr = "c" + idx;
	var cell = document.getElementById(idstr);
	if (cell) {
	    if (selectHints) {
		cell.setAttribute("selected", cell.getAttribute("hint"));
		cell.setAttribute("hover", cell.getAttribute("hint"))
	    } else {
		cell.setAttribute("selected", "opaque");
	    }
	}
	// fill the guess for the cell
	if (puzzleContent) {
	    var pcIdx = idx - 1;
	    // console.log(puzzleContent);
	    if ('aval' in puzzleContent[pcIdx]) {
		fillGuess();
	    } else if ('bval' in puzzleContent[pcIdx]) {
		var val = puzzleContent[pcIdx].bval
		fillGuess([ val ], idx, val, puzzleContent[pcIdx].bsrc);
	    } else
		fillGuess(puzzleContent[pcIdx].pvals, idx);
	} else {
	    fillGuess();
	}
	setFeedback("Cell " + idx);
    }
}

function clickGuess(guess) {
    if (guessContent.guesses.indexOf(guess) >= 0) {
	setFeedback("Submitting guess...");
	AssignPuzzle(guessContent.index, guess);
    } else {
	setFeedback("Guess not allowed!");
    }
    event.stopPropagation();
}

function keyGuess(event) {
    if(event.keyCode >= '1'.charCodeAt(0) && event.keyCode <= '9'.charCodeAt(0)) {
	// console.log("Key pressed: ", event.keyCode - '0'.charCodeAt(0));
	clickGuess(event.keyCode - '0'.charCodeAt(0));
    }
}

function clickWhy(event) {
    event.stopPropagation();
    if (guessContent.bsrc) {
	var reasons = "Cell " + guessContent.index;
	reasons += " is the only cell that can contain " + guessContent.bval;
	for (i = 0; i < guessContent.bsrc.length; i++) {
	    if (i == 0) {
		reasons += " in ";
	    } else {
		reasons += " and ";
	    }
	    reasons += guessContent.bsrc[i].gtype + " " + guessContent.bsrc[i].index;
	}
	setFeedback(reasons + ".");
    }
}

function clickCell(idx) {
    selectCell(idx)
    event.stopPropagation();
};

function clickNowhere(event) {
    selectCell(null)
    event.stopPropagation();
}

function clickHoverHints(val) {
    setHoverHints(val);
    refillPuzzle();
    selectCell(-1);
    event.stopPropagation();
}

function setHoverHints(val) {
    if (val) {
	hoverHints = true;
	// localStorage often can only contain strings
	localStorage.hoverHints = "yes";
	// turning on hoverHints requires turning on selectHints
	// otherwise your selected color is not your hover color
	setSelectHints(true);
    } else {
	hoverHints = false;
	// localStorage often can only contain strings
	localStorage.hoverHints = "no";
    }
    document.getElementById("hoverOn").checked = hoverHints;
    document.getElementById("hoverOff").checked = ! hoverHints;
};

function clickSelectHints(val) {
    setSelectHints(val);
    refillPuzzle();
    selectCell(-1);
    event.stopPropagation();
}

function setSelectHints(val) {
    if (val) {
	selectHints = true;
	// localStorage often can only contain strings
	localStorage.selectHints = "yes";
    } else {
	selectHints = false;
	// localStorage often can only contain strings
	localStorage.selectHints = "no";
	// turning off selectHints requires turning off hoverHints
	// otherwise your selected color is not your hover color
	setHoverHints(selectHints)
    }
    document.getElementById("selectOn").checked = selectHints;
    document.getElementById("selectOff").checked = ! selectHints;
};

function clickGuessHints(val) {
    setGuessHints(val);
    refillGuess();
    event.stopPropagation();
}

function setGuessHints(val) {
    if (val) {
	guessHints = true;
	// localStorage often can only contain strings
	localStorage.guessHints = "yes";
    } else {
	guessHints = false;
	// localStorage often can only contain strings
	localStorage.guessHints = "no";
    }
    document.getElementById("guessOn").checked = guessHints;
    document.getElementById("guessOff").checked = ! guessHints;
};

function undoGuess() {
    LoadPuzzle(backURL);
}

function resetPuzzle() {
    LoadPuzzle(resetURL);
}

function setPuzzle(pid) {
    if (!pid) {
	pid = "1-star";
    }
    // // deselect current puzzle button
    // if (puzzleID && pid != puzzleID) {
    // 	button = document.getElementById(puzzleID);
    // 	if (button) {
    // 	    button.setAttribute("current", "no")
    // 	}
    // }
    // button = document.getElementById(pid);
    // if (button) {
    // 	button.setAttribute("current", "yes");
    // }
    puzzleID = pid;
    localStorage.puzzleID = puzzleID;
}

function goHome() {
    window.location = "/home/"
}

function initializePage(sideLen) {
    sessionID = document.body.getAttribute("sessionID")
    if (localStorage.sessionID == sessionID) {
	// reuse existing session
	setHoverHints(localStorage.hoverHints == "yes")
	setSelectHints(localStorage.selectHints != "no")
	setGuessHints(localStorage.guessHints == "yes")
	if (!sessionID) {
	    console.log("Warning: empty session ID")
	}
    } else {
	// create new session
	localStorage.sessionID = sessionID
	setHoverHints(false)
	setSelectHints(true)
	setGuessHints(false)
    }
    puzzleID = document.body.getAttribute("puzzleID")
    if (puzzleID) {
	setPuzzle(puzzleID)
    } else {
	console.log("Warning: empty puzzle ID: using previous:", localStorage.puzzleID)
	puzzleID = localStorage.puzzleID
    }
    if (sideLen) {
	puzzleSideLength = sideLen
    } else {
	console.log("Warning: no side length specified, guessing 9!")
	puzzleSideLength = 9
    }
    LoadPuzzle()
}
