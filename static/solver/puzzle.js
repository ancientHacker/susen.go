var hoverHints;			// are we giving hover hints?
var selectHints;		// are we giving selection hints?
var guessHints;			// are we giving guess hints?
var puzzleID;			// puzzle's name
var boundCount = 0;		// how many single-value-only unassigned squares are in the puzzle
var puzzleContent = null;	// squares in the puzzle
var puzzleErrors = null;	// errors in the puzzle
var guessContent = null;	// allowed guess info for a selected square
var puzzleSideLength = 0;	// side length of the puzzle
var stateURL = "/api/state/";
var assignURL = "/api/assign/";
var backURL = "/api/back/";
var resetURL = "/api/reset/";
var homeURL = "/home/";
var solverURL = "/solver/";

function receivePuzzleState() {
    if (this.readyState == 4) {
	if (this.status == 200) {
	    // console.log("Got puzzle state:", this.responseText);
            var result = JSON.parse(this.responseText);
	    fillPuzzle(result.squares);
	    if ("errors" in result) {
		puzzleErrors = result.errors
		message = puzzleErrorMessage()
		setFeedback("Puzzle can't be solved. " + message);
	    } else {
		puzzleErrors = null
		setFeedback("Click a square to select it.");
	    }
	} else if (this.status >= 400 && this.status < 500) {
            var result = JSON.parse(this.responseText);
	    setFeedback("Couldn't load puzzle; will retry in 4 seconds:<br />" + result.message);
	    setTimeout(function(){window.location = solverURL;}, 4000);
	} else {
	    setFeedback("Couldn't load puzzle; will retry in 4 seconds:<br />Internal Server Error.");
	    setTimeout(function(){window.location = solverURL;}, 4000);
	}
    }
}

var getStateRequest = new XMLHttpRequest();
getStateRequest.onreadystatechange = receivePuzzleState;

function receivePuzzleUpdate() {
    if (this.readyState == 4) {
	if (this.status == 200) {
	    // console.log("Got puzzle update:", this.responseText);
            var result = JSON.parse(this.responseText);
	    updatePuzzle(result.squares);
	    if ("errors" in result) {
		puzzleErrors = result.errors
		message = puzzleErrorMessage()
		setFeedback("Assign made puzzle unsolvable. " + message);
	    } else {
		puzzleErrors = null
		setFeedback("Assign successful; puzzle updated.");
	    }
	} else if (this.status >= 400 && this.status <= 500) {
            var result = JSON.parse(this.responseText);
	    setFeedback("Assign failed:<br />" + result.message);
	    setTimeout(function(){window.location = solverURL;}, 4000);
	} else {
	    setFeedback("Couldn't load puzzle; will retry in 4 seconds:<br />Internal Server Error.");
	    setTimeout(function(){window.location = solverURL;}, 4000);
	}
    }
}

var postAssignRequest = new XMLHttpRequest();
postAssignRequest.onreadystatechange = receivePuzzleUpdate;

function LoadPuzzle(url) {
    if (!url) {
	url = stateURL;
    }
    console.log("GET request for", url);
    getStateRequest.open("GET", url, true);
    getStateRequest.send(null);
}

function fillPuzzle(squares) {
    selectCell(null)
    if (squares) {
	puzzleContent = squares;
	boundCount = 0;
	for (i = 0; i < puzzleContent.length; i++) {
	    if ("bval" in squares[i] || ("pvals" in squares[i] && squares[i].pvals.length == 1)) {
		boundCount++;
	    }
	}
    }
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
	    if (squares[i].index > 0 && squares[i].index <= puzzleContent.length) {
		var pcIdx = squares[i].index - 1;
		var wasBound = "bval" in puzzleContent[pcIdx] ||
		    ("pvals" in puzzleContent[pcIdx] && puzzleContent[pcIdx].pvals.length == 1);
		var isBound = "bval" in squares[i] ||
		    ("pvals" in squares[i] && squares[i].pvals.length == 1);
		puzzleContent[squares[i].index-1] = squares[i];
		if  (wasBound != isBound) {
		    if (wasBound) boundCount--; else boundCount++;
		}
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
	    } else if ('pvals' in puzzleContent[pcIdx]) {
		cell.innerHTML = "&nbsp;";
		plen = puzzleContent[pcIdx].pvals.length
		if (plen == 1)
		    cell.setAttribute("hint", "one");
		else if (boundCount > 0) {
		    cell.setAttribute("hint", "many");
		} else if (plen == 2) {
		    cell.setAttribute("hint", "two")
		} else {
		    cell.setAttribute("hint", "many");
		}
	    } else {
		cell.innerHTML = "&empty;";
		cell.setAttribute("hint", "zero");
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

function puzzleErrorMessage() {
    var message = ""
    if (puzzleErrors) {
	message += "Puzzle not solvable:"
	for (i = 0; i < puzzleErrors.length; i++) {
	    message += "<br />" + puzzleErrors[i].message
	}
    }
    return message
}    

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
	setFeedback("Click a square to select it.");
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
	if (puzzleErrors) {
	    emsg = puzzleErrorMessage()
	    setFeedback("Cell " + idx + " selected. " + emsg)
	} else {
	    if (puzzleContent) {
		var pcIdx = idx - 1;
		// console.log(puzzleContent);
		if ('aval' in puzzleContent[pcIdx]) {
		    fillGuess();
		} else if ('bval' in puzzleContent[pcIdx]) {
		    var val = puzzleContent[pcIdx].bval
		    fillGuess([ val ], idx, val, puzzleContent[pcIdx].bsrc);
		} else if ('pvals' in puzzleContent[pcIdx]) {
		    fillGuess(puzzleContent[pcIdx].pvals, idx);
		} else {
		    fillGuess([], idx)
		}
	    } else {
		fillGuess();
	    }
	    setFeedback("Cell " + idx);
	}
    }
}

function clickGuess(guess) {
    event.stopPropagation();
    if (guessContent.guesses.indexOf(guess) >= 0) {
	setFeedback("Submitting guess...");
	AssignPuzzle(guessContent.index, guess);
    } else {
	setFeedback("Guess not allowed!");
    }
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
    event.stopPropagation();
    selectCell(idx)
};

function clickNowhere(event) {
    event.stopPropagation();
    selectCell(null)
}

function clickHoverHints(val) {
    event.stopPropagation();
    setHoverHints(val);
    refillPuzzle();
    selectCell(-1);
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
    event.stopPropagation();
    setSelectHints(val);
    refillPuzzle();
    selectCell(-1);
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
    event.stopPropagation();
    setGuessHints(val);
    refillGuess();
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
    window.location = homeURL;
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
    LoadPuzzle();
}
