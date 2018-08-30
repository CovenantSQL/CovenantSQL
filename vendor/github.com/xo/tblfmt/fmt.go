package tblfmt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

// Formatter is the common interface for formatting values.
type Formatter interface {
	// Header returns a slice of formatted values for the provided headers.
	Header([]string) ([]*Value, error)

	// Format returns a slice of formatted value the provided row values.
	Format([]interface{}) ([]*Value, error)
}

// EscapeFormatter is an escaping formatter, that handles formatting the
// standard Go types.
//
// If Marshaler is not nil, then it will be passed any map[string]interface{}
// and []interface{} values encountered. If nil, then the standard
// encoding/json.Encoder will be used instead.
type EscapeFormatter struct {
	// mask is used to format header values when the formatted value (after
	// trimming spaces) is the empty string.
	//
	// Note: will have %d replaced with the column number (starting at 1).
	mask string

	// timeFormat is the format to use for time values.
	timeFormat string

	// marshaler will be used to marshal map[string]interface{} and
	// []interface{} types.
	//
	// If nil, the standard encoding/json.Encoder will be used instead.
	marshaler func(interface{}) ([]byte, error)

	// prefix is indent prefix used by the JSON encoder when Marshaler is nil.
	prefix string

	// indent is the indent used by the JSON encoder when Marshaler is nil.
	indent string

	// escapeHTML sets the JSON encoder used when Marshaler is nil to escape HTML
	// characters.
	escapeHTML bool

	// invalid is the value used for invalid utf8 runes when escaping.
	invalid []byte

	// invalidWidth is the rune width of the invalid value
	invalidWidth int
}

// NewEscapeFormatter creates a escape formatter to handle basic Go values,
// such as []byte, string, time.Time. Formatting for map[string]interface{} and
// []interface{} will be passed to a marshaler provided by WithMarshaler,
// otherwise the standard encoding/json.Encoder will be used to marshal those
// values.
func NewEscapeFormatter(opts ...EscapeFormatterOption) *EscapeFormatter {
	f := &EscapeFormatter{
		mask:       "%d",
		timeFormat: time.RFC3339Nano,
		indent:     "  ",
	}
	for _, o := range opts {
		o(f)
	}
	return f
}

// Header satisfies the Formatter interface.
func (f *EscapeFormatter) Header(headers []string) ([]*Value, error) {
	n := len(headers)
	res := make([]*Value, n)
	for i := 0; i < n; i++ {
		s := strings.TrimSpace(headers[i])
		if s == "" {
			s = fmt.Sprintf(f.mask, i+1)
		}
		res[i] = f.escapeString(s)
	}
	return res, nil
}

// Format satisfies the Formatter interface.
func (f *EscapeFormatter) Format(vals []interface{}) ([]*Value, error) {
	n := len(vals)
	res := make([]*Value, n)

	// TODO: change time to v.AppendFormat() + pool
	// TODO: use strconv.Format* for numeric times
	// TODO: use pool
	// TODO: don't use escapeString() for everything
	// TODO: allow configurable runes that can be escaped

	for i := 0; i < n; i++ {
		switch v := (*(vals[i].(*interface{}))).(type) {
		case nil:

		case bool:
			res[i] = f.escapeString(strconv.FormatBool(v))

		case *bool:
			if v != nil {
				res[i] = f.escapeString(strconv.FormatBool(*v))
			}

		// SPECIAL CASE -- should be a single character!
		case uint8:
			res[i] = f.escapeString(string(rune(v)))
		case *uint8:
			if v != nil {
				res[i] = f.escapeString(string(rune(*v)))
			}

		case int:
			res[i] = f.escapeString(strconv.FormatInt(int64(v), 10))
			res[i].Align = AlignRight
		case int8:
			res[i] = f.escapeString(strconv.FormatInt(int64(v), 10))
			res[i].Align = AlignRight
		case int16:
			res[i] = f.escapeString(strconv.FormatInt(int64(v), 10))
			res[i].Align = AlignRight
		case int32:
			res[i] = f.escapeString(strconv.FormatInt(int64(v), 10))
			res[i].Align = AlignRight
		case int64:
			res[i] = f.escapeString(strconv.FormatInt(v, 10))
			res[i].Align = AlignRight

		case *int:
			if v != nil {
				res[i] = f.escapeString(strconv.FormatInt(int64(*v), 10))
				res[i].Align = AlignRight
			}
		case *int8:
			if v != nil {
				res[i] = f.escapeString(strconv.FormatInt(int64(*v), 10))
				res[i].Align = AlignRight
			}
		case *int16:
			if v != nil {
				res[i] = f.escapeString(strconv.FormatInt(int64(*v), 10))
				res[i].Align = AlignRight
			}
		case *int32:
			if v != nil {
				res[i] = f.escapeString(strconv.FormatInt(int64(*v), 10))
				res[i].Align = AlignRight
			}
		case *int64:
			if v != nil {
				res[i] = f.escapeString(strconv.FormatInt(*v, 10))
				res[i].Align = AlignRight
			}

		case uint:
			res[i] = f.escapeString(strconv.FormatUint(uint64(v), 10))
			res[i].Align = AlignRight
		case uint16:
			res[i] = f.escapeString(strconv.FormatUint(uint64(v), 10))
			res[i].Align = AlignRight
		case uint32:
			res[i] = f.escapeString(strconv.FormatUint(uint64(v), 10))
			res[i].Align = AlignRight
		case uint64:
			res[i] = f.escapeString(strconv.FormatUint(v, 10))
			res[i].Align = AlignRight
		case *uint:
			if v != nil {
				res[i] = f.escapeString(strconv.FormatUint(uint64(*v), 10))
				res[i].Align = AlignRight
			}
		case *uint16:
			if v != nil {
				res[i] = f.escapeString(strconv.FormatUint(uint64(*v), 10))
				res[i].Align = AlignRight
			}
		case *uint32:
			if v != nil {
				res[i] = f.escapeString(strconv.FormatUint(uint64(*v), 10))
				res[i].Align = AlignRight
			}
		case *uint64:
			if v != nil {
				res[i] = f.escapeString(strconv.FormatUint(*v, 10))
			}

		case uintptr:
			res[i] = f.escapeString(fmt.Sprintf("(0x%x)", v))
			res[i].Align = AlignRight
		case *uintptr:
			if v != nil {
				res[i] = f.escapeString(fmt.Sprintf("(0x%x)", *v))
				res[i].Align = AlignRight
			}

		case float32:
			res[i] = f.escapeString(strconv.FormatFloat(float64(v), 'g', -1, 32))
			res[i].Align = AlignRight
		case float64:
			res[i] = f.escapeString(strconv.FormatFloat(float64(v), 'g', -1, 64))
			res[i].Align = AlignRight
		case *float32:
			if v != nil {
				res[i] = f.escapeString(strconv.FormatFloat(float64(*v), 'g', -1, 32))
				res[i].Align = AlignRight
			}
		case *float64:
			if v != nil {
				res[i] = f.escapeString(strconv.FormatFloat(float64(*v), 'g', -1, 64))
				res[i].Align = AlignRight
			}

		case complex64:
			res[i] = f.escapeString(fmt.Sprintf("%g", v))
			res[i].Align = AlignRight
		case complex128:
			res[i] = f.escapeString(fmt.Sprintf("%g", v))
			res[i].Align = AlignRight
		case *complex64:
			if v != nil {
				res[i] = f.escapeString(fmt.Sprintf("%g", *v))
				res[i].Align = AlignRight
			}
		case *complex128:
			if v != nil {
				res[i] = f.escapeString(fmt.Sprintf("%g", *v))
				res[i].Align = AlignRight
			}

		case []byte:
			res[i] = f.escapeBytes(v)

		case *[]byte:
			if v != nil {
				res[i] = f.escapeBytes(*v)
			}

		case string:
			res[i] = f.escapeString(v)

		case *string:
			if v != nil {
				res[i] = f.escapeString(*v)
			}

		case time.Time:
			res[i] = f.escapeString(v.Format(f.timeFormat))

		case *time.Time:
			if v != nil {
				res[i] = f.escapeString(v.Format(f.timeFormat))
			}

		case fmt.Stringer:
			res[i] = f.escapeString(v.String())

		default:
			// TODO: pool
			if f.marshaler != nil {
				buf, err := f.marshaler(v)
				if err != nil {
					return nil, err
				}
				res[i] = f.escapeBytes(buf)
			} else {
				// json encode
				buf := new(bytes.Buffer)
				enc := json.NewEncoder(buf)
				enc.SetIndent(f.prefix, f.indent)
				enc.SetEscapeHTML(f.escapeHTML)
				if err := enc.Encode(v); err != nil {
					return nil, err
				}

				res[i] = f.escapeBytes(bytes.TrimSpace(buf.Bytes()))
			}
		}
	}

	return res, nil
}

// escape escapes buf.
//
// TODO: use a pool.
func (f *EscapeFormatter) escapeString(src string) *Value {
	return f.escapeBytes([]byte(src))
}

// escapeBytes escapes bytes.
//
// TODO: use a pool.
func (f *EscapeFormatter) escapeBytes(src []byte) *Value {
	return escape(src, f.invalid, f.invalidWidth)
}

// escape parses src, saving escaped (encoded) and unescaped runes to a Value,
// along with tab and newline positions in the generated buf.
func escape(src []byte, invalid []byte, invalidWidth int) *Value {
	res := &Value{
		Tabs: make([][][2]int, 1),
	}

	var tmp [utf8.MaxRune]byte

	var r rune
	var l, w int
	for ; len(src) > 0; src = src[w:] {
		r, w = rune(src[0]), 1

		// lazy decode
		if r >= utf8.RuneSelf {
			r, w = utf8.DecodeRune(src)
		}

		// invalid rune decoded
		if w == 1 && r == utf8.RuneError {
			// replace with invalid (if set), otherwise hex encode
			if invalid != nil {
				res.Buf = append(res.Buf, invalid...)
				res.Width += invalidWidth
			} else {
				res.Buf = append(res.Buf, '\\', 'x', lowerhex[src[0]>>4], lowerhex[src[0]&0xf])
				res.Width += 4
			}
			continue
		}

		// printable character
		if strconv.IsGraphic(r) {
			n := utf8.EncodeRune(tmp[:], r)
			res.Buf = append(res.Buf, tmp[:n]...)
			res.Width += runewidth.RuneWidth(r)
			continue
		}

		switch r {
		// escape \a \b \f \r \v (Go special characters)
		case '\a':
			res.Buf = append(res.Buf, '\\', 'a')
			res.Width += 2
		case '\b':
			res.Buf = append(res.Buf, '\\', 'b')
			res.Width += 2
		case '\f':
			res.Buf = append(res.Buf, '\\', 'f')
			res.Width += 2
		case '\r':
			res.Buf = append(res.Buf, '\\', 'r')
			res.Width += 2
		case '\v':
			res.Buf = append(res.Buf, '\\', 'v')
			res.Width += 2

		case '\t':
			// save position
			res.Tabs[l] = append(res.Tabs[l], [2]int{len(res.Buf), res.Width})
			res.Buf = append(res.Buf, '\t')
			res.Width = 0

		case '\n':
			// save position
			res.Newlines = append(res.Newlines, [2]int{len(res.Buf), res.Width})
			res.Buf = append(res.Buf, '\n')
			res.Width = 0

			// increase line count
			res.Tabs = append(res.Tabs, nil)
			l++

		default:
			switch {
			// escape as \x00
			case r < ' ':
				res.Buf = append(res.Buf, '\\', 'x', lowerhex[byte(r)>>4], lowerhex[byte(r)&0xf])
				res.Width += 4

			// escape as \u0000
			case r > utf8.MaxRune:
				r = 0xfffd
				fallthrough
			case r < 0x10000:
				res.Buf = append(res.Buf, '\\', 'u')
				for s := 12; s >= 0; s -= 4 {
					res.Buf = append(res.Buf, lowerhex[r>>uint(s)&0xf])
				}
				res.Width += 6

			// escape as \U00000000
			default:
				res.Buf = append(res.Buf, '\\', 'U')
				for s := 28; s >= 0; s -= 4 {
					res.Buf = append(res.Buf, lowerhex[r>>uint(s)&0xf])
				}
				res.Width += 10
			}
		}
	}

	return res
}

// Value contains information pertaining to a formatted value.
type Value struct {
	// Buf is the formatted value.
	Buf []byte

	// Newlines are the positions of newline characters in Buf.
	Newlines [][2]int

	// Tabs are the positions of tab characters in Buf, split per line.
	Tabs [][][2]int

	// Width is the remaining width.
	Width int

	// Align indicates value alignment.
	Align Align
}

// LineWidth returns the line width (in runes) of line l.
func (v *Value) LineWidth(l, offset, tab int) int {
	var width int
	if l < len(v.Newlines) {
		width += v.Newlines[l][1]
	}
	if len(v.Tabs[l]) != 0 {
		width += tabwidth(v.Tabs[l], offset, tab)
	}
	if l == len(v.Newlines) {
		width += v.Width
	}
	return width
}

// MaxWidth calculates the maximum width (in runes) of the longest line
// contained in Buf, relative to starting offset and the tab width.
func (v *Value) MaxWidth(offset, tab int) int {
	var width int
	for l := 0; l < len(v.Tabs); l++ {
		width = max(width, v.LineWidth(l, offset, tab))
	}
	return width
}

// Align indicates an alignment direction for a value.
type Align int

// Align values.
const (
	AlignLeft Align = iota
	AlignRight
	AlignCenter
)

// String satisfies the fmt.Stringer interface.
func (a Align) String() string {
	switch a {
	case AlignLeft:
		return "Left"
	case AlignRight:
		return "Right"
	case AlignCenter:
		return "Center"
	}
	return fmt.Sprintf("Align(%d)", a)
}

// tabwidth returns the rune width of buf containing tabs from start position
// in buf, a column offset, and given tab width.
func tabwidth(tabs [][2]int, offset, tab int) int {
	//log.Printf("tabs: %v, offset: %d, tab: %d", tabs, offset, tab)
	width := offset
	for i := 0; i < len(tabs); i++ {
		width += tabs[i][1]
		width += (tab - width%tab)
	}
	//log.Printf("res: %d", width-offset)
	return width - offset
}

// EscapeFormatterOption is an escape formatter option.
type EscapeFormatterOption func(*EscapeFormatter)

// WithMask is an escape formatter option to set the mask used for empty
// headers.
func WithMask(mask string) EscapeFormatterOption {
	return func(f *EscapeFormatter) {
		f.mask = mask
	}
}

// WithTimeFormat is an escape formatter option to set the time format used for
// time values.
func WithTimeFormat(timeFormat string) EscapeFormatterOption {
	return func(f *EscapeFormatter) {
		f.timeFormat = timeFormat
	}
}

// WithMarshaler is an escape formatter option to set a standard Go encoder to
// use for encoding the value.
func WithMarshaler(marshaler func(interface{}) ([]byte, error)) EscapeFormatterOption {
	return func(f *EscapeFormatter) {
		f.marshaler = marshaler
	}
}

// WithJSONConfig is an escape formatter option to set the JSON encoding
// prefix, indent value, and whether or not to escape HTML. Passed to the
// standard encoding/json.Encoder when a marshaler has not been set on the
// escape formatter.
func WithJSONConfig(prefix, indent string, escapeHTML bool) EscapeFormatterOption {
	return func(f *EscapeFormatter) {
		f.prefix, f.indent, f.escapeHTML = prefix, indent, escapeHTML
	}
}

// WithInvalid is an escape formatter option to set the invalid value used when
// an invalid rune is encountered during escaping.
func WithInvalid(invalid string) EscapeFormatterOption {
	return func(f *EscapeFormatter) {
		f.invalid = []byte(invalid)
		f.invalidWidth = runewidth.StringWidth(invalid)
	}
}
