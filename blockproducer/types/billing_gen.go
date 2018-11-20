package types

// Code generated by github.com/CovenantSQL/HashStablePack DO NOT EDIT.

import (
	hsp "github.com/CovenantSQL/HashStablePack/marshalhash"
)

// MarshalHash marshals for hash
func (z *Billing) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 3
	o = append(o, 0x83, 0x83)
	if oTemp, err := z.BillingHeader.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	o = append(o, 0x83)
	if oTemp, err := z.TransactionTypeMixin.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	o = append(o, 0x83)
	if oTemp, err := z.DefaultHashSignVerifierImpl.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *Billing) Msgsize() (s int) {
	s = 1 + 14 + z.BillingHeader.Msgsize() + 21 + z.TransactionTypeMixin.Msgsize() + 28 + z.DefaultHashSignVerifierImpl.Msgsize()
	return
}

// MarshalHash marshals for hash
func (z *BillingHeader) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 6
	o = append(o, 0x86, 0x86)
	if oTemp, err := z.BillingRequest.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	o = append(o, 0x86)
	o = hsp.AppendArrayHeader(o, uint32(len(z.Receivers)))
	for za0001 := range z.Receivers {
		if z.Receivers[za0001] == nil {
			o = hsp.AppendNil(o)
		} else {
			if oTemp, err := z.Receivers[za0001].MarshalHash(); err != nil {
				return nil, err
			} else {
				o = hsp.AppendBytes(o, oTemp)
			}
		}
	}
	o = append(o, 0x86)
	o = hsp.AppendArrayHeader(o, uint32(len(z.Fees)))
	for za0002 := range z.Fees {
		o = hsp.AppendUint64(o, z.Fees[za0002])
	}
	o = append(o, 0x86)
	o = hsp.AppendArrayHeader(o, uint32(len(z.Rewards)))
	for za0003 := range z.Rewards {
		o = hsp.AppendUint64(o, z.Rewards[za0003])
	}
	o = append(o, 0x86)
	if oTemp, err := z.Nonce.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	o = append(o, 0x86)
	if oTemp, err := z.Producer.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *BillingHeader) Msgsize() (s int) {
	s = 1 + 15 + z.BillingRequest.Msgsize() + 10 + hsp.ArrayHeaderSize
	for za0001 := range z.Receivers {
		if z.Receivers[za0001] == nil {
			s += hsp.NilSize
		} else {
			s += z.Receivers[za0001].Msgsize()
		}
	}
	s += 5 + hsp.ArrayHeaderSize + (len(z.Fees) * (hsp.Uint64Size)) + 8 + hsp.ArrayHeaderSize + (len(z.Rewards) * (hsp.Uint64Size)) + 6 + z.Nonce.Msgsize() + 9 + z.Producer.Msgsize()
	return
}
