package tblfmt

import (
	"bytes"
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

	// ErrInvalidFormat is the invalid format error.
	ErrInvalidFormat Error = "invalid format"

	// ErrInvalidLineStyle is the invalid line style error.
	ErrInvalidLineStyle Error = "invalid line style"
)

// errEncoder provides a no-op encoder that always returns the wrapped error.
type errEncoder struct {
	err error
}

// Encode satisfies the Encoder interface.
func (err errEncoder) Encode(io.Writer) error {
	return err.err
}

// EncodeAll satisfies the Encoder interface.
func (err errEncoder) EncodeAll(io.Writer) error {
	return err.err
}

// newErrEncoder creates a no-op error encoder.
func newErrEncoder(_ ResultSet, opts ...Option) (Encoder, error) {
	var err error
	enc := &errEncoder{}
	for _, o := range opts {
		if err = o(enc); err != nil {
			return nil, err
		}
	}
	return enc, enc.err
}

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

// max returns the max of a, b.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// condWrite conditionally writes runes to w.
func condWrite(w io.Writer, repeat int, runes ...rune) error {
	var buf []byte
	for _, r := range runes {
		if r != 0 {
			buf = append(buf, []byte(string(r))...)
		}
	}
	if repeat > 1 {
		buf = bytes.Repeat(buf, repeat)
	}
	_, err := w.Write(buf)
	return err
}
