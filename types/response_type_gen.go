package types

// Code generated by github.com/CovenantSQL/HashStablePack DO NOT EDIT.

import (
	hsp "github.com/CovenantSQL/HashStablePack/marshalhash"
)

// MarshalHash marshals for hash
func (z *Response) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 2
	o = append(o, 0x82, 0x82)
	if oTemp, err := z.Payload.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	o = append(o, 0x82)
	if oTemp, err := z.Header.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *Response) Msgsize() (s int) {
	s = 1 + 8 + z.Payload.Msgsize() + 7 + z.Header.Msgsize()
	return
}

// MarshalHash marshals for hash
func (z *ResponseHeader) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 8
	o = append(o, 0x88, 0x88)
	if oTemp, err := z.Request.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	o = append(o, 0x88)
	if oTemp, err := z.DataHash.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	o = append(o, 0x88)
	o = hsp.AppendInt64(o, z.LastInsertID)
	o = append(o, 0x88)
	o = hsp.AppendInt64(o, z.AffectedRows)
	o = append(o, 0x88)
	if oTemp, err := z.NodeID.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	o = append(o, 0x88)
	o = hsp.AppendTime(o, z.Timestamp)
	o = append(o, 0x88)
	o = hsp.AppendUint64(o, z.RowCount)
	o = append(o, 0x88)
	o = hsp.AppendUint64(o, z.LogOffset)
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *ResponseHeader) Msgsize() (s int) {
	s = 1 + 8 + z.Request.Msgsize() + 9 + z.DataHash.Msgsize() + 13 + hsp.Int64Size + 13 + hsp.Int64Size + 7 + z.NodeID.Msgsize() + 10 + hsp.TimeSize + 9 + hsp.Uint64Size + 10 + hsp.Uint64Size
	return
}

// MarshalHash marshals for hash
func (z *ResponsePayload) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 3
	o = append(o, 0x83, 0x83)
	o = hsp.AppendArrayHeader(o, uint32(len(z.Rows)))
	for za0003 := range z.Rows {
		// map header, size 1
		o = append(o, 0x81, 0x81)
		o = hsp.AppendArrayHeader(o, uint32(len(z.Rows[za0003].Values)))
		for za0004 := range z.Rows[za0003].Values {
			o, err = hsp.AppendIntf(o, z.Rows[za0003].Values[za0004])
			if err != nil {
				return
			}
		}
	}
	o = append(o, 0x83)
	o = hsp.AppendArrayHeader(o, uint32(len(z.Columns)))
	for za0001 := range z.Columns {
		o = hsp.AppendString(o, z.Columns[za0001])
	}
	o = append(o, 0x83)
	o = hsp.AppendArrayHeader(o, uint32(len(z.DeclTypes)))
	for za0002 := range z.DeclTypes {
		o = hsp.AppendString(o, z.DeclTypes[za0002])
	}
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *ResponsePayload) Msgsize() (s int) {
	s = 1 + 5 + hsp.ArrayHeaderSize
	for za0003 := range z.Rows {
		s += 1 + 7 + hsp.ArrayHeaderSize
		for za0004 := range z.Rows[za0003].Values {
			s += hsp.GuessSize(z.Rows[za0003].Values[za0004])
		}
	}
	s += 8 + hsp.ArrayHeaderSize
	for za0001 := range z.Columns {
		s += hsp.StringPrefixSize + len(z.Columns[za0001])
	}
	s += 10 + hsp.ArrayHeaderSize
	for za0002 := range z.DeclTypes {
		s += hsp.StringPrefixSize + len(z.DeclTypes[za0002])
	}
	return
}

// MarshalHash marshals for hash
func (z *ResponseRow) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 1
	o = append(o, 0x81, 0x81)
	o = hsp.AppendArrayHeader(o, uint32(len(z.Values)))
	for za0001 := range z.Values {
		o, err = hsp.AppendIntf(o, z.Values[za0001])
		if err != nil {
			return
		}
	}
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *ResponseRow) Msgsize() (s int) {
	s = 1 + 7 + hsp.ArrayHeaderSize
	for za0001 := range z.Values {
		s += hsp.GuessSize(z.Values[za0001])
	}
	return
}

// MarshalHash marshals for hash
func (z *SignedResponseHeader) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 4
	o = append(o, 0x84, 0x84)
	if z.Signee == nil {
		o = hsp.AppendNil(o)
	} else {
		if oTemp, err := z.Signee.MarshalHash(); err != nil {
			return nil, err
		} else {
			o = hsp.AppendBytes(o, oTemp)
		}
	}
	o = append(o, 0x84)
	if z.Signature == nil {
		o = hsp.AppendNil(o)
	} else {
		if oTemp, err := z.Signature.MarshalHash(); err != nil {
			return nil, err
		} else {
			o = hsp.AppendBytes(o, oTemp)
		}
	}
	o = append(o, 0x84)
	if oTemp, err := z.ResponseHeader.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	o = append(o, 0x84)
	if oTemp, err := z.Hash.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *SignedResponseHeader) Msgsize() (s int) {
	s = 1 + 7
	if z.Signee == nil {
		s += hsp.NilSize
	} else {
		s += z.Signee.Msgsize()
	}
	s += 10
	if z.Signature == nil {
		s += hsp.NilSize
	} else {
		s += z.Signature.Msgsize()
	}
	s += 15 + z.ResponseHeader.Msgsize() + 5 + z.Hash.Msgsize()
	return
}