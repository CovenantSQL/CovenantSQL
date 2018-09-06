// Package tblfmt provides encoders for streaming / writing rows of data.
package tblfmt

import (
	"io"
)

// EncodeTable uses the DefaultTableEncoder to encode the result set to the
// writer.
func EncodeTable(w io.Writer, rs ResultSet, opts ...TableEncoderOption) error {
	return NewTableEncoder(rs, opts...).Encode(w)
}
