package tblfmt

import (
	"fmt"
	"io"
	"runtime"
)

const lowerhex = "0123456789abcdef"

// newline is the default newline used by the system.
var newline []byte

func init() {
	if runtime.GOOS == "windows" {
		newline = []byte("\r\n")
	} else {
		newline = []byte("\n")
	}
}

// Error is an error.
type Error string

// Error satisfies the error interface.
func (err Error) Error() string {
	return string(err)
}

// Error values.
const (
	// ErrResultSetIsNil is the result set is nil error.
	ErrResultSetIsNil Error = "result set is nil"

	// ErrResultSetHasNoColumns is the result set has no columns error.
	ErrResultSetHasNoColumns Error = "result set has no columns"
)

// DefaultTableSummary is the default table summary.
func DefaultTableSummary() map[int]func(io.Writer, int) (int, error) {
	return map[int]func(io.Writer, int) (int, error){
		1: func(w io.Writer, count int) (int, error) {
			return fmt.Fprintf(w, "(%d row)", count)
		},
		-1: func(w io.Writer, count int) (int, error) {
			return fmt.Fprintf(w, "(%d rows)", count)
		},
	}
}

// LineStyle is a table line style.
//
// See the ASCII, OldASCII, and Unicode styles below for predefined table
// styles.
//
// Tables generally look like the following:
//
//    +-----------+---------------------------+---+
//    | author_id |           name            | z |
//    +-----------+---------------------------+---+
//    |        14 | a       b       c       d |   |
//    |        15 | aoeu                     +|   |
//    |           | test                     +|   |
//    |           |                           |   |
//    +-----------+---------------------------+---+
//
// When border is 0, then no surrounding borders will be shown:
//
//    author_id           name            z
//    --------- ------------------------- -
//           14 a       b       c       d
//           15 aoeu                     +
//              test                     +
//
// When border is 1, then a border between columns will be shown:
//
//     author_id |           name            | z
//    -----------+---------------------------+---
//            14 | a       b       c       d |
//            15 | aoeu                     +|
//               | test                     +|
//               |                           |
//
type LineStyle struct {
	Top  [4]rune
	Mid  [4]rune
	Row  [4]rune
	Wrap [4]rune
	End  [4]rune
}

// ASCIILineStyle is the ascii line style for tables.
//
// Tables using this style will look like the following:
//
//    +-----------+---------------------------+---+
//    | author_id |           name            | z |
//    +-----------+---------------------------+---+
//    |        14 | a       b       c       d |   |
//    |        15 | aoeu                     +|   |
//    |           | test                     +|   |
//    |           |                           |   |
//    +-----------+---------------------------+---+
//
func ASCIILineStyle() LineStyle {
	return LineStyle{
		// left char sep right
		Top:  [4]rune{'+', '-', '+', '+'},
		Mid:  [4]rune{'+', '-', '+', '+'},
		Row:  [4]rune{'|', ' ', '|', '|'},
		Wrap: [4]rune{'|', '+', '|', '|'},
		End:  [4]rune{'+', '-', '+', '+'},
	}
}

// OldASCIILineStyle is the old ascii line style for tables.
//
// Tables using this style will look like the following:
//
//    +-----------+---------------------------+---+
//    | author_id |           name            | z |
//    +-----------+---------------------------+---+
//    |        14 | a       b       c       d |   |
//    |        15 | aoeu                      |   |
//    |           : test                          |
//    |           :                               |
//    +-----------+---------------------------+---+
//
func OldASCIILineStyle() LineStyle {
	s := ASCIILineStyle()
	s.Wrap[1], s.Wrap[2] = ' ', ':'
	return s
}

// UnicodeLineStyle is the unicode line style for tables.
//
// Tables using this style will look like the following:
//
//    ┌───────────┬───────────────────────────┬───┐
//    │ author_id │           name            │ z │
//    ├───────────┼───────────────────────────┼───┤
//    │        14 │ a       b       c       d │   │
//    │        15 │ aoeu                     ↵│   │
//    │           │ test                     ↵│   │
//    │           │                           │   │
//    └───────────┴───────────────────────────┴───┘
//
func UnicodeLineStyle() LineStyle {
	return LineStyle{
		// left char sep right
		Top:  [4]rune{'┌', '─', '┬', '┐'},
		Mid:  [4]rune{'├', '─', '┼', '┤'},
		Row:  [4]rune{'│', ' ', '│', '│'},
		Wrap: [4]rune{'│', '↵', '│', '│'},
		End:  [4]rune{'└', '─', '┴', '┘'},
	}
}

// UnicodeDoubleLineStyle is the unicode double line style for tables.
//
// Tables using this style will look like the following:
//
//    ╔═══════════╦═══════════════════════════╦═══╗
//    ║ author_id ║           name            ║ z ║
//    ╠═══════════╬═══════════════════════════╬═══╣
//    ║        14 ║ a       b       c       d ║   ║
//    ║        15 ║ aoeu                     ↵║   ║
//    ║           ║ test                     ↵║   ║
//    ║           ║                           ║   ║
//    ╚═══════════╩═══════════════════════════╩═══╝
//
func UnicodeDoubleLineStyle() LineStyle {
	return LineStyle{
		// left char sep right
		Top:  [4]rune{'╔', '═', '╦', '╗'},
		Mid:  [4]rune{'╠', '═', '╬', '╣'},
		Row:  [4]rune{'║', ' ', '║', '║'},
		Wrap: [4]rune{'║', '↵', '║', '║'},
		End:  [4]rune{'╚', '═', '╩', '╝'},
	}
}

// max returns the max of a, b.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
