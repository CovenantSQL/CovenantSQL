// Package tblfmt provides streaming table encoders for result sets (ie, from a
// database).
package tblfmt

import (
	"io"
)

// Encoder is the shared interface for encoders.
type Encoder interface {
	Encode(io.Writer) error
	EncodeAll(io.Writer) error
}

// Encode encodes the result set to the writer using the supplied map options.
func Encode(w io.Writer, resultSet ResultSet, options map[string]string) error {
	f, opts := FromMap(options)
	enc, err := f(resultSet, opts...)
	if err != nil {
		return err
	}
	return enc.Encode(w)
}

// EncodeAll encodes all result sets to the writer using the supplied map
// options.
func EncodeAll(w io.Writer, resultSet ResultSet, options map[string]string) error {
	f, opts := FromMap(options)
	enc, err := f(resultSet, opts...)
	if err != nil {
		return err
	}
	return enc.EncodeAll(w)
}

// EncodeTable encodes result set to the writer as a table using the supplied
// encoding options.
func EncodeTable(w io.Writer, resultSet ResultSet, opts ...Option) error {
	enc, err := NewTableEncoder(resultSet, opts...)
	if err != nil {
		return err
	}
	return enc.Encode(w)
}

// EncodeTableAll encodes all result sets to the writer as a table using the
// supplied encoding options.
func EncodeTableAll(w io.Writer, resultSet ResultSet, opts ...Option) error {
	enc, err := NewTableEncoder(resultSet, opts...)
	if err != nil {
		return err
	}
	return enc.EncodeAll(w)
}

// EncodeJSON encodes the result set to the writer as JSON using the supplied
// encoding options.
func EncodeJSON(w io.Writer, resultSet ResultSet, opts ...Option) error {
	enc, err := NewJSONEncoder(resultSet, opts...)
	if err != nil {
		return err
	}
	return enc.Encode(w)
}

// EncodeJSONAll encodes all result sets to the writer as JSON using the
// supplied encoding options.
func EncodeJSONAll(w io.Writer, resultSet ResultSet, opts ...Option) error {
	enc, err := NewJSONEncoder(resultSet, opts...)
	if err != nil {
		return err
	}
	return enc.Encode(w)
}

// EncodeCSV encodes the result set to the writer as CSV using the supplied
// encoding options.
func EncodeCSV(w io.Writer, resultSet ResultSet, opts ...Option) error {
	enc, err := NewCSVEncoder(resultSet, opts...)
	if err != nil {
		return err
	}
	return enc.Encode(w)
}

// EncodeCSVAll encodes all result sets to the writer as CSV using the supplied
// encoding options.
func EncodeCSVAll(w io.Writer, resultSet ResultSet, opts ...Option) error {
	enc, err := NewCSVEncoder(resultSet, opts...)
	if err != nil {
		return err
	}
	return enc.Encode(w)
}

// EncodeTemplate encodes the result set to the writer using a template from
// the supplied encoding options.
func EncodeTemplate(w io.Writer, resultSet ResultSet, opts ...Option) error {
	enc, err := NewTemplateEncoder(resultSet, opts...)
	if err != nil {
		return err
	}
	return enc.Encode(w)
}

// EncodeTemplateAll encodes all result sets to the writer using a template
// from the supplied encoding options.
func EncodeTemplateAll(w io.Writer, resultSet ResultSet, opts ...Option) error {
	enc, err := NewTemplateEncoder(resultSet, opts...)
	if err != nil {
		return err
	}
	return enc.Encode(w)
}
