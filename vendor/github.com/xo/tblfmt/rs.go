// +build !go1.8

package tblfmt

import (
	"io"
)

// ResultSet is the shared interface for a result set.
type ResultSet interface {
	Next() bool
	Scan(...interface{}) error
	Columns() ([]string, error)
	Close() error
	Err() error
}

// EncodeAll encodes all result sets to the writer using the encoder settings.
func (enc *TableEncoder) EncodeAll(w io.Writer) error {
	var err error

	if err = enc.Encode(w); err != nil {
		return err
	}

	_, err = w.Write(enc.newline)
	return err
}

// EncodeAll encodes all result sets to the writer using the encoder settings.
func (enc *JSONEncoder) EncodeAll(w io.Writer) error {
	var err error

	if err = enc.Encode(w); err != nil {
		return err
	}

	_, err = w.Write(enc.newline)
	return err
}

// EncodeAll encodes all result sets to the writer using the encoder settings.
func (enc *CSVEncoder) EncodeAll(w io.Writer) error {
	var err error

	if err = enc.Encode(w); err != nil {
		return err
	}

	_, err = w.Write(enc.newline)
	return err
}

// EncodeAll encodes all result sets to the writer using the encoder settings.
func (enc *TemplateEncoder) EncodeAll(w io.Writer) error {
	var err error

	if err = enc.Encode(w); err != nil {
		return err
	}

	_, err = w.Write(enc.newline)
	return err
}
