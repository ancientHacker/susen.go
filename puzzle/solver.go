package puzzle

import (
	"fmt"
)

/*

Sudoku puzzle solver

The solver uses an algorithm adapted from the method used by some
human solvers I have observed.  It is a depth-first search
algorithm that uses a stack for backtracking.  It is called
Ariadne's thread, after the mythical heroine who used a ball of
yarn as a stack in her depth-first search for an exit from the
minotaur's maze.

1. Fill in all the bound values you can.

2. Check the state of the puzzle:

2.1 If the puzzle is solved, you're done.

2.2 If the puzzle has errors, go to step 4.

2.3 The puzzle has unbound, empty squares.  Continue to step 3.

3. Guess a value for an unbound, empty square as follows:

3.1 Find the first unbound square with the fewest number of
possible values.  (Any order for choosing the square works, this
algorithm uses reading order.)

3.2 Save the puzzle state, the chosen square, and the possible
values on the top of the stack.

3.3 Assign the first of the possible values to the chosen square.

3.4 Go to step 1.

4. "Rewind your thread" as follows:

4.1 Pop the stack until you find an entry that has unused choices
for its chosen square.

4.2 If the stack is empty, stop.  The puzzle can't be solved.

4.3 Restore the puzzle state from the state on the stack.

4.4 Fill in the chosen square with the first remaining possible value.

4.5 Go to step 1.

This algorithm yields a sequence of assignments to unbound
squares, in the order they are tried in step 3.2.  These are
stored as entries on the stack, and can be examined upon return
from step 2.

Many Sudoku puzzles found in publications actually have multiple
solutions.  This algorithm can be easily adapted to find all such
solutions by changing step 2 to save the solution and jump to
step 4.

*/

// A choice is a puzzle, a square to choose, the choice to try
// first in that square, and the next choices to try after that.
type choice struct {
	puz    *puzzle
	cindex int
	cvalue int
	cnext  intset
}

// A thread is a stack of choices
type thread []choice

// solve a puzzle using Ariadne's thread.  Entered with a puzzle
// and a stack of prior choices (which can be empty), this finds
// the next possible solution and returns the puzzle and stack at
// time of solution (or unsolvable error).
func solve(p *puzzle, t thread) (*puzzle, thread) {
	for {
		if len(p.errors) == 0 && assignKnown(p) {
			return p, t
		}
		if len(p.errors) > 0 {
			p, t = popChoice(p, t)
			if len(t) == 0 {
				return p, t
			}
			continue
		}
		p, t = pushChoice(p, t)
	}
}

// Solutions finds all solutions to a given puzzle.  The
// puzzle is copied first, so it's not altered during the
// solutions process
func (p *puzzle) Solutions() []Solution {
	var solutions []Solution
	var t thread
	for p, t = solve(p.copy(), t); len(p.errors) == 0; p, t = solve(p, t) {
		solutions = append(solutions, newSolution(p, t))
		p, t = popChoice(p, t)
		if len(t) == 0 {
			break
		}
	}
	return solutions
}

// assignKnown takes a solvable puzzle and tries to solve it by
// assigning all the known empty squares (bound to single-valued)
// to their known value and then looping to see if those
// assignments led to more known values that it can assign.  If
// it is able to fill all the puzzle's empty squares with legal
// values, then it has solved the puzzle and returns true.  If
// there are empty squares left, or if one of its assignments
// make the puzzle unsolvable, then it returns false.
func assignKnown(p *puzzle) bool {
	for {
		known, unknown := 0, 0
		for i := 1; i <= p.mapping.scount; i++ {
			if p.squares[i].aval == 0 {
				if p.squares[i].bval != 0 {
					known++
					p.assign(i, p.squares[i].bval)
				} else if len(p.squares[i].pvals) == 1 {
					known++
					p.assign(i, p.squares[i].pvals[0])
				} else {
					unknown++
				}
				if len(p.errors) > 0 {
					return false
				}
			}
		}
		if unknown == 0 {
			return true
		}
		if known == 0 {
			return false
		}
	}
}

// popChoice resets a puzzle to the next choice after the current
// choice in a thread has failed.  If there is no next choice,
// the incoming puzzle is returned, along with the empty thread.
func popChoice(p *puzzle, t thread) (*puzzle, thread) {
	for len(t) > 0 {
		top := &t[len(t)-1]
		if len(top.cnext) == 0 {
			*top = choice{} // release storage held in choice before pop
			t = t[:len(t)-1]
			continue
		}
		new := top.puz.copy()
		top.cvalue, top.cnext = top.cnext[0], top.cnext[1:]
		new.assign(top.cindex, top.cvalue) // errors handled by caller
		return new, t
	}
	return p, t
}

// pushChoice chooses an unbound square to assign, pushes a
// puzzle copy and the choice on the stack, and then applies that
// choice to the puzzle.
func pushChoice(p *puzzle, t thread) (*puzzle, thread) {
	cindex, ccount := 0, p.mapping.sidelen+1
	for i := 1; i <= p.mapping.scount; i++ {
		if p.squares[i].aval == 0 && p.squares[i].bval == 0 {
			count := len(p.squares[i].pvals)
			if count == 2 {
				cindex, ccount = i, 2
				break
			}
			if count < ccount {
				cindex, ccount = i, count
			}
		}
	}
	if cindex == 0 {
		// internal caller error - called when no choice available
		panic(fmt.Errorf("pushChoice called with no available choices"))
	}
	c := choice{
		puz:    p.copy(),
		cindex: cindex,
		cvalue: p.squares[cindex].pvals[0],
		cnext:  newIntsetCopy(p.squares[cindex].pvals[1:]),
	}
	p.assign(c.cindex, c.cvalue)
	if len(p.errors) > 0 {
		// can't happen: the choice was unacceptable for the square
		panic(fmt.Errorf("Assign of %v to %+v failed: %v",
			c.cvalue, *p.squares[cindex], p.errors))
	}
	return p, append(t, c)
}

// newSolution constructs a solution from a solved puzzle and its
// solving thread.
func newSolution(p *puzzle, t thread) Solution {
	S := Solution{Values: p.allValues()}
	if len(t) > 0 {
		S.Choices = make([]Choice, len(t))
		for i := range t {
			S.Choices[i].Index, S.Choices[i].Value = t[i].cindex, t[i].cvalue
		}
	}
	return S
}
