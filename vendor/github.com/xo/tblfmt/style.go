package tblfmt

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

// ASCIILineStyle is the ASCII line style for tables.
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

// OldASCIILineStyle is the old ASCII line style for tables.
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

// UnicodeLineStyle is the Unicode line style for tables.
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

// UnicodeDoubleLineStyle is the Unicode double line style for tables.
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
