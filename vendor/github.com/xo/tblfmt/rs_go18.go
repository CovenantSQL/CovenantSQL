// +build go1.8

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
	NextResultSet() bool
}

// EncodeAll encodes all result sets to the writer using the encoder settings.
func (enc *TableEncoder) EncodeAll(w io.Writer) error {
	var err error

	if err = enc.Encode(w); err != nil {
		return err
	}

	for enc.resultSet.NextResultSet() {
		if _, err = w.Write(enc.newline); err != nil {
			return err
		}

		if err = enc.Encode(w); err != nil {
			return err
		}
	}

	if _, err = w.Write(enc.newline); err != nil {
		return err
	}

	return nil
}

// EncodeAll encodes all result sets to the writer using the encoder settings.
func (enc *JSONEncoder) EncodeAll(w io.Writer) error {
	var err error

	if err = enc.Encode(w); err != nil {
		return err
	}

	for enc.resultSet.NextResultSet() {
		if _, err = w.Write([]byte{','}); err != nil {
			return err
		}

		if _, err = w.Write(enc.newline); err != nil {
			return err
		}

		if err = enc.Encode(w); err != nil {
			return err
		}
	}

	if _, err = w.Write(enc.newline); err != nil {
		return err
	}

	return nil
}

// EncodeAll encodes all result sets to the writer using the encoder settings.
func (enc *CSVEncoder) EncodeAll(w io.Writer) error {
	var err error

	if err = enc.Encode(w); err != nil {
		return err
	}

	for enc.resultSet.NextResultSet() {
		if _, err = w.Write(enc.newline); err != nil {
			return err
		}

		if err = enc.Encode(w); err != nil {
			return err
		}
	}

	if _, err = w.Write(enc.newline); err != nil {
		return err
	}

	return nil
}

// EncodeAll encodes all result sets to the writer using the encoder settings.
func (enc *TemplateEncoder) EncodeAll(w io.Writer) error {
	var err error

	if err = enc.Encode(w); err != nil {
		return err
	}

	for enc.resultSet.NextResultSet() {
		if _, err = w.Write(enc.newline); err != nil {
			return err
		}
		if err = enc.Encode(w); err != nil {
			return err
		}
	}

	if _, err = w.Write(enc.newline); err != nil {
		return err
	}

	return nil
}
