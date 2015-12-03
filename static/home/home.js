var hoverHints;
var selectHints;
var guessHints;
var puzzleID;
var startURL = "/reset/";
var solverURL = "/solver/";

function clickHoverHints(val) {
    setHoverHints(val);
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

function setPuzzle(pid) {
    if (!pid) {
	pid = "1-star";
    }
    // deselect current puzzle button
    if (puzzleID && pid != puzzleID) {
	button = document.getElementById(puzzleID);
	if (button) {
	    button.setAttribute("current", "no")
	}
    }
    button = document.getElementById(pid);
    if (button) {
	button.setAttribute("current", "yes");
    }
    puzzleID = pid;
    localStorage.puzzleID = puzzleID;
}

function newPuzzle(pid) {
    if (pid && pid == puzzleID) {
	window.location = solverURL;
    } else {
	window.location = startURL + pid;
    }
}

function initializePage() {
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
}
