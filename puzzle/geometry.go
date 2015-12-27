package puzzle

/*

Puzzle Geometries

In this module, there is only one puzzle implementation, but it
supports multiple geometries whose only difference is the shape
and number of the groups.

*/

// A group descriptor identifies a group and enumerates the
// indices of its squares.
type groupDescriptor struct {
	index   int
	id      GroupID
	indices intset
}

// A puzzleMapping summarizes the geometry parameters of the
// puzzle, including specifically the indexes in each of the
// groups, and a mapping from each index to the groups that
// contain it.
type puzzleMapping struct {
	geometry string
	sidelen  int
	scount   int
	gcount   int
	gdescs   []groupDescriptor
	ixmap    [][]int
}

/*

Registered geometries

*/

const (
	SudokuGeometryName = "sudoku"
	DudokuGeometryName = "dudoku"
)

// knownGeometries is the lookup table for constructors
var knownGeometries = map[string]func([]int) (*Puzzle, error){
	"":                 newSudokuPuzzle,
	SudokuGeometryName: newSudokuPuzzle,
	DudokuGeometryName: newDudokuPuzzle,
}

// newSudokuPuzzle creates a Sudoku puzzle from the given values
func newSudokuPuzzle(values []int) (*Puzzle, error) {
	mapping, err := squarePuzzleMapping(len(values))
	if err != nil {
		return nil, err
	}
	return create(mapping, values)
}

// newDudokuPuzzle creates a Dudoku puzzle from the given values
func newDudokuPuzzle(values []int) (*Puzzle, error) {
	mapping, err := rectanglePuzzleMapping(len(values))
	if err != nil {
		return nil, err
	}
	return create(mapping, values)
}

/*

Sudoku (aka square) Geometry

*/

// squarePuzzleMaps is where we memoize computed square puzzle
// maps for each side length we've encountered, to avoid
// computing them more than once.
var squarePuzzleMaps = make(map[int]*puzzleMapping)

// Find the integer square root of val, if it exists.
func findIntSquareRoot(val int) (int, bool) {
	var i int
	for i = 1; i*i <= val; i++ {
		if i*i == val {
			return i, true
		}
	}
	return i - 1, false
}

func computeSquarePuzzleMapping(slen, tlen int) *puzzleMapping {
	gcount := (slen * 3)
	scount := (slen * slen)
	gs := make([]groupDescriptor, gcount+1) // 1-based indexing
	im := make([][]int, scount+1)           // 1-based indexing
	for i := 1; i <= scount; i++ {
		im[i] = make([]int, 3) // 3 groups for every square
	}
	for i := 0; i < slen; i++ {
		// row i + 1
		rgi := i + 1 // 1-based indexes
		row := make(intset, slen)
		for ri := 0; ri < slen; ri++ {
			si := slen*i + ri + 1 // 1-based indexes
			row[ri] = si
			im[si][0] = rgi
		}
		gs[rgi] = groupDescriptor{rgi, GroupID{GtypeRow, i + 1}, row}
		// column i + 1
		cgi := i + slen + 1 // 1-based indices
		col := make(intset, slen)
		for ci := 0; ci < slen; ci++ {
			si := slen*ci + i + 1 // 1-based indices
			col[ci] = si
			im[si][1] = cgi
		}
		gs[cgi] = groupDescriptor{cgi, GroupID{GtypeCol, i + 1}, col}
		// tile i + 1
		tgi := i + 2*slen + 1 // 1-based indices
		tile := make(intset, slen)
		baserow, basecol := tlen*(i/tlen), tlen*(i%tlen)
		for tri := 0; tri < tlen; tri++ {
			for tci := 0; tci < tlen; tci++ {
				si := slen*(baserow+tri) + (basecol + tci) + 1 // 1-based indices
				tile[tri*tlen+tci] = si
				im[si][2] = tgi
			}
		}
		gs[tgi] = groupDescriptor{tgi, GroupID{GtypeTile, i + 1}, tile}
	}
	return &puzzleMapping{SudokuGeometryName, slen, scount, gcount, gs, im}
}

// squarePuzzleMapping returns the puzzle map for a square puzzle
// with the given number of cells.  This computes (first time)
// and then returns (thereafter) the map.  Returns an error if
// the sidelength is not a perfect square.
func squarePuzzleMapping(psize int) (*puzzleMapping, error) {
	sidelen, ok := findIntSquareRoot(psize)
	if !ok {
		return nil, formatError(PuzzleSizeAttribute, psize, NonSquareCondition, 0)
	}
	min, max := 4, 225 // largest that fits in a btye
	if sidelen < min {
		return nil, formatError(SideLengthAttribute, sidelen, TooSmallCondition, min)
	}
	if sidelen > max {
		return nil, formatError(SideLengthAttribute, sidelen, TooLargeCondition, max)
	}
	tilelen, ok := findIntSquareRoot(sidelen)
	if !ok {
		return nil, formatError(SideLengthAttribute, sidelen, NonSquareCondition, 0)
	}
	pm, ok := squarePuzzleMaps[sidelen]
	if ok {
		return pm, nil
	}
	pm = computeSquarePuzzleMapping(sidelen, tilelen)
	squarePuzzleMaps[sidelen] = pm
	return pm, nil
}

/*

// play.golang.org section to figure out max sizes for standard geometry
// by considering how byte-value compression will work given tile size.

import "fmt"

func main() {
	fmt.Printf("sl\tmv\tmd\n")
	for tilelen := byte(2); tilelen < 15; tilelen++ {
		var sidelen, nibWidth, maxNibVal byte = tilelen * tilelen, 3, 7 // min side length is 4
		for maxNibVal < sidelen+(sidelen/2) {
			nibWidth, maxNibVal = nibWidth+1, maxNibVal*2+1
		}
		maxdelta := maxNibVal - sidelen // biggest index delta we can represent
		fmt.Printf("%d\t%d\t%d\n", sidelen, nibWidth, maxdelta)
	}
}

// results - pick 13 as largest allowed tile size, 169 as side length

// sl	mv	md
// 4	3	3
// 9	4	6
// 16	5	15
// 25	6	38
// 36	6	27
// 49	7	78
// 64	7	63
// 81	7	46
// 100	8	155
// 121	8	134
// 144	8	111
// 169	8	86
// 196	6	123

*/

/*

Rectangular puzzles (aka DuDoku)

*/

// rectanglePuzzleMaps is where we memoize computed rectangle
// puzzle maps for each side length we've encountered, to avoid
// computing them more than once.
var rectanglePuzzleMaps = make(map[int]*puzzleMapping)

// findDivisors: find consecutive ints that multiply to give an
// int, if they exist
func findDivisors(val int) (int, int, bool) {
	var low, high int
	for low, high = 1, 2; low*high <= val; low, high = high, high+1 {
		if low*high == val {
			return low, high, true
		}
	}
	return low - 1, low, false
}

func computeRectanglePuzzleMapping(slen, low, high int) *puzzleMapping {
	gcount := (slen * 3)
	scount := (slen * slen)
	gs := make([]groupDescriptor, gcount+1) // 1-based indexing
	im := make([][]int, scount+1)           // 1-based indexing
	for i := 1; i <= scount; i++ {
		im[i] = make([]int, 3) // 3 groups for every square
	}
	for i := 0; i < slen; i++ {
		// row i + 1
		rgi := i + 1 // 1-based indexes
		row := make(intset, slen)
		for ri := 0; ri < slen; ri++ {
			si := slen*i + ri + 1 // 1-based indexes
			row[ri] = si
			im[si][0] = rgi
		}
		gs[rgi] = groupDescriptor{rgi, GroupID{GtypeRow, i + 1}, row}
		// column i + 1
		cgi := i + slen + 1 // 1-based indices
		col := make(intset, slen)
		for ci := 0; ci < slen; ci++ {
			si := slen*ci + i + 1 // 1-based indices
			col[ci] = si
			im[si][1] = cgi
		}
		gs[cgi] = groupDescriptor{cgi, GroupID{GtypeCol, i + 1}, col}
		// tile i + 1
		tgi := i + 2*slen + 1 // 1-based indices
		tile := make(intset, slen)
		baserow, basecol := low*(i/low), high*(i%low)
		for tri := 0; tri < low; tri++ {
			for tci := 0; tci < high; tci++ {
				si := slen*(baserow+tri) + (basecol + tci) + 1 // 1-based indices
				tile[tri*high+tci] = si
				im[si][2] = tgi
			}
		}
		gs[tgi] = groupDescriptor{tgi, GroupID{GtypeTile, i + 1}, tile}
	}
	return &puzzleMapping{DudokuGeometryName, slen, scount, gcount, gs, im}
}

// rectanglePuzzleMapping returns the puzzle map for a square puzzle
// with the given number of cells.  This computes (first time)
// and then returns (thereafter) the map.  Returns an error if
// the sidelength is not a perfect square.
func rectanglePuzzleMapping(psize int) (*puzzleMapping, error) {
	sidelen, ok := findIntSquareRoot(psize)
	if !ok {
		return nil, formatError(PuzzleSizeAttribute, psize, NonSquareCondition, 0)
	}
	min, max := 6, 240 // largest that fits in a byte
	if sidelen < min {
		return nil, formatError(SideLengthAttribute, sidelen, TooSmallCondition, min)
	}
	if sidelen > max {
		return nil, formatError(SideLengthAttribute, sidelen, TooLargeCondition, max)
	}
	low, high, ok := findDivisors(sidelen)
	if !ok {
		return nil, formatError(SideLengthAttribute, sidelen, NonRectangleCondition, 0)
	}
	pm, ok := rectanglePuzzleMaps[sidelen]
	if ok {
		return pm, nil
	}
	pm = computeRectanglePuzzleMapping(sidelen, low, high)
	rectanglePuzzleMaps[sidelen] = pm
	return pm, nil
}

/*

Errors

*/

func formatError(attr ErrorAttribute, val int, cond ErrorCondition, limit int) Error {
	err := Error{
		Scope:     GeometryScope,
		Structure: AttributeValueStructure,
		Attribute: attr,
		Condition: cond,
		Values:    ErrorData{val},
	}
	if cond == TooSmallCondition || cond == TooLargeCondition {
		err.Values = append(err.Values, limit)
	}
	return err
}
