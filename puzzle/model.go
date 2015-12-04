package puzzle

/*

Sudoku puzzle representation

*/

import (
	"fmt"
)

/*

Puzzles

*/

// A puzzle (actually a reference to a puzzle) is our internal
// implementation of the Puzzle interface.  In addition to the
// geometry's puzzle mapping, it stores the actual square data
// and group data of the puzzle, and tracks any Errors that
// prevent the puzzle from being solved.
//
// Each puzzle has an associated logger so assigns can track the
// squares that are modified.
type puzzle struct {
	mapping *puzzleMapping
	squares []*square
	groups  []*group
	errors  []Error
	logger  *indexLogger
}

// indicesToValues is a helper that takes an intset of indices
// and returns the values in the squares with those indices.
func (p *puzzle) indicesToValues(is intset) []int {
	vs := make([]int, len(is))
	for i, idx := range is {
		vs[i] = p.squares[idx].aval
	}
	return vs
}

// allValues returns all the assigned values in the puzzle squares.
func (p *puzzle) allValues() []int {
	is := newIntsetRange(p.mapping.scount)
	return p.indicesToValues(is)
}

// indicesToPossibles is a helper that takes an intset of indices
// and returns the possible values in the squares with those
// indices.  The return value does not share storage with the
// puzzle.
func (p *puzzle) indicesToPossibles(is intset) [][]int {
	vs := make([][]int, len(is))
	for i, idx := range is {
		vs[i] = newIntsetCopy(p.squares[idx].pvals)
	}
	return vs
}

// allPossibles returns the possible values for all of a puzzle's
// squares.
func (p *puzzle) allPossibles() [][]int {
	is := newIntsetRange(p.mapping.scount)
	return p.indicesToPossibles(is)
}

// indicesToSquares is a helper that takes an intset of indices
// and creates a slice of Squares for those indices.
func (p *puzzle) indicesToSquares(is intset) []Square {
	SS := make([]Square, len(is))
	for i, idx := range is {
		S, s := &SS[i], p.squares[idx]
		S.Index = s.index
		if s.aval != 0 {
			S.Aval = s.aval
			continue
		}
		S.Pvals = newIntsetCopy(s.pvals)
		if len(s.pvals) == 1 {
			// don't return bindings if only one value,
			// because they are extraneous and confusing.
			continue
		}
		if s.bval != 0 {
			S.Bval = s.bval
			S.Bsrc = append(S.Bsrc, s.bsrc...)
		}
	}
	return SS
}

// allSquares returns a Square for each of a puzzle's squares.
func (p *puzzle) allSquares() []Square {
	is := newIntsetRange(p.mapping.scount)
	return p.indicesToSquares(is)
}

// allErrors returns the puzzle's Errors.  The returned slice
// doesn't share storage with the puzzle.
func (p *puzzle) allErrors(verbose bool) []Error {
	errs := append([]Error(nil), p.errors...)
	if verbose {
		for i := range errs {
			errs[i].Message = errs[i].Error() // verbalize the error
		}
	}
	return errs
}

// assign a value to an (assumed) empty square in a puzzle,
// returning an intset of the indices of all the squares modified
// during the assignment (including the assigned square).
//
// Does constraint relaxation to remove possible values and to
// bind squares based on the assignment.  Any Errors produced by
// the assignment or the constraint relaxation are added to the
// puzzle.
func (p *puzzle) assign(idx, val int) intset {
	// set up to log the affected squares, so they can be returned.
	p.logger.start(idx)
	// after we're done, reset the puzzle logger
	defer func() { p.logger.stop() }()

	// do the assignment
	errs := p.squares[idx].assign(val)
	if len(errs) > 0 {
		p.errors = append(p.errors, errs...)
	}

	// propagate the assignment through the containing groups,
	// which happens in three parts:
	//
	// Part 1: Find all the groups containing squares that will
	// be affected by the assignment.  This is not just the three
	// groups containing the assigned square, but also the groups
	// containing unassigned squares in those three containing
	// groups (because those unassigned squares will have the
	// assigned value removed).
	affected := make([]int, p.mapping.gcount+1) // 1-based group indexes
	for _, gi := range p.mapping.ixmap[idx] {
		// this group needs to be analyzed
		affected[gi]++
		for _, ei := range p.mapping.gdescs[gi].indices {
			// and for each of its unassigned squares...
			if p.squares[ei].aval == 0 {
				// ... its containing groups need to be analyzed
				for _, gi := range p.mapping.ixmap[ei] {
					affected[gi]++
				}
			}
		}
	}

	// Part 2: Notify the three groups containing the assigned
	// square of the assignment.  Each of them will remove the
	// assigned value from all their unassigned squares
	for _, gi := range p.mapping.ixmap[idx] {
		if errs := p.groups[gi].assign(p.squares, idx); len(errs) > 0 {
			// group assign Errors make the puzzle unsolvable
			p.errors = append(p.errors, errs...)
		}
	}

	/// Part 3: Analyze all the affected groups.  This allows
	/// them to discover solvability problems and also required
	/// bindings induced by the assignment.
	for gi, count := range affected {
		if count > 0 {
			if errs := p.groups[gi].analyze(p.squares); len(errs) > 0 {
				// group analyze Errors make the puzzle unsolvable
				p.errors = append(p.errors, errs...)
			}
		}
	}
	return p.logger.entries
}

// copy returns a deep copy of a puzzle
func (p *puzzle) copy() *puzzle {
	if p == nil {
		return nil
	}
	// first the puzzle structure
	c := &puzzle{
		mapping: p.mapping,          // mappings are invariant and always shared
		logger:  &indexLogger{},     // loggers are per-puzzle, initialized empty
		errors:  p.allErrors(false), // errors are per-puzzle, copied from source
	}
	// then the squares
	c.squares = make([]*square, c.mapping.scount+1) // 1-based indexing
	for i := 1; i <= c.mapping.scount; i++ {
		c.squares[i] = &square{
			index:  p.squares[i].index,
			aval:   p.squares[i].aval,
			pvals:  newIntsetCopy(p.squares[i].pvals),
			bval:   p.squares[i].bval,
			bsrc:   append([]GroupID(nil), p.squares[i].bsrc...),
			logger: c.logger,
		}
	}
	// then the groups
	c.groups = make([]*group, c.mapping.gcount+1) // 1-based indexing
	for i := 1; i <= c.mapping.gcount; i++ {
		c.groups[i] = &group{
			desc:  p.groups[i].desc, // descriptors are part of mappings, so shared
			where: append([]int(nil), p.groups[i].where...),
			need:  newIntsetCopy(p.groups[i].need),
			free:  newIntsetCopy(p.groups[i].free),
		}
	}
	return c
}

/*

Interface entries: if you call these with a nil puzzle pointer,
you will panic.  That's because the interface entries are
supposed to be called through the interface wrapper, and you
should never be wrapping the interface around a nil puzzle
pointer!

*/

// State returns a *State object for the puzzle
func (p *puzzle) State() State {
	return State{
		int(p.mapping.geometry),
		p.mapping.sidelen,
		p.allValues(),
		p.allErrors(true),
	}
}

// Squares returns a Square for each of squares in a puzzle (in
// index order).  The return value does not share underlying
// storage with the puzzle, so future changes to the puzzle do
// not affect prior returns from Squares.
func (p *puzzle) Squares() []Square {
	return p.allSquares()
}

// Assign a choice to a puzzle, returning an Update for the
// puzzle.  If the puzzle is already unsolvable, the target
// square is already assigned, or the assigned index or value are
// out of range, the puzzle isn't updated and an Error is
// returned.
func (p *puzzle) Assign(choice Choice) (Update, error) {
	if count := len(p.errors); count != 0 {
		err := Error{
			Scope:     ArgumentScope,
			Structure: ScopeStructure,
			Condition: InvalidPuzzleAssignmentCondition,
		}
		err.Message = err.Error()
		return Update{}, err
	}
	idx, val := choice.Index, choice.Value
	if idx < 1 || idx > p.mapping.scount {
		return Update{}, rangeError(IndexAttribute, idx, 1, p.mapping.scount)
	}
	if val < 1 || val > p.mapping.sidelen {
		return Update{}, rangeError(ValueAttribute, val, 1, p.mapping.sidelen)
	}
	if p.squares[idx].aval != 0 {
		err := Error{
			Scope:     ArgumentScope,
			Structure: AttributeValueStructure,
			Attribute: AssignedValueAttribute,
			Condition: DuplicateAssignmentCondition,
			Values:    ErrorData{val, idx, p.squares[idx].aval},
		}
		err.Message = err.Error()
		return Update{}, err
	}

	// assigning this value to this square is allowed, so try it
	is := p.assign(idx, val)
	return Update{p.indicesToSquares(is), p.allErrors(true)}, nil
}

// Copy returns a copy of the wrapped puzzle (no shared structure)
func (p *puzzle) Copy() Puzzle {
	return p.copy()
}

/*

Puzzle construction

*/

// create takes a mapping and a list of assigned values, one for
// each square, and creates a new puzzle filled with the given
// values.  Input values of 0 mean an empty square.  Gives an
// Error if the values are out of range for the puzzle.
// Constraint relaxation is done on the puzzle, so that
// unassigned squares have the minimal set of possible values,
// and all possible bindings have been done.  This may lead to
// the returned puzzle having Errors, which make it unsolvable.
func create(mapping *puzzleMapping, values []int) (*puzzle, error) {
	// create the square array.  Errors encountered in this phase
	// mean that the puzzle can not be created because the inputs
	// were bad.
	squares := make([]*square, len(values)+1) // 1-based indices
	logger := &indexLogger{}                  // uninitialized, so no logging done
	for i, val := range values {
		if val == 0 {
			squares[i+1] = newEmptySquare(i+1, mapping.sidelen, logger)
		} else {
			if val < 1 || val > mapping.sidelen {
				return nil, rangeError(ValueAttribute, val, 1, mapping.sidelen)
			}
			squares[i+1] = newFilledSquare(i+1, mapping.sidelen, val, logger)
		}
	}

	// Assemble the groups, which will remove the assigned values
	// from all of the unassigned squares in those groups.
	// Errors encountered in this phase and the next mean the
	// puzzle is not solvable, so they are added to the puzzle
	// and the puzzle is returned.
	var errs, errors []Error
	groups := make([]*group, mapping.gcount+1) // 1-based indices
	for i := 1; i <= mapping.gcount; i++ {
		groups[i], errs = newGroup(&mapping.gdescs[i], squares)
		if len(errs) > 0 {
			errors = append(errors, errs...)
		}
	}

	// Analyze the constructed groups, which will assemble their
	// candidate lists and then do constraint relaxation.
	for i := 1; i <= mapping.gcount; i++ {
		errs = groups[i].analyze(squares)
		if len(errs) > 0 {
			errors = append(errors, errs...)
		}
	}

	// assemble the puzzle from its pieces
	return &puzzle{mapping, squares, groups, errors, logger}, nil
}

/*

Groups

*/

// A group is a set of squares that together must contain one of
// each number, thus: a row, a column, a sub-square (aka a tile).
// Groups keep track of which values are assigned (and where),
// which values they still need, and which squares are free (that
// is, can be candidates for the needed values).  Then, whenever
// asked (which is when one of their squares has been assigned),
// they analyze the free squares to see if any needed values have
// only one candidate.  If so, they bind the candidate to the
// needed value, and remove the needed value and the (formerly)
// free square.
//
// NOTE: Groups do not look at bindings deduced by other groups.
// They always assume any of their free squares can take on any
// of its possible values.  If two groups disagree on the binding
// of a square, this shows up as an Error when the second group
// tries to bind the square to a different value.
type group struct {
	desc  *groupDescriptor
	where []int  // array map: where[v] = index of square with assigned value v
	need  intset // values the group still needs assigned or bound
	free  intset // indexes of squares not yet assigned or bound
}

// newGroup constructor: create the specified group of squares,
// which may already have assigned values.  Returns a list of
// Errors encountered during the construction of the group.
func newGroup(gd *groupDescriptor, ss []*square) (*group, []Error) {
	// initialize the group members
	sidelen := len(gd.indices)
	where := make([]int, sidelen+1) // 1-based values
	need := newIntsetRange(sidelen)
	free := append(intset(nil), gd.indices...)

	// work in two passes:
	//
	// Pass 1: walk the assigned squares, rembering what value is
	// assigned where, removing the assigned values from the
	// needed values, and removing all assigned squares from the
	// free squares
	var errs []Error
	for _, i := range gd.indices {
		s := ss[i]
		if a := s.aval; a != 0 {
			if where[a] != 0 {
				errs = append(errs, groupError(gd.id, a, DuplicateGroupValuesCondition))
			}
			where[a] = i
			free.remove(i)
			need.remove(a)
		}
	}

	// Pass 2: Walk the non-assigned (free) squares, removing
	// assigned values from them.
	for _, i := range free {
		errs = append(errs, ss[i].intersect(need)...)
	}

	return &group{gd, where, need, free}, errs
}

// analyze a group for solvability.  For each needed value in a
// group, we find all look at which of the free squares in the
// group to see how many are candidates:
//
// - if there are none, the puzzle cannot be solved, and we
// return an Error to indicate this.
//
// - if there is only one, the puzzle can only be solved by
// assigning that value to that square, so we bind the square.
// If that produces an Error, we return that Error.
//
// The result of the analysis is the sequence of Errors (if any)
// that were generated.
//
// Both group construction and group assignment must be followed
// by this analysis.  Neither of them includes analysis because
// the constructed or assigned group can not be analyzed alone;
// the overlapping groups need to be constructed/assigned before
// all of them can be analyzed together.
func (g *group) analyze(ss []*square) []Error {
	counts := make([]int, len(g.desc.indices)+1) // candidate counts for each needed value
	lasts := make([]int, len(g.desc.indices)+1)  // last candidates for each needed value

	// First walk the list of free squares, collecting which ones
	// are candidates for which values.  We walk the list back to
	// front, so we can remove bound values without screwing up
	// the iteration.
	for fi := len(g.free) - 1; fi >= 0; fi-- {
		i := g.free[fi]
		if len(ss[i].pvals) == 1 {
			// this square can only have one value, so it must be
			// used as the candidate for that value.
			g.free.remove(i)
			g.need.remove(ss[i].pvals[0])
		} else {
			// remember this square as a candidate for each of
			// its possible values
			for _, v := range ss[i].pvals {
				counts[v]++
				lasts[v] = i
			}
		}
	}
	// Now walk the list of candidates for each needed value,
	// raising an Error if there aren't any, and binding them if
	// they are the only ones.  We walk the list back to front,
	// so we can remove bound values without screwing up the
	// iteration.
	var errs []Error
	for i := len(g.need) - 1; i >= 0; i-- {
		switch v := g.need[i]; counts[v] {
		case 0:
			errs = append(errs, groupError(g.desc.id, v, NoGroupValueCondition))
		case 1:
			errs = append(errs, ss[lasts[v]].bind(v, g.desc.id)...)
			g.free.remove(lasts[v])
			g.need.remove(v)
		}
	}
	return errs
}

// Add an assigned square to a group, which has just had some
// possible values removed.  Removes the square's assigned values
// from all unassigned squares in the group, returning an Error
// if this removal produces an Error.  This is the single-square
// equivalent of what happens during group construction.
func (g *group) assign(ss []*square, ai int) []Error {
	var errs []Error
	av := ss[ai].aval
	if av == 0 {
		// not really assigned; this shouldn't happen!
		panic(fmt.Errorf("In %v.assign(%v): square is not assigned!", g, ss[ai]))
	}

	// check if we've already seen this assignemnt
	if wi := g.where[av]; wi != 0 {
		if wi == ai {
			return nil
		}
		errs = append(errs, groupError(g.desc.id, av, DuplicateGroupValuesCondition))
	}

	// record the assignment
	g.where[av] = ai
	g.need.remove(av)
	g.free.remove(ai)

	// remove this possible value from all the unassigned squares in the group
	for _, i := range g.desc.indices {
		if ss[i].aval == 0 {
			errs = append(errs, ss[i].remove(av)...)
		}
	}
	return errs
}

/*

Squares

*/

// A square has an index, an optional assigned value (0 if
// unassigned), a set of possible values, and an optional bound
// value (the only one of the possible values that can be
// assigned in a solution).  If one or more groups have forced
// the bound value (as opposed to an assignment or removal of all
// other possible values), the indexes of the group are also
// recorded for explanation to users.
//
// Squares also have a logger, where they log modifications.
type square struct {
	index  int
	aval   int
	pvals  intset
	bval   int
	bsrc   []GroupID
	logger *indexLogger
}

// Make an empty square with the given index in a puzzle with the
// given side length.  Doesn't do error checking.
func newEmptySquare(index, sidelen int, logger *indexLogger) *square {
	return &square{index: index, pvals: newIntsetRange(sidelen), logger: logger}
}

// Make a square with the given index in a puzzle with the given
// side length, and fill it with the given value.  Doesn't do
// error checking.
func newFilledSquare(index, sidelen int, value int, logger *indexLogger) *square {
	return &square{index: index, aval: value, logger: logger}
}

// Assign a value to an empty square.  Returns any errors
// generated by the assignment.  Doesn't guard against the square
// already being assigned, and will assign an impossible value.
func (s *square) assign(aval int) (errs []Error) {
	if s.bval != 0 && s.bval != aval {
		for i := range s.bsrc {
			errs = append(errs, groupError(s.bsrc[i], s.bval, NoGroupValueCondition))
		}
	}
	_, found := s.pvals.find(aval)
	if !found {
		errs = append(errs, squareError(s, aval, AssignedValueAttribute, NotInSetCondition))
	}
	s.aval = aval
	s.pvals = nil
	s.logger.log(s.index)
	return
}

// Bind one of multiple possible values to a square, remembering
// the source of the binding.  Returns any Errors generated by
// the binding.  Doesn't guard against the square being assigned,
// or binding an impossible value.
func (s *square) bind(bval int, bsrc GroupID) (errs []Error) {
	if s.bval != 0 && s.bval != bval {
		for i := range s.bsrc {
			errs = append(errs, groupError(s.bsrc[i], s.bval, NoGroupValueCondition))
		}
	}
	_, found := s.pvals.find(bval)
	if !found {
		errs = append(errs, squareError(s, bval, BoundValueAttribute, NotInSetCondition))
	}
	s.bval = bval
	s.bsrc = append(s.bsrc, bsrc)
	s.logger.log(s.index)
	return
}

// Remove a possible value from an empty square.  Returns any
// Errors generated by the removal.  Doesn't guard against the
// square being assigned, or being left with no possible values.
func (s *square) remove(val int) (errs []Error) {
	if val == s.bval {
		for i := range s.bsrc {
			errs = append(errs, groupError(s.bsrc[i], s.bval, NoGroupValueCondition))
		}
	}
	removed := s.pvals.remove(val)
	if removed {
		if len(s.pvals) == 0 {
			errs = append(errs,
				squareError(s, val, RemovedValueAttribute, NoPossibleValuesCondition))
		}
		s.logger.log(s.index)
	}
	return
}

// Subtract possible values from a square.  Returns any Errors
// generated by the removal.  Doesn't guard against the square
// being assigned, or being left with no possible values.
func (s *square) subtract(vals intset) []Error {
	return s.removeMultiple(vals, false)
}

// Intersect possible values on a square.  Returns any Errors
// generated by the intersection.  Doesn't guard against the
// square being assigned, or being left with no possible values.
func (s *square) intersect(vals intset) []Error {
	return s.removeMultiple(vals, true)
}

// Validate and apply the result of a set operation on a square.
// This is a helper that does the work of subract and intersect.
func (s *square) removeMultiple(vals intset, keepVals bool) (errs []Error) {
	var remsome, rembound bool
	var attr ErrorAttribute
	if keepVals {
		attr = RetainedValuesAttribute
		remsome, rembound = s.pvals.intersect(vals, s.bval)
	} else {
		attr = RemovedValuesAttribute
		remsome, rembound = s.pvals.subtract(vals, s.bval)
	}
	if rembound {
		for i := range s.bsrc {
			errs = append(errs, groupError(s.bsrc[i], s.bval, NoGroupValueCondition))
		}
	}
	if len(s.pvals) == 0 {
		errs = append(errs, squareError(s, vals, attr, NoPossibleValuesCondition))
	}
	if remsome {
		s.logger.log(s.index)
	}
	return
}

/*

indexLoggers

*/

// An indexLogger is an intset that is used to log indices.
type indexLogger struct {
	logging bool
	entries intset
}

// start turns on a logger, giving it an initial entry.
func (l *indexLogger) start(idx int) {
	if l != nil {
		l.logging = true
		l.entries = intset{idx}
	}
}

// stop turns off a logger, leaving its entries intact.
func (l *indexLogger) stop() {
	if l != nil {
		l.logging = false
	}
}

// log adds an index to a logger, if it's operating.
func (l *indexLogger) log(idx int) {
	if l != nil {
		if l.logging {
			l.entries.insert(idx)
		}
	}
}

/*

Integer sets

*/

// An intset is a set of integers, represented as a sorted slice.
// We use intsets to represent both sets of possible values for
// squares and sets of indices.
type intset []int

// newIntsetRange: Make an intset from a range of values, 1 to max.
func newIntsetRange(max int) intset {
	if max < 1 {
		return intset{}
	}
	out := make(intset, max)
	for i := 0; i < max; i++ {
		out[i] = i + 1
	}
	return out
}

// newIntsetCopy: Make a copy of an intset.
func newIntsetCopy(in intset) intset {
	if in == nil {
		return nil
	}
	out := make(intset, len(in))
	copy(out, in)
	return out
}

// Find value v, returning where it should be in the intset and
// whether it was found there.
func (ps *intset) find(v int) (int, bool) {
	end := len(*ps)
	where := end
	for i := 0; i < end; i++ {
		if (*ps)[i] == v {
			return i, true
		}
		if (*ps)[i] > v {
			where = i
			break
		}
	}
	return where, false
}

// Insert value v, returning whether it was there already.
func (ps *intset) insert(v int) bool {
	end := len(*ps)
	where, found := ps.find(v)
	if found {
		return true
	}
	// insert by lengthening, shifting, inserting
	// see https://github.com/golang/go/wiki/SliceTricks
	*ps = append(*ps, v)
	if where < end {
		copy((*ps)[where+1:], (*ps)[where:])
		(*ps)[where] = v
	}
	return false
}

// Remove value v, returning whether it was there.
func (ps *intset) remove(v int) bool {
	end := len(*ps)
	for i := 0; i < end; i++ {
		pv := (*ps)[i]
		if pv == v {
			copy((*ps)[i:], (*ps)[i+1:])
			*ps = (*ps)[:end-1]
			return true
		}
		if pv > v {
			return false
		}
	}
	return false
}

// Subtract the passed intset, returning whether anything was
// removed.  Also takes a marker value and returns whether it was
// removed.
func (ps *intset) subtract(xs intset, marker int) (bool, bool) {
	pend, xend := len(*ps), len(xs)
	pi := 0
	newend := pi
	remmarker := false
	// process the input set
	for xi := 0; pi < pend && xi < xend; {
		pv, xv := (*ps)[pi], xs[xi]
		switch {
		case pv == xv:
			if pv == marker {
				remmarker = true
			}
			pi++
			xi++
		case pv < xv:
			if newend != pi {
				(*ps)[newend] = pv
			}
			newend++
			pi++
		case pv > xv:
			xi++
		}
	}
	if newend == pi {
		// nothing was removed
		return false, false
	}
	// copy any remaining non-removed values
	newend += copy((*ps)[newend:], (*ps)[pi:])
	*ps = (*ps)[:newend]
	return true, remmarker
}

// Intersect the passed intset, returning whether anything was
// removed.  Also takes a marker value and returns whether it was
// removed.
func (ps *intset) intersect(xs intset, marker int) (bool, bool) {
	pend, xend := len(*ps), len(xs)
	sawmarker := false
	savedmarker := false
	pi := 0
	newend := pi
	// process the input set
	for xi := 0; pi < pend && xi < xend; {
		pv, xv := (*ps)[pi], xs[xi]
		if pv == marker {
			sawmarker = true
		}
		switch {
		case pv == xv:
			if pv == marker {
				savedmarker = true
			}
			if newend != pi {
				(*ps)[newend] = pv
			}
			newend++
			pi++
			xi++
		case pv < xv:
			pi++
		case pv > xv:
			xi++
		}
	}
	// process the removed tail
	for _, pv := range (*ps)[pi:] {
		if pv == marker {
			sawmarker = true
		}
	}
	*ps = (*ps)[:newend]
	return newend < pend, sawmarker && !savedmarker
}

/*

Errors: used to report problems making and operating on puzzles.

*/

// rangeError returns an Error that describes an out-of-range argument.
func rangeError(attr ErrorAttribute, val int, min int, max int) Error {
	err := Error{
		Scope:     ArgumentScope,
		Structure: AttributeValueStructure,
		Attribute: attr,
		Condition: TooLargeCondition,
		Values:    ErrorData{val, max},
	}
	if val < min {
		err.Condition = TooSmallCondition
		err.Values[1] = min
	}
	return err
}

// squareError returns an Error from an attempted operation on a
// square that would violate a constraint on the square.  The
// square has not been modified when this error is returned.
func squareError(s *square, v interface{}, attr ErrorAttribute, cond ErrorCondition) Error {
	err := Error{
		Scope:     SquareScope,
		Structure: AttributeValueStructure,
		Attribute: attr,
		Condition: cond,
		Values:    ErrorData{s.index, v},
	}
	switch cond {
	case NotInSetCondition:
		err.Values = append(err.Values, s.pvals)
	case NoPossibleValuesCondition:
	default:
		panic(fmt.Errorf("Unexpected square error condition (%v) in square %+v", cond, *s))
	}
	return err
}

func groupError(gid GroupID, v int, cond ErrorCondition) Error {
	err := Error{
		Scope:     GroupScope,
		Structure: ScopeStructure,
		Condition: cond,
		Values:    ErrorData{gid, v},
	}
	switch cond {
	case NoGroupValueCondition:
	case DuplicateGroupValuesCondition:
	default:
		panic(fmt.Errorf("Unexpected group error condition (%v) in group %v", cond, gid))
	}
	return err
}
