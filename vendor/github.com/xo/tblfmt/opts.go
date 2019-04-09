package tblfmt

import (
	"io"
	"strconv"
	"unicode/utf8"
)

// Builder is the shared builder interface.
type Builder func(ResultSet, ...Option) (Encoder, error)

// Option is a Encoder option.
type Option func(interface{}) error

// FromMap creates an encoder for the provided result set, applying the named
// options.
func FromMap(opts map[string]string) (Builder, []Option) {
	// unaligned, aligned, wrapped, html, asciidoc, latex, latex-longtable, troff-ms, json, csv
	switch opts["format"] {
	case "json":
		return NewJSONEncoder, nil

	case "csv":
		var csvOpts []Option
		if s, ok := opts["fieldsep"]; ok {
			sep, _ := utf8.DecodeRuneInString(s)
			csvOpts = append(csvOpts, WithFieldSeparator(sep))
		}
		return NewCSVEncoder, csvOpts

	case "html", "asciidoc", "latex", "latex-longtable", "troff-ms":
		//return newErrEncoder, []Option{withError(fmt.Errorf("%q format not implemented", opts["format"]))}
		return NewTemplateEncoder, []Option{WithNamedTemplate(opts["format"])}

	case "unaligned":
		fallthrough

	case "aligned":
		var tableOpts []Option
		if s, ok := opts["border"]; ok {
			border, _ := strconv.Atoi(s)
			tableOpts = append(tableOpts, WithBorder(border))
		}
		if s, ok := opts["linestyle"]; ok {
			switch s {
			case "ascii":
				tableOpts = append(tableOpts, WithLineStyle(ASCIILineStyle()))
			case "old-ascii":
				tableOpts = append(tableOpts, WithLineStyle(OldASCIILineStyle()))
			case "unicode":
				switch opts["unicode_border_linestyle"] {
				case "single":
					tableOpts = append(tableOpts, WithLineStyle(UnicodeLineStyle()))
				case "double":
					tableOpts = append(tableOpts, WithLineStyle(UnicodeDoubleLineStyle()))
				}
			}
		}
		return NewTableEncoder, tableOpts

	default:
		return newErrEncoder, []Option{withError(ErrInvalidFormat)}
	}
}

// WithCount is a encoder option to set the buffered line count.
func WithCount(count int) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *TableEncoder:
			enc.count = count
		}
		return nil
	}
}

// WithLineStyle is a encoder option to set the table line style.
func WithLineStyle(lineStyle LineStyle) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *TableEncoder:
			enc.lineStyle = lineStyle
		}
		return nil
	}
}

// WithFormatter is a encoder option to set a formatter for formatting values.
func WithFormatter(formatter Formatter) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *TableEncoder:
			enc.formatter = formatter
		}
		return nil
	}
}

// WithSummary is a encoder option to set a summary callback map.
func WithSummary(summary map[int]func(io.Writer, int) (int, error)) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *TableEncoder:
			enc.summary = summary
		}
		return nil
	}
}

// WithInline is a encoder option to set the column headers as inline to the
// top line.
func WithInline(inline bool) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *TableEncoder:
			enc.inline = inline
		}
		return nil
	}
}

// WithTitle is a encoder option to set the title value used.
func WithTitle(title string) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *TableEncoder:
			vals, err := enc.formatter.Header([]string{title})
			if err != nil {
				return err
			}
			enc.title = vals[0]
		}
		return nil
	}
}

// WithEmpty is a encoder option to set the value used in empty (nil)
// cells.
func WithEmpty(empty string) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *TableEncoder:
			cell := interface{}(empty)
			v, err := enc.formatter.Format([]interface{}{&cell})
			if err != nil {
				return err
			}
			enc.empty = v[0]
		}
		return nil
	}
}

// WithWidths is a encoder option to set (minimum) widths for a column.
func WithWidths(widths []int) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *TableEncoder:
			enc.widths = widths
		}
		return nil
	}
}

// WithNewline is a encoder option to set the newline.
func WithNewline(newline string) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *TableEncoder:
			enc.newline = []byte(newline)
		case *JSONEncoder:
			enc.newline = []byte(newline)
		case *CSVEncoder:
			enc.newline = []byte(newline)
		case *TemplateEncoder:
			enc.newline = []byte(newline)
		}
		return nil
	}
}

// WithFieldSeparator is a encoder option to set the field separator.
func WithFieldSeparator(fieldsep rune) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *CSVEncoder:
			enc.fieldsep = fieldsep
		}
		return nil
	}
}

// WithBorder is a encoder option to set the border size.
func WithBorder(border int) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *TableEncoder:
			enc.border = border
		}
		return nil
	}
}

// WithTemplate is a encoder option to set the raw template used.
func WithTemplate(template string) Option {
	return func(v interface{}) error {
		switch v.(type) {
		case *TemplateEncoder:
		}
		return nil
	}
}

// WithNamedTemplate is a encoder option to set the template used.
func WithNamedTemplate(name string) Option {
	return func(v interface{}) error {
		switch v.(type) {
		case *TemplateEncoder:
		}
		return nil
	}
}

// withError is a encoder option to force an error.
func withError(err error) Option {
	return func(v interface{}) error {
		switch enc := v.(type) {
		case *errEncoder:
			enc.err = err
		}
		return err
	}
}
