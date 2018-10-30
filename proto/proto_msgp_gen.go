package proto

import (
	pkg2_elliptic "crypto/elliptic"
	"errors"
	pkg3_big "math/big"
	"runtime"
	"strconv"
	"time"

	pkg1_asymmetric "github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	pkg5_hash "github.com/CovenantSQL/CovenantSQL/crypto/hash"
	pkg4_cpuminer "github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	codec1978 "github.com/ugorji/go/codec"
)

const (
	// ----- content types ----
	codecSelferCcUTF89172 = 1
	codecSelferCcRAW9172  = 0
	// ----- value types used ----
	codecSelferValueTypeArray9172  = 10
	codecSelferValueTypeMap9172    = 9
	codecSelferValueTypeString9172 = 6
	codecSelferValueTypeInt9172    = 2
	codecSelferValueTypeUint9172   = 3
	codecSelferValueTypeFloat9172  = 4
	codecSelferBitsize9172         = uint8(32 << (^uint(0) >> 63))
)

var (
	errCodecSelferOnlyMapOrArrayEncodeToStruct9172 = errors.New(`only encoded map or array can be decoded into a struct`)
)

type codecSelfer9172 struct{}

func init() {
	if codec1978.GenVersion != 8 {
		_, file, _, _ := runtime.Caller(0)
		panic("codecgen version mismatch: current: 8, need " + strconv.FormatInt(int64(codec1978.GenVersion), 10) + ". Re-generate file: " + file)
	}
	if false { // reference the types, but skip this branch at build/run time
		var v0 pkg2_elliptic.Curve
		var v1 pkg1_asymmetric.PublicKey
		var v2 pkg5_hash.Hash
		var v3 pkg4_cpuminer.Uint256
		var v4 pkg3_big.Int
		var v5 time.Duration
		_, _, _, _, _, _ = v0, v1, v2, v3, v4, v5
	}
}

func (x *PingReq) CodecEncodeSelf(e *codec1978.Encoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperEncoder(e)
	_, _, _ = h, z, r
	if x == nil {
		r.EncodeNil()
	} else {
		if false {
		} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
			z.EncExtension(x, yyxt1)
		} else {
			yysep2 := !z.EncBinary()
			yy2arr2 := z.EncBasicHandle().StructToArray
			_, _ = yysep2, yy2arr2
			const yyr2 bool = false // struct tag has 'toArray'
			if yyr2 || yy2arr2 {
				r.WriteArrayStart(5)
			} else {
				r.WriteMapStart(5)
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				yy4 := &x.Node
				yy4.CodecEncodeSelf(e)
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `Node`)
				r.WriteMapElemValue()
				yy6 := &x.Node
				yy6.CodecEncodeSelf(e)
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if false {
				} else {
					r.EncodeString(codecSelferCcUTF89172, string(x.Version))
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `v`)
				r.WriteMapElemValue()
				if false {
				} else {
					r.EncodeString(codecSelferCcUTF89172, string(x.Version))
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if false {
				} else if yyxt12 := z.Extension(z.I2Rtid(x.TTL)); yyxt12 != nil {
					z.EncExtension(x.TTL, yyxt12)
				} else {
					r.EncodeInt(int64(x.TTL))
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `t`)
				r.WriteMapElemValue()
				if false {
				} else if yyxt13 := z.Extension(z.I2Rtid(x.TTL)); yyxt13 != nil {
					z.EncExtension(x.TTL, yyxt13)
				} else {
					r.EncodeInt(int64(x.TTL))
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if false {
				} else if yyxt15 := z.Extension(z.I2Rtid(x.Expire)); yyxt15 != nil {
					z.EncExtension(x.Expire, yyxt15)
				} else {
					r.EncodeInt(int64(x.Expire))
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `e`)
				r.WriteMapElemValue()
				if false {
				} else if yyxt16 := z.Extension(z.I2Rtid(x.Expire)); yyxt16 != nil {
					z.EncExtension(x.Expire, yyxt16)
				} else {
					r.EncodeInt(int64(x.Expire))
				}
			}
			var yyn17 bool
			if x.Envelope.NodeID == nil {
				yyn17 = true
				goto LABEL17
			}
		LABEL17:
			if yyr2 || yy2arr2 {
				if yyn17 {
					r.WriteArrayElem()
					r.EncodeNil()
				} else {
					r.WriteArrayElem()
					if x.NodeID == nil {
						r.EncodeNil()
					} else {
						x.NodeID.CodecEncodeSelf(e)
					}
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `id`)
				r.WriteMapElemValue()
				if yyn17 {
					r.EncodeNil()
				} else {
					if x.NodeID == nil {
						r.EncodeNil()
					} else {
						x.NodeID.CodecEncodeSelf(e)
					}
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayEnd()
			} else {
				r.WriteMapEnd()
			}
		}
	}
}

func (x *PingReq) CodecDecodeSelf(d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	if false {
	} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
		z.DecExtension(x, yyxt1)
	} else {
		yyct2 := r.ContainerType()
		if yyct2 == codecSelferValueTypeMap9172 {
			yyl2 := r.ReadMapStart()
			if yyl2 == 0 {
				r.ReadMapEnd()
			} else {
				x.codecDecodeSelfFromMap(yyl2, d)
			}
		} else if yyct2 == codecSelferValueTypeArray9172 {
			yyl2 := r.ReadArrayStart()
			if yyl2 == 0 {
				r.ReadArrayEnd()
			} else {
				x.codecDecodeSelfFromArray(yyl2, d)
			}
		} else {
			panic(errCodecSelferOnlyMapOrArrayEncodeToStruct9172)
		}
	}
}

func (x *PingReq) codecDecodeSelfFromMap(l int, d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	var yyhl3 bool = l >= 0
	for yyj3 := 0; ; yyj3++ {
		if yyhl3 {
			if yyj3 >= l {
				break
			}
		} else {
			if r.CheckBreak() {
				break
			}
		}
		r.ReadMapElemKey()
		yys3 := z.StringView(r.DecodeStringAsBytes())
		r.ReadMapElemValue()
		switch yys3 {
		case "Node":
			if r.TryDecodeAsNil() {
				x.Node = Node{}
			} else {
				x.Node.CodecDecodeSelf(d)
			}
		case "v":
			if r.TryDecodeAsNil() {
				x.Envelope.Version = ""
			} else {
				x.Version = (string)(r.DecodeString())
			}
		case "t":
			if r.TryDecodeAsNil() {
				x.Envelope.TTL = 0
			} else {
				if false {
				} else if yyxt7 := z.Extension(z.I2Rtid(x.TTL)); yyxt7 != nil {
					z.DecExtension(x.TTL, yyxt7)
				} else {
					x.TTL = (time.Duration)(r.DecodeInt64())
				}
			}
		case "e":
			if r.TryDecodeAsNil() {
				x.Envelope.Expire = 0
			} else {
				if false {
				} else if yyxt9 := z.Extension(z.I2Rtid(x.Expire)); yyxt9 != nil {
					z.DecExtension(x.Expire, yyxt9)
				} else {
					x.Expire = (time.Duration)(r.DecodeInt64())
				}
			}
		case "id":
			if r.TryDecodeAsNil() {
				if true && x.Envelope.NodeID != nil {
					x.Envelope.NodeID = nil
				}
			} else {
				if x.Envelope.NodeID == nil {
					x.Envelope.NodeID = new(RawNodeID)
				}

				x.NodeID.CodecDecodeSelf(d)
			}
		default:
			z.DecStructFieldNotFound(-1, yys3)
		} // end switch yys3
	} // end for yyj3
	r.ReadMapEnd()
}

func (x *PingReq) codecDecodeSelfFromArray(l int, d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	var yyj11 int
	var yyb11 bool
	var yyhl11 bool = l >= 0
	yyj11++
	if yyhl11 {
		yyb11 = yyj11 > l
	} else {
		yyb11 = r.CheckBreak()
	}
	if yyb11 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Node = Node{}
	} else {
		x.Node.CodecDecodeSelf(d)
	}
	yyj11++
	if yyhl11 {
		yyb11 = yyj11 > l
	} else {
		yyb11 = r.CheckBreak()
	}
	if yyb11 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Envelope.Version = ""
	} else {
		x.Version = (string)(r.DecodeString())
	}
	yyj11++
	if yyhl11 {
		yyb11 = yyj11 > l
	} else {
		yyb11 = r.CheckBreak()
	}
	if yyb11 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Envelope.TTL = 0
	} else {
		if false {
		} else if yyxt15 := z.Extension(z.I2Rtid(x.TTL)); yyxt15 != nil {
			z.DecExtension(x.TTL, yyxt15)
		} else {
			x.TTL = (time.Duration)(r.DecodeInt64())
		}
	}
	yyj11++
	if yyhl11 {
		yyb11 = yyj11 > l
	} else {
		yyb11 = r.CheckBreak()
	}
	if yyb11 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Envelope.Expire = 0
	} else {
		if false {
		} else if yyxt17 := z.Extension(z.I2Rtid(x.Expire)); yyxt17 != nil {
			z.DecExtension(x.Expire, yyxt17)
		} else {
			x.Expire = (time.Duration)(r.DecodeInt64())
		}
	}
	yyj11++
	if yyhl11 {
		yyb11 = yyj11 > l
	} else {
		yyb11 = r.CheckBreak()
	}
	if yyb11 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		if true && x.Envelope.NodeID != nil {
			x.Envelope.NodeID = nil
		}
	} else {
		if x.Envelope.NodeID == nil {
			x.Envelope.NodeID = new(RawNodeID)
		}

		x.NodeID.CodecDecodeSelf(d)
	}
	for {
		yyj11++
		if yyhl11 {
			yyb11 = yyj11 > l
		} else {
			yyb11 = r.CheckBreak()
		}
		if yyb11 {
			break
		}
		r.ReadArrayElem()
		z.DecStructFieldNotFound(yyj11-1, "")
	}
	r.ReadArrayEnd()
}

func (x *PingResp) CodecEncodeSelf(e *codec1978.Encoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperEncoder(e)
	_, _, _ = h, z, r
	if x == nil {
		r.EncodeNil()
	} else {
		if false {
		} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
			z.EncExtension(x, yyxt1)
		} else {
			yysep2 := !z.EncBinary()
			yy2arr2 := z.EncBasicHandle().StructToArray
			_, _ = yysep2, yy2arr2
			const yyr2 bool = false // struct tag has 'toArray'
			if yyr2 || yy2arr2 {
				r.WriteArrayStart(5)
			} else {
				r.WriteMapStart(5)
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if false {
				} else {
					r.EncodeString(codecSelferCcUTF89172, string(x.Msg))
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `Msg`)
				r.WriteMapElemValue()
				if false {
				} else {
					r.EncodeString(codecSelferCcUTF89172, string(x.Msg))
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if false {
				} else {
					r.EncodeString(codecSelferCcUTF89172, string(x.Version))
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `v`)
				r.WriteMapElemValue()
				if false {
				} else {
					r.EncodeString(codecSelferCcUTF89172, string(x.Version))
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if false {
				} else if yyxt10 := z.Extension(z.I2Rtid(x.TTL)); yyxt10 != nil {
					z.EncExtension(x.TTL, yyxt10)
				} else {
					r.EncodeInt(int64(x.TTL))
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `t`)
				r.WriteMapElemValue()
				if false {
				} else if yyxt11 := z.Extension(z.I2Rtid(x.TTL)); yyxt11 != nil {
					z.EncExtension(x.TTL, yyxt11)
				} else {
					r.EncodeInt(int64(x.TTL))
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if false {
				} else if yyxt13 := z.Extension(z.I2Rtid(x.Expire)); yyxt13 != nil {
					z.EncExtension(x.Expire, yyxt13)
				} else {
					r.EncodeInt(int64(x.Expire))
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `e`)
				r.WriteMapElemValue()
				if false {
				} else if yyxt14 := z.Extension(z.I2Rtid(x.Expire)); yyxt14 != nil {
					z.EncExtension(x.Expire, yyxt14)
				} else {
					r.EncodeInt(int64(x.Expire))
				}
			}
			var yyn15 bool
			if x.Envelope.NodeID == nil {
				yyn15 = true
				goto LABEL15
			}
		LABEL15:
			if yyr2 || yy2arr2 {
				if yyn15 {
					r.WriteArrayElem()
					r.EncodeNil()
				} else {
					r.WriteArrayElem()
					if x.NodeID == nil {
						r.EncodeNil()
					} else {
						x.NodeID.CodecEncodeSelf(e)
					}
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `id`)
				r.WriteMapElemValue()
				if yyn15 {
					r.EncodeNil()
				} else {
					if x.NodeID == nil {
						r.EncodeNil()
					} else {
						x.NodeID.CodecEncodeSelf(e)
					}
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayEnd()
			} else {
				r.WriteMapEnd()
			}
		}
	}
}

func (x *PingResp) CodecDecodeSelf(d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	if false {
	} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
		z.DecExtension(x, yyxt1)
	} else {
		yyct2 := r.ContainerType()
		if yyct2 == codecSelferValueTypeMap9172 {
			yyl2 := r.ReadMapStart()
			if yyl2 == 0 {
				r.ReadMapEnd()
			} else {
				x.codecDecodeSelfFromMap(yyl2, d)
			}
		} else if yyct2 == codecSelferValueTypeArray9172 {
			yyl2 := r.ReadArrayStart()
			if yyl2 == 0 {
				r.ReadArrayEnd()
			} else {
				x.codecDecodeSelfFromArray(yyl2, d)
			}
		} else {
			panic(errCodecSelferOnlyMapOrArrayEncodeToStruct9172)
		}
	}
}

func (x *PingResp) codecDecodeSelfFromMap(l int, d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	var yyhl3 bool = l >= 0
	for yyj3 := 0; ; yyj3++ {
		if yyhl3 {
			if yyj3 >= l {
				break
			}
		} else {
			if r.CheckBreak() {
				break
			}
		}
		r.ReadMapElemKey()
		yys3 := z.StringView(r.DecodeStringAsBytes())
		r.ReadMapElemValue()
		switch yys3 {
		case "Msg":
			if r.TryDecodeAsNil() {
				x.Msg = ""
			} else {
				x.Msg = (string)(r.DecodeString())
			}
		case "v":
			if r.TryDecodeAsNil() {
				x.Envelope.Version = ""
			} else {
				x.Version = (string)(r.DecodeString())
			}
		case "t":
			if r.TryDecodeAsNil() {
				x.Envelope.TTL = 0
			} else {
				if false {
				} else if yyxt7 := z.Extension(z.I2Rtid(x.TTL)); yyxt7 != nil {
					z.DecExtension(x.TTL, yyxt7)
				} else {
					x.TTL = (time.Duration)(r.DecodeInt64())
				}
			}
		case "e":
			if r.TryDecodeAsNil() {
				x.Envelope.Expire = 0
			} else {
				if false {
				} else if yyxt9 := z.Extension(z.I2Rtid(x.Expire)); yyxt9 != nil {
					z.DecExtension(x.Expire, yyxt9)
				} else {
					x.Expire = (time.Duration)(r.DecodeInt64())
				}
			}
		case "id":
			if r.TryDecodeAsNil() {
				if true && x.Envelope.NodeID != nil {
					x.Envelope.NodeID = nil
				}
			} else {
				if x.Envelope.NodeID == nil {
					x.Envelope.NodeID = new(RawNodeID)
				}

				x.NodeID.CodecDecodeSelf(d)
			}
		default:
			z.DecStructFieldNotFound(-1, yys3)
		} // end switch yys3
	} // end for yyj3
	r.ReadMapEnd()
}

func (x *PingResp) codecDecodeSelfFromArray(l int, d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	var yyj11 int
	var yyb11 bool
	var yyhl11 bool = l >= 0
	yyj11++
	if yyhl11 {
		yyb11 = yyj11 > l
	} else {
		yyb11 = r.CheckBreak()
	}
	if yyb11 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Msg = ""
	} else {
		x.Msg = (string)(r.DecodeString())
	}
	yyj11++
	if yyhl11 {
		yyb11 = yyj11 > l
	} else {
		yyb11 = r.CheckBreak()
	}
	if yyb11 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Envelope.Version = ""
	} else {
		x.Version = (string)(r.DecodeString())
	}
	yyj11++
	if yyhl11 {
		yyb11 = yyj11 > l
	} else {
		yyb11 = r.CheckBreak()
	}
	if yyb11 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Envelope.TTL = 0
	} else {
		if false {
		} else if yyxt15 := z.Extension(z.I2Rtid(x.TTL)); yyxt15 != nil {
			z.DecExtension(x.TTL, yyxt15)
		} else {
			x.TTL = (time.Duration)(r.DecodeInt64())
		}
	}
	yyj11++
	if yyhl11 {
		yyb11 = yyj11 > l
	} else {
		yyb11 = r.CheckBreak()
	}
	if yyb11 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Envelope.Expire = 0
	} else {
		if false {
		} else if yyxt17 := z.Extension(z.I2Rtid(x.Expire)); yyxt17 != nil {
			z.DecExtension(x.Expire, yyxt17)
		} else {
			x.Expire = (time.Duration)(r.DecodeInt64())
		}
	}
	yyj11++
	if yyhl11 {
		yyb11 = yyj11 > l
	} else {
		yyb11 = r.CheckBreak()
	}
	if yyb11 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		if true && x.Envelope.NodeID != nil {
			x.Envelope.NodeID = nil
		}
	} else {
		if x.Envelope.NodeID == nil {
			x.Envelope.NodeID = new(RawNodeID)
		}

		x.NodeID.CodecDecodeSelf(d)
	}
	for {
		yyj11++
		if yyhl11 {
			yyb11 = yyj11 > l
		} else {
			yyb11 = r.CheckBreak()
		}
		if yyb11 {
			break
		}
		r.ReadArrayElem()
		z.DecStructFieldNotFound(yyj11-1, "")
	}
	r.ReadArrayEnd()
}

func (x *UploadMetricsReq) CodecEncodeSelf(e *codec1978.Encoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperEncoder(e)
	_, _, _ = h, z, r
	if x == nil {
		r.EncodeNil()
	} else {
		if false {
		} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
			z.EncExtension(x, yyxt1)
		} else {
			yysep2 := !z.EncBinary()
			yy2arr2 := z.EncBasicHandle().StructToArray
			_, _ = yysep2, yy2arr2
			const yyr2 bool = false // struct tag has 'toArray'
			if yyr2 || yy2arr2 {
				r.WriteArrayStart(5)
			} else {
				r.WriteMapStart(5)
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if x.MFBytes == nil {
					r.EncodeNil()
				} else {
					if false {
					} else {
						h.encSliceSliceuint8(([][]uint8)(x.MFBytes), e)
					}
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `MFBytes`)
				r.WriteMapElemValue()
				if x.MFBytes == nil {
					r.EncodeNil()
				} else {
					if false {
					} else {
						h.encSliceSliceuint8(([][]uint8)(x.MFBytes), e)
					}
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if false {
				} else {
					r.EncodeString(codecSelferCcUTF89172, string(x.Version))
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `v`)
				r.WriteMapElemValue()
				if false {
				} else {
					r.EncodeString(codecSelferCcUTF89172, string(x.Version))
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if false {
				} else if yyxt10 := z.Extension(z.I2Rtid(x.TTL)); yyxt10 != nil {
					z.EncExtension(x.TTL, yyxt10)
				} else {
					r.EncodeInt(int64(x.TTL))
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `t`)
				r.WriteMapElemValue()
				if false {
				} else if yyxt11 := z.Extension(z.I2Rtid(x.TTL)); yyxt11 != nil {
					z.EncExtension(x.TTL, yyxt11)
				} else {
					r.EncodeInt(int64(x.TTL))
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if false {
				} else if yyxt13 := z.Extension(z.I2Rtid(x.Expire)); yyxt13 != nil {
					z.EncExtension(x.Expire, yyxt13)
				} else {
					r.EncodeInt(int64(x.Expire))
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `e`)
				r.WriteMapElemValue()
				if false {
				} else if yyxt14 := z.Extension(z.I2Rtid(x.Expire)); yyxt14 != nil {
					z.EncExtension(x.Expire, yyxt14)
				} else {
					r.EncodeInt(int64(x.Expire))
				}
			}
			var yyn15 bool
			if x.Envelope.NodeID == nil {
				yyn15 = true
				goto LABEL15
			}
		LABEL15:
			if yyr2 || yy2arr2 {
				if yyn15 {
					r.WriteArrayElem()
					r.EncodeNil()
				} else {
					r.WriteArrayElem()
					if x.NodeID == nil {
						r.EncodeNil()
					} else {
						x.NodeID.CodecEncodeSelf(e)
					}
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `id`)
				r.WriteMapElemValue()
				if yyn15 {
					r.EncodeNil()
				} else {
					if x.NodeID == nil {
						r.EncodeNil()
					} else {
						x.NodeID.CodecEncodeSelf(e)
					}
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayEnd()
			} else {
				r.WriteMapEnd()
			}
		}
	}
}

func (x *UploadMetricsReq) CodecDecodeSelf(d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	if false {
	} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
		z.DecExtension(x, yyxt1)
	} else {
		yyct2 := r.ContainerType()
		if yyct2 == codecSelferValueTypeMap9172 {
			yyl2 := r.ReadMapStart()
			if yyl2 == 0 {
				r.ReadMapEnd()
			} else {
				x.codecDecodeSelfFromMap(yyl2, d)
			}
		} else if yyct2 == codecSelferValueTypeArray9172 {
			yyl2 := r.ReadArrayStart()
			if yyl2 == 0 {
				r.ReadArrayEnd()
			} else {
				x.codecDecodeSelfFromArray(yyl2, d)
			}
		} else {
			panic(errCodecSelferOnlyMapOrArrayEncodeToStruct9172)
		}
	}
}

func (x *UploadMetricsReq) codecDecodeSelfFromMap(l int, d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	var yyhl3 bool = l >= 0
	for yyj3 := 0; ; yyj3++ {
		if yyhl3 {
			if yyj3 >= l {
				break
			}
		} else {
			if r.CheckBreak() {
				break
			}
		}
		r.ReadMapElemKey()
		yys3 := z.StringView(r.DecodeStringAsBytes())
		r.ReadMapElemValue()
		switch yys3 {
		case "MFBytes":
			if r.TryDecodeAsNil() {
				x.MFBytes = nil
			} else {
				if false {
				} else {
					h.decSliceSliceuint8((*[][]uint8)(&x.MFBytes), d)
				}
			}
		case "v":
			if r.TryDecodeAsNil() {
				x.Envelope.Version = ""
			} else {
				x.Version = (string)(r.DecodeString())
			}
		case "t":
			if r.TryDecodeAsNil() {
				x.Envelope.TTL = 0
			} else {
				if false {
				} else if yyxt8 := z.Extension(z.I2Rtid(x.TTL)); yyxt8 != nil {
					z.DecExtension(x.TTL, yyxt8)
				} else {
					x.TTL = (time.Duration)(r.DecodeInt64())
				}
			}
		case "e":
			if r.TryDecodeAsNil() {
				x.Envelope.Expire = 0
			} else {
				if false {
				} else if yyxt10 := z.Extension(z.I2Rtid(x.Expire)); yyxt10 != nil {
					z.DecExtension(x.Expire, yyxt10)
				} else {
					x.Expire = (time.Duration)(r.DecodeInt64())
				}
			}
		case "id":
			if r.TryDecodeAsNil() {
				if true && x.Envelope.NodeID != nil {
					x.Envelope.NodeID = nil
				}
			} else {
				if x.Envelope.NodeID == nil {
					x.Envelope.NodeID = new(RawNodeID)
				}

				x.NodeID.CodecDecodeSelf(d)
			}
		default:
			z.DecStructFieldNotFound(-1, yys3)
		} // end switch yys3
	} // end for yyj3
	r.ReadMapEnd()
}

func (x *UploadMetricsReq) codecDecodeSelfFromArray(l int, d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	var yyj12 int
	var yyb12 bool
	var yyhl12 bool = l >= 0
	yyj12++
	if yyhl12 {
		yyb12 = yyj12 > l
	} else {
		yyb12 = r.CheckBreak()
	}
	if yyb12 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.MFBytes = nil
	} else {
		if false {
		} else {
			h.decSliceSliceuint8((*[][]uint8)(&x.MFBytes), d)
		}
	}
	yyj12++
	if yyhl12 {
		yyb12 = yyj12 > l
	} else {
		yyb12 = r.CheckBreak()
	}
	if yyb12 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Envelope.Version = ""
	} else {
		x.Version = (string)(r.DecodeString())
	}
	yyj12++
	if yyhl12 {
		yyb12 = yyj12 > l
	} else {
		yyb12 = r.CheckBreak()
	}
	if yyb12 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Envelope.TTL = 0
	} else {
		if false {
		} else if yyxt17 := z.Extension(z.I2Rtid(x.TTL)); yyxt17 != nil {
			z.DecExtension(x.TTL, yyxt17)
		} else {
			x.TTL = (time.Duration)(r.DecodeInt64())
		}
	}
	yyj12++
	if yyhl12 {
		yyb12 = yyj12 > l
	} else {
		yyb12 = r.CheckBreak()
	}
	if yyb12 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Envelope.Expire = 0
	} else {
		if false {
		} else if yyxt19 := z.Extension(z.I2Rtid(x.Expire)); yyxt19 != nil {
			z.DecExtension(x.Expire, yyxt19)
		} else {
			x.Expire = (time.Duration)(r.DecodeInt64())
		}
	}
	yyj12++
	if yyhl12 {
		yyb12 = yyj12 > l
	} else {
		yyb12 = r.CheckBreak()
	}
	if yyb12 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		if true && x.Envelope.NodeID != nil {
			x.Envelope.NodeID = nil
		}
	} else {
		if x.Envelope.NodeID == nil {
			x.Envelope.NodeID = new(RawNodeID)
		}

		x.NodeID.CodecDecodeSelf(d)
	}
	for {
		yyj12++
		if yyhl12 {
			yyb12 = yyj12 > l
		} else {
			yyb12 = r.CheckBreak()
		}
		if yyb12 {
			break
		}
		r.ReadArrayElem()
		z.DecStructFieldNotFound(yyj12-1, "")
	}
	r.ReadArrayEnd()
}

func (x *UploadMetricsResp) CodecEncodeSelf(e *codec1978.Encoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperEncoder(e)
	_, _, _ = h, z, r
	if x == nil {
		r.EncodeNil()
	} else {
		if false {
		} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
			z.EncExtension(x, yyxt1)
		} else {
			yysep2 := !z.EncBinary()
			yy2arr2 := z.EncBasicHandle().StructToArray
			_, _ = yysep2, yy2arr2
			const yyr2 bool = false // struct tag has 'toArray'
			if yyr2 || yy2arr2 {
				r.WriteArrayStart(5)
			} else {
				r.WriteMapStart(5)
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if false {
				} else {
					r.EncodeString(codecSelferCcUTF89172, string(x.Msg))
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `Msg`)
				r.WriteMapElemValue()
				if false {
				} else {
					r.EncodeString(codecSelferCcUTF89172, string(x.Msg))
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if false {
				} else {
					r.EncodeString(codecSelferCcUTF89172, string(x.Version))
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `v`)
				r.WriteMapElemValue()
				if false {
				} else {
					r.EncodeString(codecSelferCcUTF89172, string(x.Version))
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if false {
				} else if yyxt10 := z.Extension(z.I2Rtid(x.TTL)); yyxt10 != nil {
					z.EncExtension(x.TTL, yyxt10)
				} else {
					r.EncodeInt(int64(x.TTL))
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `t`)
				r.WriteMapElemValue()
				if false {
				} else if yyxt11 := z.Extension(z.I2Rtid(x.TTL)); yyxt11 != nil {
					z.EncExtension(x.TTL, yyxt11)
				} else {
					r.EncodeInt(int64(x.TTL))
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if false {
				} else if yyxt13 := z.Extension(z.I2Rtid(x.Expire)); yyxt13 != nil {
					z.EncExtension(x.Expire, yyxt13)
				} else {
					r.EncodeInt(int64(x.Expire))
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `e`)
				r.WriteMapElemValue()
				if false {
				} else if yyxt14 := z.Extension(z.I2Rtid(x.Expire)); yyxt14 != nil {
					z.EncExtension(x.Expire, yyxt14)
				} else {
					r.EncodeInt(int64(x.Expire))
				}
			}
			var yyn15 bool
			if x.Envelope.NodeID == nil {
				yyn15 = true
				goto LABEL15
			}
		LABEL15:
			if yyr2 || yy2arr2 {
				if yyn15 {
					r.WriteArrayElem()
					r.EncodeNil()
				} else {
					r.WriteArrayElem()
					if x.NodeID == nil {
						r.EncodeNil()
					} else {
						x.NodeID.CodecEncodeSelf(e)
					}
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `id`)
				r.WriteMapElemValue()
				if yyn15 {
					r.EncodeNil()
				} else {
					if x.NodeID == nil {
						r.EncodeNil()
					} else {
						x.NodeID.CodecEncodeSelf(e)
					}
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayEnd()
			} else {
				r.WriteMapEnd()
			}
		}
	}
}

func (x *UploadMetricsResp) CodecDecodeSelf(d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	if false {
	} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
		z.DecExtension(x, yyxt1)
	} else {
		yyct2 := r.ContainerType()
		if yyct2 == codecSelferValueTypeMap9172 {
			yyl2 := r.ReadMapStart()
			if yyl2 == 0 {
				r.ReadMapEnd()
			} else {
				x.codecDecodeSelfFromMap(yyl2, d)
			}
		} else if yyct2 == codecSelferValueTypeArray9172 {
			yyl2 := r.ReadArrayStart()
			if yyl2 == 0 {
				r.ReadArrayEnd()
			} else {
				x.codecDecodeSelfFromArray(yyl2, d)
			}
		} else {
			panic(errCodecSelferOnlyMapOrArrayEncodeToStruct9172)
		}
	}
}

func (x *UploadMetricsResp) codecDecodeSelfFromMap(l int, d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	var yyhl3 bool = l >= 0
	for yyj3 := 0; ; yyj3++ {
		if yyhl3 {
			if yyj3 >= l {
				break
			}
		} else {
			if r.CheckBreak() {
				break
			}
		}
		r.ReadMapElemKey()
		yys3 := z.StringView(r.DecodeStringAsBytes())
		r.ReadMapElemValue()
		switch yys3 {
		case "Msg":
			if r.TryDecodeAsNil() {
				x.Msg = ""
			} else {
				x.Msg = (string)(r.DecodeString())
			}
		case "v":
			if r.TryDecodeAsNil() {
				x.Envelope.Version = ""
			} else {
				x.Version = (string)(r.DecodeString())
			}
		case "t":
			if r.TryDecodeAsNil() {
				x.Envelope.TTL = 0
			} else {
				if false {
				} else if yyxt7 := z.Extension(z.I2Rtid(x.TTL)); yyxt7 != nil {
					z.DecExtension(x.TTL, yyxt7)
				} else {
					x.TTL = (time.Duration)(r.DecodeInt64())
				}
			}
		case "e":
			if r.TryDecodeAsNil() {
				x.Envelope.Expire = 0
			} else {
				if false {
				} else if yyxt9 := z.Extension(z.I2Rtid(x.Expire)); yyxt9 != nil {
					z.DecExtension(x.Expire, yyxt9)
				} else {
					x.Expire = (time.Duration)(r.DecodeInt64())
				}
			}
		case "id":
			if r.TryDecodeAsNil() {
				if true && x.Envelope.NodeID != nil {
					x.Envelope.NodeID = nil
				}
			} else {
				if x.Envelope.NodeID == nil {
					x.Envelope.NodeID = new(RawNodeID)
				}

				x.NodeID.CodecDecodeSelf(d)
			}
		default:
			z.DecStructFieldNotFound(-1, yys3)
		} // end switch yys3
	} // end for yyj3
	r.ReadMapEnd()
}

func (x *UploadMetricsResp) codecDecodeSelfFromArray(l int, d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	var yyj11 int
	var yyb11 bool
	var yyhl11 bool = l >= 0
	yyj11++
	if yyhl11 {
		yyb11 = yyj11 > l
	} else {
		yyb11 = r.CheckBreak()
	}
	if yyb11 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Msg = ""
	} else {
		x.Msg = (string)(r.DecodeString())
	}
	yyj11++
	if yyhl11 {
		yyb11 = yyj11 > l
	} else {
		yyb11 = r.CheckBreak()
	}
	if yyb11 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Envelope.Version = ""
	} else {
		x.Version = (string)(r.DecodeString())
	}
	yyj11++
	if yyhl11 {
		yyb11 = yyj11 > l
	} else {
		yyb11 = r.CheckBreak()
	}
	if yyb11 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Envelope.TTL = 0
	} else {
		if false {
		} else if yyxt15 := z.Extension(z.I2Rtid(x.TTL)); yyxt15 != nil {
			z.DecExtension(x.TTL, yyxt15)
		} else {
			x.TTL = (time.Duration)(r.DecodeInt64())
		}
	}
	yyj11++
	if yyhl11 {
		yyb11 = yyj11 > l
	} else {
		yyb11 = r.CheckBreak()
	}
	if yyb11 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Envelope.Expire = 0
	} else {
		if false {
		} else if yyxt17 := z.Extension(z.I2Rtid(x.Expire)); yyxt17 != nil {
			z.DecExtension(x.Expire, yyxt17)
		} else {
			x.Expire = (time.Duration)(r.DecodeInt64())
		}
	}
	yyj11++
	if yyhl11 {
		yyb11 = yyj11 > l
	} else {
		yyb11 = r.CheckBreak()
	}
	if yyb11 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		if true && x.Envelope.NodeID != nil {
			x.Envelope.NodeID = nil
		}
	} else {
		if x.Envelope.NodeID == nil {
			x.Envelope.NodeID = new(RawNodeID)
		}

		x.NodeID.CodecDecodeSelf(d)
	}
	for {
		yyj11++
		if yyhl11 {
			yyb11 = yyj11 > l
		} else {
			yyb11 = r.CheckBreak()
		}
		if yyb11 {
			break
		}
		r.ReadArrayElem()
		z.DecStructFieldNotFound(yyj11-1, "")
	}
	r.ReadArrayEnd()
}

func (x *FindNeighborReq) CodecEncodeSelf(e *codec1978.Encoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperEncoder(e)
	_, _, _ = h, z, r
	if x == nil {
		r.EncodeNil()
	} else {
		if false {
		} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
			z.EncExtension(x, yyxt1)
		} else {
			yysep2 := !z.EncBinary()
			yy2arr2 := z.EncBasicHandle().StructToArray
			_, _ = yysep2, yy2arr2
			const yyr2 bool = false // struct tag has 'toArray'
			if yyr2 || yy2arr2 {
				r.WriteArrayStart(7)
			} else {
				r.WriteMapStart(7)
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				x.NodeID.CodecEncodeSelf(e)
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `NodeID`)
				r.WriteMapElemValue()
				x.NodeID.CodecEncodeSelf(e)
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if x.Roles == nil {
					r.EncodeNil()
				} else {
					if false {
					} else {
						h.encSliceServerRole(([]ServerRole)(x.Roles), e)
					}
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `Roles`)
				r.WriteMapElemValue()
				if x.Roles == nil {
					r.EncodeNil()
				} else {
					if false {
					} else {
						h.encSliceServerRole(([]ServerRole)(x.Roles), e)
					}
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if false {
				} else {
					r.EncodeInt(int64(x.Count))
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `Count`)
				r.WriteMapElemValue()
				if false {
				} else {
					r.EncodeInt(int64(x.Count))
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if false {
				} else {
					r.EncodeString(codecSelferCcUTF89172, string(x.Version))
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `v`)
				r.WriteMapElemValue()
				if false {
				} else {
					r.EncodeString(codecSelferCcUTF89172, string(x.Version))
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if false {
				} else if yyxt16 := z.Extension(z.I2Rtid(x.TTL)); yyxt16 != nil {
					z.EncExtension(x.TTL, yyxt16)
				} else {
					r.EncodeInt(int64(x.TTL))
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `t`)
				r.WriteMapElemValue()
				if false {
				} else if yyxt17 := z.Extension(z.I2Rtid(x.TTL)); yyxt17 != nil {
					z.EncExtension(x.TTL, yyxt17)
				} else {
					r.EncodeInt(int64(x.TTL))
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if false {
				} else if yyxt19 := z.Extension(z.I2Rtid(x.Expire)); yyxt19 != nil {
					z.EncExtension(x.Expire, yyxt19)
				} else {
					r.EncodeInt(int64(x.Expire))
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `e`)
				r.WriteMapElemValue()
				if false {
				} else if yyxt20 := z.Extension(z.I2Rtid(x.Expire)); yyxt20 != nil {
					z.EncExtension(x.Expire, yyxt20)
				} else {
					r.EncodeInt(int64(x.Expire))
				}
			}
			var yyn21 bool
			if x.Envelope.NodeID == nil {
				yyn21 = true
				goto LABEL21
			}
		LABEL21:
			if yyr2 || yy2arr2 {
				if yyn21 {
					r.WriteArrayElem()
					r.EncodeNil()
				} else {
					r.WriteArrayElem()
					x.NodeID.CodecEncodeSelf(e)
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `id`)
				r.WriteMapElemValue()
				if yyn21 {
					r.EncodeNil()
				} else {
					x.NodeID.CodecEncodeSelf(e)
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayEnd()
			} else {
				r.WriteMapEnd()
			}
		}
	}
}

func (x *FindNeighborReq) CodecDecodeSelf(d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	if false {
	} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
		z.DecExtension(x, yyxt1)
	} else {
		yyct2 := r.ContainerType()
		if yyct2 == codecSelferValueTypeMap9172 {
			yyl2 := r.ReadMapStart()
			if yyl2 == 0 {
				r.ReadMapEnd()
			} else {
				x.codecDecodeSelfFromMap(yyl2, d)
			}
		} else if yyct2 == codecSelferValueTypeArray9172 {
			yyl2 := r.ReadArrayStart()
			if yyl2 == 0 {
				r.ReadArrayEnd()
			} else {
				x.codecDecodeSelfFromArray(yyl2, d)
			}
		} else {
			panic(errCodecSelferOnlyMapOrArrayEncodeToStruct9172)
		}
	}
}

func (x *FindNeighborReq) codecDecodeSelfFromMap(l int, d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	var yyhl3 bool = l >= 0
	for yyj3 := 0; ; yyj3++ {
		if yyhl3 {
			if yyj3 >= l {
				break
			}
		} else {
			if r.CheckBreak() {
				break
			}
		}
		r.ReadMapElemKey()
		yys3 := z.StringView(r.DecodeStringAsBytes())
		r.ReadMapElemValue()
		switch yys3 {
		case "NodeID":
			if r.TryDecodeAsNil() {
				x.NodeID = ""
			} else {
				x.NodeID.CodecDecodeSelf(d)
			}
		case "Roles":
			if r.TryDecodeAsNil() {
				x.Roles = nil
			} else {
				if false {
				} else {
					h.decSliceServerRole((*[]ServerRole)(&x.Roles), d)
				}
			}
		case "Count":
			if r.TryDecodeAsNil() {
				x.Count = 0
			} else {
				x.Count = (int)(z.C.IntV(r.DecodeInt64(), codecSelferBitsize9172))
			}
		case "v":
			if r.TryDecodeAsNil() {
				x.Envelope.Version = ""
			} else {
				x.Version = (string)(r.DecodeString())
			}
		case "t":
			if r.TryDecodeAsNil() {
				x.Envelope.TTL = 0
			} else {
				if false {
				} else if yyxt10 := z.Extension(z.I2Rtid(x.TTL)); yyxt10 != nil {
					z.DecExtension(x.TTL, yyxt10)
				} else {
					x.TTL = (time.Duration)(r.DecodeInt64())
				}
			}
		case "e":
			if r.TryDecodeAsNil() {
				x.Envelope.Expire = 0
			} else {
				if false {
				} else if yyxt12 := z.Extension(z.I2Rtid(x.Expire)); yyxt12 != nil {
					z.DecExtension(x.Expire, yyxt12)
				} else {
					x.Expire = (time.Duration)(r.DecodeInt64())
				}
			}
		case "id":
			if r.TryDecodeAsNil() {
				if true && x.Envelope.NodeID != nil {
					x.Envelope.NodeID = nil
				}
			} else {
				if x.Envelope.NodeID == nil {
					x.Envelope.NodeID = new(RawNodeID)
				}

				x.NodeID.CodecDecodeSelf(d)
			}
		default:
			z.DecStructFieldNotFound(-1, yys3)
		} // end switch yys3
	} // end for yyj3
	r.ReadMapEnd()
}

func (x *FindNeighborReq) codecDecodeSelfFromArray(l int, d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	var yyj14 int
	var yyb14 bool
	var yyhl14 bool = l >= 0
	yyj14++
	if yyhl14 {
		yyb14 = yyj14 > l
	} else {
		yyb14 = r.CheckBreak()
	}
	if yyb14 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.NodeID = ""
	} else {
		x.NodeID.CodecDecodeSelf(d)
	}
	yyj14++
	if yyhl14 {
		yyb14 = yyj14 > l
	} else {
		yyb14 = r.CheckBreak()
	}
	if yyb14 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Roles = nil
	} else {
		if false {
		} else {
			h.decSliceServerRole((*[]ServerRole)(&x.Roles), d)
		}
	}
	yyj14++
	if yyhl14 {
		yyb14 = yyj14 > l
	} else {
		yyb14 = r.CheckBreak()
	}
	if yyb14 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Count = 0
	} else {
		x.Count = (int)(z.C.IntV(r.DecodeInt64(), codecSelferBitsize9172))
	}
	yyj14++
	if yyhl14 {
		yyb14 = yyj14 > l
	} else {
		yyb14 = r.CheckBreak()
	}
	if yyb14 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Envelope.Version = ""
	} else {
		x.Version = (string)(r.DecodeString())
	}
	yyj14++
	if yyhl14 {
		yyb14 = yyj14 > l
	} else {
		yyb14 = r.CheckBreak()
	}
	if yyb14 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Envelope.TTL = 0
	} else {
		if false {
		} else if yyxt21 := z.Extension(z.I2Rtid(x.TTL)); yyxt21 != nil {
			z.DecExtension(x.TTL, yyxt21)
		} else {
			x.TTL = (time.Duration)(r.DecodeInt64())
		}
	}
	yyj14++
	if yyhl14 {
		yyb14 = yyj14 > l
	} else {
		yyb14 = r.CheckBreak()
	}
	if yyb14 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Envelope.Expire = 0
	} else {
		if false {
		} else if yyxt23 := z.Extension(z.I2Rtid(x.Expire)); yyxt23 != nil {
			z.DecExtension(x.Expire, yyxt23)
		} else {
			x.Expire = (time.Duration)(r.DecodeInt64())
		}
	}
	yyj14++
	if yyhl14 {
		yyb14 = yyj14 > l
	} else {
		yyb14 = r.CheckBreak()
	}
	if yyb14 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		if true && x.Envelope.NodeID != nil {
			x.Envelope.NodeID = nil
		}
	} else {
		if x.Envelope.NodeID == nil {
			x.Envelope.NodeID = new(RawNodeID)
		}

		x.NodeID.CodecDecodeSelf(d)
	}
	for {
		yyj14++
		if yyhl14 {
			yyb14 = yyj14 > l
		} else {
			yyb14 = r.CheckBreak()
		}
		if yyb14 {
			break
		}
		r.ReadArrayElem()
		z.DecStructFieldNotFound(yyj14-1, "")
	}
	r.ReadArrayEnd()
}

func (x *FindNeighborResp) CodecEncodeSelf(e *codec1978.Encoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperEncoder(e)
	_, _, _ = h, z, r
	if x == nil {
		r.EncodeNil()
	} else {
		if false {
		} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
			z.EncExtension(x, yyxt1)
		} else {
			yysep2 := !z.EncBinary()
			yy2arr2 := z.EncBasicHandle().StructToArray
			_, _ = yysep2, yy2arr2
			const yyr2 bool = false // struct tag has 'toArray'
			if yyr2 || yy2arr2 {
				r.WriteArrayStart(6)
			} else {
				r.WriteMapStart(6)
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if x.Nodes == nil {
					r.EncodeNil()
				} else {
					if false {
					} else {
						h.encSliceNode(([]Node)(x.Nodes), e)
					}
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `Nodes`)
				r.WriteMapElemValue()
				if x.Nodes == nil {
					r.EncodeNil()
				} else {
					if false {
					} else {
						h.encSliceNode(([]Node)(x.Nodes), e)
					}
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if false {
				} else {
					r.EncodeString(codecSelferCcUTF89172, string(x.Msg))
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `Msg`)
				r.WriteMapElemValue()
				if false {
				} else {
					r.EncodeString(codecSelferCcUTF89172, string(x.Msg))
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if false {
				} else {
					r.EncodeString(codecSelferCcUTF89172, string(x.Version))
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `v`)
				r.WriteMapElemValue()
				if false {
				} else {
					r.EncodeString(codecSelferCcUTF89172, string(x.Version))
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if false {
				} else if yyxt13 := z.Extension(z.I2Rtid(x.TTL)); yyxt13 != nil {
					z.EncExtension(x.TTL, yyxt13)
				} else {
					r.EncodeInt(int64(x.TTL))
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `t`)
				r.WriteMapElemValue()
				if false {
				} else if yyxt14 := z.Extension(z.I2Rtid(x.TTL)); yyxt14 != nil {
					z.EncExtension(x.TTL, yyxt14)
				} else {
					r.EncodeInt(int64(x.TTL))
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if false {
				} else if yyxt16 := z.Extension(z.I2Rtid(x.Expire)); yyxt16 != nil {
					z.EncExtension(x.Expire, yyxt16)
				} else {
					r.EncodeInt(int64(x.Expire))
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `e`)
				r.WriteMapElemValue()
				if false {
				} else if yyxt17 := z.Extension(z.I2Rtid(x.Expire)); yyxt17 != nil {
					z.EncExtension(x.Expire, yyxt17)
				} else {
					r.EncodeInt(int64(x.Expire))
				}
			}
			var yyn18 bool
			if x.Envelope.NodeID == nil {
				yyn18 = true
				goto LABEL18
			}
		LABEL18:
			if yyr2 || yy2arr2 {
				if yyn18 {
					r.WriteArrayElem()
					r.EncodeNil()
				} else {
					r.WriteArrayElem()
					if x.NodeID == nil {
						r.EncodeNil()
					} else {
						x.NodeID.CodecEncodeSelf(e)
					}
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `id`)
				r.WriteMapElemValue()
				if yyn18 {
					r.EncodeNil()
				} else {
					if x.NodeID == nil {
						r.EncodeNil()
					} else {
						x.NodeID.CodecEncodeSelf(e)
					}
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayEnd()
			} else {
				r.WriteMapEnd()
			}
		}
	}
}

func (x *FindNeighborResp) CodecDecodeSelf(d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	if false {
	} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
		z.DecExtension(x, yyxt1)
	} else {
		yyct2 := r.ContainerType()
		if yyct2 == codecSelferValueTypeMap9172 {
			yyl2 := r.ReadMapStart()
			if yyl2 == 0 {
				r.ReadMapEnd()
			} else {
				x.codecDecodeSelfFromMap(yyl2, d)
			}
		} else if yyct2 == codecSelferValueTypeArray9172 {
			yyl2 := r.ReadArrayStart()
			if yyl2 == 0 {
				r.ReadArrayEnd()
			} else {
				x.codecDecodeSelfFromArray(yyl2, d)
			}
		} else {
			panic(errCodecSelferOnlyMapOrArrayEncodeToStruct9172)
		}
	}
}

func (x *FindNeighborResp) codecDecodeSelfFromMap(l int, d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	var yyhl3 bool = l >= 0
	for yyj3 := 0; ; yyj3++ {
		if yyhl3 {
			if yyj3 >= l {
				break
			}
		} else {
			if r.CheckBreak() {
				break
			}
		}
		r.ReadMapElemKey()
		yys3 := z.StringView(r.DecodeStringAsBytes())
		r.ReadMapElemValue()
		switch yys3 {
		case "Nodes":
			if r.TryDecodeAsNil() {
				x.Nodes = nil
			} else {
				if false {
				} else {
					h.decSliceNode((*[]Node)(&x.Nodes), d)
				}
			}
		case "Msg":
			if r.TryDecodeAsNil() {
				x.Msg = ""
			} else {
				x.Msg = (string)(r.DecodeString())
			}
		case "v":
			if r.TryDecodeAsNil() {
				x.Envelope.Version = ""
			} else {
				x.Version = (string)(r.DecodeString())
			}
		case "t":
			if r.TryDecodeAsNil() {
				x.Envelope.TTL = 0
			} else {
				if false {
				} else if yyxt9 := z.Extension(z.I2Rtid(x.TTL)); yyxt9 != nil {
					z.DecExtension(x.TTL, yyxt9)
				} else {
					x.TTL = (time.Duration)(r.DecodeInt64())
				}
			}
		case "e":
			if r.TryDecodeAsNil() {
				x.Envelope.Expire = 0
			} else {
				if false {
				} else if yyxt11 := z.Extension(z.I2Rtid(x.Expire)); yyxt11 != nil {
					z.DecExtension(x.Expire, yyxt11)
				} else {
					x.Expire = (time.Duration)(r.DecodeInt64())
				}
			}
		case "id":
			if r.TryDecodeAsNil() {
				if true && x.Envelope.NodeID != nil {
					x.Envelope.NodeID = nil
				}
			} else {
				if x.Envelope.NodeID == nil {
					x.Envelope.NodeID = new(RawNodeID)
				}

				x.NodeID.CodecDecodeSelf(d)
			}
		default:
			z.DecStructFieldNotFound(-1, yys3)
		} // end switch yys3
	} // end for yyj3
	r.ReadMapEnd()
}

func (x *FindNeighborResp) codecDecodeSelfFromArray(l int, d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	var yyj13 int
	var yyb13 bool
	var yyhl13 bool = l >= 0
	yyj13++
	if yyhl13 {
		yyb13 = yyj13 > l
	} else {
		yyb13 = r.CheckBreak()
	}
	if yyb13 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Nodes = nil
	} else {
		if false {
		} else {
			h.decSliceNode((*[]Node)(&x.Nodes), d)
		}
	}
	yyj13++
	if yyhl13 {
		yyb13 = yyj13 > l
	} else {
		yyb13 = r.CheckBreak()
	}
	if yyb13 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Msg = ""
	} else {
		x.Msg = (string)(r.DecodeString())
	}
	yyj13++
	if yyhl13 {
		yyb13 = yyj13 > l
	} else {
		yyb13 = r.CheckBreak()
	}
	if yyb13 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Envelope.Version = ""
	} else {
		x.Version = (string)(r.DecodeString())
	}
	yyj13++
	if yyhl13 {
		yyb13 = yyj13 > l
	} else {
		yyb13 = r.CheckBreak()
	}
	if yyb13 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Envelope.TTL = 0
	} else {
		if false {
		} else if yyxt19 := z.Extension(z.I2Rtid(x.TTL)); yyxt19 != nil {
			z.DecExtension(x.TTL, yyxt19)
		} else {
			x.TTL = (time.Duration)(r.DecodeInt64())
		}
	}
	yyj13++
	if yyhl13 {
		yyb13 = yyj13 > l
	} else {
		yyb13 = r.CheckBreak()
	}
	if yyb13 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Envelope.Expire = 0
	} else {
		if false {
		} else if yyxt21 := z.Extension(z.I2Rtid(x.Expire)); yyxt21 != nil {
			z.DecExtension(x.Expire, yyxt21)
		} else {
			x.Expire = (time.Duration)(r.DecodeInt64())
		}
	}
	yyj13++
	if yyhl13 {
		yyb13 = yyj13 > l
	} else {
		yyb13 = r.CheckBreak()
	}
	if yyb13 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		if true && x.Envelope.NodeID != nil {
			x.Envelope.NodeID = nil
		}
	} else {
		if x.Envelope.NodeID == nil {
			x.Envelope.NodeID = new(RawNodeID)
		}

		x.NodeID.CodecDecodeSelf(d)
	}
	for {
		yyj13++
		if yyhl13 {
			yyb13 = yyj13 > l
		} else {
			yyb13 = r.CheckBreak()
		}
		if yyb13 {
			break
		}
		r.ReadArrayElem()
		z.DecStructFieldNotFound(yyj13-1, "")
	}
	r.ReadArrayEnd()
}

func (x *FindNodeReq) CodecEncodeSelf(e *codec1978.Encoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperEncoder(e)
	_, _, _ = h, z, r
	if x == nil {
		r.EncodeNil()
	} else {
		if false {
		} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
			z.EncExtension(x, yyxt1)
		} else {
			yysep2 := !z.EncBinary()
			yy2arr2 := z.EncBasicHandle().StructToArray
			_, _ = yysep2, yy2arr2
			const yyr2 bool = false // struct tag has 'toArray'
			if yyr2 || yy2arr2 {
				r.WriteArrayStart(5)
			} else {
				r.WriteMapStart(5)
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				x.NodeID.CodecEncodeSelf(e)
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `NodeID`)
				r.WriteMapElemValue()
				x.NodeID.CodecEncodeSelf(e)
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if false {
				} else {
					r.EncodeString(codecSelferCcUTF89172, string(x.Version))
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `v`)
				r.WriteMapElemValue()
				if false {
				} else {
					r.EncodeString(codecSelferCcUTF89172, string(x.Version))
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if false {
				} else if yyxt10 := z.Extension(z.I2Rtid(x.TTL)); yyxt10 != nil {
					z.EncExtension(x.TTL, yyxt10)
				} else {
					r.EncodeInt(int64(x.TTL))
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `t`)
				r.WriteMapElemValue()
				if false {
				} else if yyxt11 := z.Extension(z.I2Rtid(x.TTL)); yyxt11 != nil {
					z.EncExtension(x.TTL, yyxt11)
				} else {
					r.EncodeInt(int64(x.TTL))
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if false {
				} else if yyxt13 := z.Extension(z.I2Rtid(x.Expire)); yyxt13 != nil {
					z.EncExtension(x.Expire, yyxt13)
				} else {
					r.EncodeInt(int64(x.Expire))
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `e`)
				r.WriteMapElemValue()
				if false {
				} else if yyxt14 := z.Extension(z.I2Rtid(x.Expire)); yyxt14 != nil {
					z.EncExtension(x.Expire, yyxt14)
				} else {
					r.EncodeInt(int64(x.Expire))
				}
			}
			var yyn15 bool
			if x.Envelope.NodeID == nil {
				yyn15 = true
				goto LABEL15
			}
		LABEL15:
			if yyr2 || yy2arr2 {
				if yyn15 {
					r.WriteArrayElem()
					r.EncodeNil()
				} else {
					r.WriteArrayElem()
					x.NodeID.CodecEncodeSelf(e)
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `id`)
				r.WriteMapElemValue()
				if yyn15 {
					r.EncodeNil()
				} else {
					x.NodeID.CodecEncodeSelf(e)
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayEnd()
			} else {
				r.WriteMapEnd()
			}
		}
	}
}

func (x *FindNodeReq) CodecDecodeSelf(d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	if false {
	} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
		z.DecExtension(x, yyxt1)
	} else {
		yyct2 := r.ContainerType()
		if yyct2 == codecSelferValueTypeMap9172 {
			yyl2 := r.ReadMapStart()
			if yyl2 == 0 {
				r.ReadMapEnd()
			} else {
				x.codecDecodeSelfFromMap(yyl2, d)
			}
		} else if yyct2 == codecSelferValueTypeArray9172 {
			yyl2 := r.ReadArrayStart()
			if yyl2 == 0 {
				r.ReadArrayEnd()
			} else {
				x.codecDecodeSelfFromArray(yyl2, d)
			}
		} else {
			panic(errCodecSelferOnlyMapOrArrayEncodeToStruct9172)
		}
	}
}

func (x *FindNodeReq) codecDecodeSelfFromMap(l int, d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	var yyhl3 bool = l >= 0
	for yyj3 := 0; ; yyj3++ {
		if yyhl3 {
			if yyj3 >= l {
				break
			}
		} else {
			if r.CheckBreak() {
				break
			}
		}
		r.ReadMapElemKey()
		yys3 := z.StringView(r.DecodeStringAsBytes())
		r.ReadMapElemValue()
		switch yys3 {
		case "NodeID":
			if r.TryDecodeAsNil() {
				x.NodeID = ""
			} else {
				x.NodeID.CodecDecodeSelf(d)
			}
		case "v":
			if r.TryDecodeAsNil() {
				x.Envelope.Version = ""
			} else {
				x.Version = (string)(r.DecodeString())
			}
		case "t":
			if r.TryDecodeAsNil() {
				x.Envelope.TTL = 0
			} else {
				if false {
				} else if yyxt7 := z.Extension(z.I2Rtid(x.TTL)); yyxt7 != nil {
					z.DecExtension(x.TTL, yyxt7)
				} else {
					x.TTL = (time.Duration)(r.DecodeInt64())
				}
			}
		case "e":
			if r.TryDecodeAsNil() {
				x.Envelope.Expire = 0
			} else {
				if false {
				} else if yyxt9 := z.Extension(z.I2Rtid(x.Expire)); yyxt9 != nil {
					z.DecExtension(x.Expire, yyxt9)
				} else {
					x.Expire = (time.Duration)(r.DecodeInt64())
				}
			}
		case "id":
			if r.TryDecodeAsNil() {
				if true && x.Envelope.NodeID != nil {
					x.Envelope.NodeID = nil
				}
			} else {
				if x.Envelope.NodeID == nil {
					x.Envelope.NodeID = new(RawNodeID)
				}

				x.NodeID.CodecDecodeSelf(d)
			}
		default:
			z.DecStructFieldNotFound(-1, yys3)
		} // end switch yys3
	} // end for yyj3
	r.ReadMapEnd()
}

func (x *FindNodeReq) codecDecodeSelfFromArray(l int, d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	var yyj11 int
	var yyb11 bool
	var yyhl11 bool = l >= 0
	yyj11++
	if yyhl11 {
		yyb11 = yyj11 > l
	} else {
		yyb11 = r.CheckBreak()
	}
	if yyb11 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.NodeID = ""
	} else {
		x.NodeID.CodecDecodeSelf(d)
	}
	yyj11++
	if yyhl11 {
		yyb11 = yyj11 > l
	} else {
		yyb11 = r.CheckBreak()
	}
	if yyb11 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Envelope.Version = ""
	} else {
		x.Version = (string)(r.DecodeString())
	}
	yyj11++
	if yyhl11 {
		yyb11 = yyj11 > l
	} else {
		yyb11 = r.CheckBreak()
	}
	if yyb11 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Envelope.TTL = 0
	} else {
		if false {
		} else if yyxt15 := z.Extension(z.I2Rtid(x.TTL)); yyxt15 != nil {
			z.DecExtension(x.TTL, yyxt15)
		} else {
			x.TTL = (time.Duration)(r.DecodeInt64())
		}
	}
	yyj11++
	if yyhl11 {
		yyb11 = yyj11 > l
	} else {
		yyb11 = r.CheckBreak()
	}
	if yyb11 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Envelope.Expire = 0
	} else {
		if false {
		} else if yyxt17 := z.Extension(z.I2Rtid(x.Expire)); yyxt17 != nil {
			z.DecExtension(x.Expire, yyxt17)
		} else {
			x.Expire = (time.Duration)(r.DecodeInt64())
		}
	}
	yyj11++
	if yyhl11 {
		yyb11 = yyj11 > l
	} else {
		yyb11 = r.CheckBreak()
	}
	if yyb11 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		if true && x.Envelope.NodeID != nil {
			x.Envelope.NodeID = nil
		}
	} else {
		if x.Envelope.NodeID == nil {
			x.Envelope.NodeID = new(RawNodeID)
		}

		x.NodeID.CodecDecodeSelf(d)
	}
	for {
		yyj11++
		if yyhl11 {
			yyb11 = yyj11 > l
		} else {
			yyb11 = r.CheckBreak()
		}
		if yyb11 {
			break
		}
		r.ReadArrayElem()
		z.DecStructFieldNotFound(yyj11-1, "")
	}
	r.ReadArrayEnd()
}

func (x *FindNodeResp) CodecEncodeSelf(e *codec1978.Encoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperEncoder(e)
	_, _, _ = h, z, r
	if x == nil {
		r.EncodeNil()
	} else {
		if false {
		} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
			z.EncExtension(x, yyxt1)
		} else {
			yysep2 := !z.EncBinary()
			yy2arr2 := z.EncBasicHandle().StructToArray
			_, _ = yysep2, yy2arr2
			const yyr2 bool = false // struct tag has 'toArray'
			if yyr2 || yy2arr2 {
				r.WriteArrayStart(6)
			} else {
				r.WriteMapStart(6)
			}
			var yyn3 bool
			if x.Node == nil {
				yyn3 = true
				goto LABEL3
			}
		LABEL3:
			if yyr2 || yy2arr2 {
				if yyn3 {
					r.WriteArrayElem()
					r.EncodeNil()
				} else {
					r.WriteArrayElem()
					if x.Node == nil {
						r.EncodeNil()
					} else {
						x.Node.CodecEncodeSelf(e)
					}
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `Node`)
				r.WriteMapElemValue()
				if yyn3 {
					r.EncodeNil()
				} else {
					if x.Node == nil {
						r.EncodeNil()
					} else {
						x.Node.CodecEncodeSelf(e)
					}
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if false {
				} else {
					r.EncodeString(codecSelferCcUTF89172, string(x.Msg))
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `Msg`)
				r.WriteMapElemValue()
				if false {
				} else {
					r.EncodeString(codecSelferCcUTF89172, string(x.Msg))
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if false {
				} else {
					r.EncodeString(codecSelferCcUTF89172, string(x.Version))
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `v`)
				r.WriteMapElemValue()
				if false {
				} else {
					r.EncodeString(codecSelferCcUTF89172, string(x.Version))
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if false {
				} else if yyxt13 := z.Extension(z.I2Rtid(x.TTL)); yyxt13 != nil {
					z.EncExtension(x.TTL, yyxt13)
				} else {
					r.EncodeInt(int64(x.TTL))
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `t`)
				r.WriteMapElemValue()
				if false {
				} else if yyxt14 := z.Extension(z.I2Rtid(x.TTL)); yyxt14 != nil {
					z.EncExtension(x.TTL, yyxt14)
				} else {
					r.EncodeInt(int64(x.TTL))
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if false {
				} else if yyxt16 := z.Extension(z.I2Rtid(x.Expire)); yyxt16 != nil {
					z.EncExtension(x.Expire, yyxt16)
				} else {
					r.EncodeInt(int64(x.Expire))
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `e`)
				r.WriteMapElemValue()
				if false {
				} else if yyxt17 := z.Extension(z.I2Rtid(x.Expire)); yyxt17 != nil {
					z.EncExtension(x.Expire, yyxt17)
				} else {
					r.EncodeInt(int64(x.Expire))
				}
			}
			var yyn18 bool
			if x.Envelope.NodeID == nil {
				yyn18 = true
				goto LABEL18
			}
		LABEL18:
			if yyr2 || yy2arr2 {
				if yyn18 {
					r.WriteArrayElem()
					r.EncodeNil()
				} else {
					r.WriteArrayElem()
					if x.NodeID == nil {
						r.EncodeNil()
					} else {
						x.NodeID.CodecEncodeSelf(e)
					}
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `id`)
				r.WriteMapElemValue()
				if yyn18 {
					r.EncodeNil()
				} else {
					if x.NodeID == nil {
						r.EncodeNil()
					} else {
						x.NodeID.CodecEncodeSelf(e)
					}
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayEnd()
			} else {
				r.WriteMapEnd()
			}
		}
	}
}

func (x *FindNodeResp) CodecDecodeSelf(d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	if false {
	} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
		z.DecExtension(x, yyxt1)
	} else {
		yyct2 := r.ContainerType()
		if yyct2 == codecSelferValueTypeMap9172 {
			yyl2 := r.ReadMapStart()
			if yyl2 == 0 {
				r.ReadMapEnd()
			} else {
				x.codecDecodeSelfFromMap(yyl2, d)
			}
		} else if yyct2 == codecSelferValueTypeArray9172 {
			yyl2 := r.ReadArrayStart()
			if yyl2 == 0 {
				r.ReadArrayEnd()
			} else {
				x.codecDecodeSelfFromArray(yyl2, d)
			}
		} else {
			panic(errCodecSelferOnlyMapOrArrayEncodeToStruct9172)
		}
	}
}

func (x *FindNodeResp) codecDecodeSelfFromMap(l int, d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	var yyhl3 bool = l >= 0
	for yyj3 := 0; ; yyj3++ {
		if yyhl3 {
			if yyj3 >= l {
				break
			}
		} else {
			if r.CheckBreak() {
				break
			}
		}
		r.ReadMapElemKey()
		yys3 := z.StringView(r.DecodeStringAsBytes())
		r.ReadMapElemValue()
		switch yys3 {
		case "Node":
			if r.TryDecodeAsNil() {
				if true && x.Node != nil {
					x.Node = nil
				}
			} else {
				if x.Node == nil {
					x.Node = new(Node)
				}

				x.Node.CodecDecodeSelf(d)
			}
		case "Msg":
			if r.TryDecodeAsNil() {
				x.Msg = ""
			} else {
				x.Msg = (string)(r.DecodeString())
			}
		case "v":
			if r.TryDecodeAsNil() {
				x.Envelope.Version = ""
			} else {
				x.Version = (string)(r.DecodeString())
			}
		case "t":
			if r.TryDecodeAsNil() {
				x.Envelope.TTL = 0
			} else {
				if false {
				} else if yyxt8 := z.Extension(z.I2Rtid(x.TTL)); yyxt8 != nil {
					z.DecExtension(x.TTL, yyxt8)
				} else {
					x.TTL = (time.Duration)(r.DecodeInt64())
				}
			}
		case "e":
			if r.TryDecodeAsNil() {
				x.Envelope.Expire = 0
			} else {
				if false {
				} else if yyxt10 := z.Extension(z.I2Rtid(x.Expire)); yyxt10 != nil {
					z.DecExtension(x.Expire, yyxt10)
				} else {
					x.Expire = (time.Duration)(r.DecodeInt64())
				}
			}
		case "id":
			if r.TryDecodeAsNil() {
				if true && x.Envelope.NodeID != nil {
					x.Envelope.NodeID = nil
				}
			} else {
				if x.Envelope.NodeID == nil {
					x.Envelope.NodeID = new(RawNodeID)
				}

				x.NodeID.CodecDecodeSelf(d)
			}
		default:
			z.DecStructFieldNotFound(-1, yys3)
		} // end switch yys3
	} // end for yyj3
	r.ReadMapEnd()
}

func (x *FindNodeResp) codecDecodeSelfFromArray(l int, d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	var yyj12 int
	var yyb12 bool
	var yyhl12 bool = l >= 0
	yyj12++
	if yyhl12 {
		yyb12 = yyj12 > l
	} else {
		yyb12 = r.CheckBreak()
	}
	if yyb12 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		if true && x.Node != nil {
			x.Node = nil
		}
	} else {
		if x.Node == nil {
			x.Node = new(Node)
		}

		x.Node.CodecDecodeSelf(d)
	}
	yyj12++
	if yyhl12 {
		yyb12 = yyj12 > l
	} else {
		yyb12 = r.CheckBreak()
	}
	if yyb12 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Msg = ""
	} else {
		x.Msg = (string)(r.DecodeString())
	}
	yyj12++
	if yyhl12 {
		yyb12 = yyj12 > l
	} else {
		yyb12 = r.CheckBreak()
	}
	if yyb12 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Envelope.Version = ""
	} else {
		x.Version = (string)(r.DecodeString())
	}
	yyj12++
	if yyhl12 {
		yyb12 = yyj12 > l
	} else {
		yyb12 = r.CheckBreak()
	}
	if yyb12 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Envelope.TTL = 0
	} else {
		if false {
		} else if yyxt17 := z.Extension(z.I2Rtid(x.TTL)); yyxt17 != nil {
			z.DecExtension(x.TTL, yyxt17)
		} else {
			x.TTL = (time.Duration)(r.DecodeInt64())
		}
	}
	yyj12++
	if yyhl12 {
		yyb12 = yyj12 > l
	} else {
		yyb12 = r.CheckBreak()
	}
	if yyb12 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Envelope.Expire = 0
	} else {
		if false {
		} else if yyxt19 := z.Extension(z.I2Rtid(x.Expire)); yyxt19 != nil {
			z.DecExtension(x.Expire, yyxt19)
		} else {
			x.Expire = (time.Duration)(r.DecodeInt64())
		}
	}
	yyj12++
	if yyhl12 {
		yyb12 = yyj12 > l
	} else {
		yyb12 = r.CheckBreak()
	}
	if yyb12 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		if true && x.Envelope.NodeID != nil {
			x.Envelope.NodeID = nil
		}
	} else {
		if x.Envelope.NodeID == nil {
			x.Envelope.NodeID = new(RawNodeID)
		}

		x.NodeID.CodecDecodeSelf(d)
	}
	for {
		yyj12++
		if yyhl12 {
			yyb12 = yyj12 > l
		} else {
			yyb12 = r.CheckBreak()
		}
		if yyb12 {
			break
		}
		r.ReadArrayElem()
		z.DecStructFieldNotFound(yyj12-1, "")
	}
	r.ReadArrayEnd()
}

func (x DatabaseID) CodecEncodeSelf(e *codec1978.Encoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperEncoder(e)
	_, _, _ = h, z, r
	if false {
	} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
		z.EncExtension(x, yyxt1)
	} else {
		r.EncodeString(codecSelferCcUTF89172, string(x))
	}
}

func (x *DatabaseID) CodecDecodeSelf(d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	if false {
	} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
		z.DecExtension(x, yyxt1)
	} else {
		*x = (DatabaseID)(r.DecodeString())
	}
}

func (x *RawNodeID) CodecEncodeSelf(e *codec1978.Encoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperEncoder(e)
	_, _, _ = h, z, r
	if x == nil {
		r.EncodeNil()
	} else {
		if false {
		} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
			z.EncExtension(x, yyxt1)
		} else {
			yysep2 := !z.EncBinary()
			yy2arr2 := z.EncBasicHandle().StructToArray
			_, _ = yysep2, yy2arr2
			const yyr2 bool = false // struct tag has 'toArray'
			if yyr2 || yy2arr2 {
				r.WriteArrayStart(1)
			} else {
				r.WriteMapStart(1)
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				yy4 := &x.Hash
				if false {
				} else if yyxt5 := z.Extension(z.I2Rtid(yy4)); yyxt5 != nil {
					z.EncExtension(yy4, yyxt5)
				} else {
					h.enchash_Hash((*pkg5_hash.Hash)(yy4), e)
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `Hash`)
				r.WriteMapElemValue()
				yy6 := &x.Hash
				if false {
				} else if yyxt7 := z.Extension(z.I2Rtid(yy6)); yyxt7 != nil {
					z.EncExtension(yy6, yyxt7)
				} else {
					h.enchash_Hash((*pkg5_hash.Hash)(yy6), e)
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayEnd()
			} else {
				r.WriteMapEnd()
			}
		}
	}
}

func (x *RawNodeID) CodecDecodeSelf(d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	if false {
	} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
		z.DecExtension(x, yyxt1)
	} else {
		yyct2 := r.ContainerType()
		if yyct2 == codecSelferValueTypeMap9172 {
			yyl2 := r.ReadMapStart()
			if yyl2 == 0 {
				r.ReadMapEnd()
			} else {
				x.codecDecodeSelfFromMap(yyl2, d)
			}
		} else if yyct2 == codecSelferValueTypeArray9172 {
			yyl2 := r.ReadArrayStart()
			if yyl2 == 0 {
				r.ReadArrayEnd()
			} else {
				x.codecDecodeSelfFromArray(yyl2, d)
			}
		} else {
			panic(errCodecSelferOnlyMapOrArrayEncodeToStruct9172)
		}
	}
}

func (x *RawNodeID) codecDecodeSelfFromMap(l int, d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	var yyhl3 bool = l >= 0
	for yyj3 := 0; ; yyj3++ {
		if yyhl3 {
			if yyj3 >= l {
				break
			}
		} else {
			if r.CheckBreak() {
				break
			}
		}
		r.ReadMapElemKey()
		yys3 := z.StringView(r.DecodeStringAsBytes())
		r.ReadMapElemValue()
		switch yys3 {
		case "Hash":
			if r.TryDecodeAsNil() {
				x.Hash = pkg5_hash.Hash{}
			} else {
				if false {
				} else if yyxt5 := z.Extension(z.I2Rtid(x.Hash)); yyxt5 != nil {
					z.DecExtension(x.Hash, yyxt5)
				} else {
					h.dechash_Hash((*pkg5_hash.Hash)(&x.Hash), d)
				}
			}
		default:
			z.DecStructFieldNotFound(-1, yys3)
		} // end switch yys3
	} // end for yyj3
	r.ReadMapEnd()
}

func (x *RawNodeID) codecDecodeSelfFromArray(l int, d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	var yyj6 int
	var yyb6 bool
	var yyhl6 bool = l >= 0
	yyj6++
	if yyhl6 {
		yyb6 = yyj6 > l
	} else {
		yyb6 = r.CheckBreak()
	}
	if yyb6 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Hash = pkg5_hash.Hash{}
	} else {
		if false {
		} else if yyxt8 := z.Extension(z.I2Rtid(x.Hash)); yyxt8 != nil {
			z.DecExtension(x.Hash, yyxt8)
		} else {
			h.dechash_Hash((*pkg5_hash.Hash)(&x.Hash), d)
		}
	}
	for {
		yyj6++
		if yyhl6 {
			yyb6 = yyj6 > l
		} else {
			yyb6 = r.CheckBreak()
		}
		if yyb6 {
			break
		}
		r.ReadArrayElem()
		z.DecStructFieldNotFound(yyj6-1, "")
	}
	r.ReadArrayEnd()
}

func (x NodeID) CodecEncodeSelf(e *codec1978.Encoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperEncoder(e)
	_, _, _ = h, z, r
	if false {
	} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
		z.EncExtension(x, yyxt1)
	} else if z.EncBinary() {
		z.EncBinaryMarshal(&x)
	} else {
		r.EncodeString(codecSelferCcUTF89172, string(x))
	}
}

func (x *NodeID) CodecDecodeSelf(d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	if false {
	} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
		z.DecExtension(x, yyxt1)
	} else if z.DecBinary() {
		z.DecBinaryUnmarshal(x)
	} else {
		*x = (NodeID)(r.DecodeString())
	}
}

func (x *NodeKey) CodecEncodeSelf(e *codec1978.Encoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperEncoder(e)
	_, _, _ = h, z, r
	if x == nil {
		r.EncodeNil()
	} else {
		if false {
		} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
			z.EncExtension(x, yyxt1)
		} else {
			yysep2 := !z.EncBinary()
			yy2arr2 := z.EncBasicHandle().StructToArray
			_, _ = yysep2, yy2arr2
			const yyr2 bool = false // struct tag has 'toArray'
			if yyr2 || yy2arr2 {
				r.WriteArrayStart(1)
			} else {
				r.WriteMapStart(1)
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				yy4 := &x.Hash
				if false {
				} else if yyxt5 := z.Extension(z.I2Rtid(yy4)); yyxt5 != nil {
					z.EncExtension(yy4, yyxt5)
				} else {
					h.enchash_Hash((*pkg5_hash.Hash)(yy4), e)
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `Hash`)
				r.WriteMapElemValue()
				yy6 := &x.Hash
				if false {
				} else if yyxt7 := z.Extension(z.I2Rtid(yy6)); yyxt7 != nil {
					z.EncExtension(yy6, yyxt7)
				} else {
					h.enchash_Hash((*pkg5_hash.Hash)(yy6), e)
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayEnd()
			} else {
				r.WriteMapEnd()
			}
		}
	}
}

func (x *NodeKey) CodecDecodeSelf(d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	if false {
	} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
		z.DecExtension(x, yyxt1)
	} else {
		yyct2 := r.ContainerType()
		if yyct2 == codecSelferValueTypeMap9172 {
			yyl2 := r.ReadMapStart()
			if yyl2 == 0 {
				r.ReadMapEnd()
			} else {
				x.codecDecodeSelfFromMap(yyl2, d)
			}
		} else if yyct2 == codecSelferValueTypeArray9172 {
			yyl2 := r.ReadArrayStart()
			if yyl2 == 0 {
				r.ReadArrayEnd()
			} else {
				x.codecDecodeSelfFromArray(yyl2, d)
			}
		} else {
			panic(errCodecSelferOnlyMapOrArrayEncodeToStruct9172)
		}
	}
}

func (x *NodeKey) codecDecodeSelfFromMap(l int, d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	var yyhl3 bool = l >= 0
	for yyj3 := 0; ; yyj3++ {
		if yyhl3 {
			if yyj3 >= l {
				break
			}
		} else {
			if r.CheckBreak() {
				break
			}
		}
		r.ReadMapElemKey()
		yys3 := z.StringView(r.DecodeStringAsBytes())
		r.ReadMapElemValue()
		switch yys3 {
		case "Hash":
			if r.TryDecodeAsNil() {
				x.Hash = pkg5_hash.Hash{}
			} else {
				if false {
				} else if yyxt5 := z.Extension(z.I2Rtid(x.Hash)); yyxt5 != nil {
					z.DecExtension(x.Hash, yyxt5)
				} else {
					h.dechash_Hash((*pkg5_hash.Hash)(&x.Hash), d)
				}
			}
		default:
			z.DecStructFieldNotFound(-1, yys3)
		} // end switch yys3
	} // end for yyj3
	r.ReadMapEnd()
}

func (x *NodeKey) codecDecodeSelfFromArray(l int, d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	var yyj6 int
	var yyb6 bool
	var yyhl6 bool = l >= 0
	yyj6++
	if yyhl6 {
		yyb6 = yyj6 > l
	} else {
		yyb6 = r.CheckBreak()
	}
	if yyb6 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Hash = pkg5_hash.Hash{}
	} else {
		if false {
		} else if yyxt8 := z.Extension(z.I2Rtid(x.Hash)); yyxt8 != nil {
			z.DecExtension(x.Hash, yyxt8)
		} else {
			h.dechash_Hash((*pkg5_hash.Hash)(&x.Hash), d)
		}
	}
	for {
		yyj6++
		if yyhl6 {
			yyb6 = yyj6 > l
		} else {
			yyb6 = r.CheckBreak()
		}
		if yyb6 {
			break
		}
		r.ReadArrayElem()
		z.DecStructFieldNotFound(yyj6-1, "")
	}
	r.ReadArrayEnd()
}

func (x *Node) CodecEncodeSelf(e *codec1978.Encoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperEncoder(e)
	_, _, _ = h, z, r
	if x == nil {
		r.EncodeNil()
	} else {
		if false {
		} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
			z.EncExtension(x, yyxt1)
		} else {
			yysep2 := !z.EncBinary()
			yy2arr2 := z.EncBasicHandle().StructToArray
			_, _ = yysep2, yy2arr2
			const yyr2 bool = false // struct tag has 'toArray'
			if yyr2 || yy2arr2 {
				r.WriteArrayStart(5)
			} else {
				r.WriteMapStart(5)
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				x.ID.CodecEncodeSelf(e)
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `ID`)
				r.WriteMapElemValue()
				x.ID.CodecEncodeSelf(e)
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				x.Role.CodecEncodeSelf(e)
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `Role`)
				r.WriteMapElemValue()
				x.Role.CodecEncodeSelf(e)
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if false {
				} else {
					r.EncodeString(codecSelferCcUTF89172, string(x.Addr))
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `Addr`)
				r.WriteMapElemValue()
				if false {
				} else {
					r.EncodeString(codecSelferCcUTF89172, string(x.Addr))
				}
			}
			var yyn12 bool
			if x.PublicKey == nil {
				yyn12 = true
				goto LABEL12
			}
		LABEL12:
			if yyr2 || yy2arr2 {
				if yyn12 {
					r.WriteArrayElem()
					r.EncodeNil()
				} else {
					r.WriteArrayElem()
					if x.PublicKey == nil {
						r.EncodeNil()
					} else {
						if false {
						} else if yyxt13 := z.Extension(z.I2Rtid(x.PublicKey)); yyxt13 != nil {
							z.EncExtension(x.PublicKey, yyxt13)
						} else if z.EncBinary() {
							z.EncBinaryMarshal(x.PublicKey)
						} else {
							z.EncFallback(x.PublicKey)
						}
					}
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `PublicKey`)
				r.WriteMapElemValue()
				if yyn12 {
					r.EncodeNil()
				} else {
					if x.PublicKey == nil {
						r.EncodeNil()
					} else {
						if false {
						} else if yyxt14 := z.Extension(z.I2Rtid(x.PublicKey)); yyxt14 != nil {
							z.EncExtension(x.PublicKey, yyxt14)
						} else if z.EncBinary() {
							z.EncBinaryMarshal(x.PublicKey)
						} else {
							z.EncFallback(x.PublicKey)
						}
					}
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				yy16 := &x.Nonce
				if false {
				} else if yyxt17 := z.Extension(z.I2Rtid(yy16)); yyxt17 != nil {
					z.EncExtension(yy16, yyxt17)
				} else {
					z.EncFallback(yy16)
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `Nonce`)
				r.WriteMapElemValue()
				yy18 := &x.Nonce
				if false {
				} else if yyxt19 := z.Extension(z.I2Rtid(yy18)); yyxt19 != nil {
					z.EncExtension(yy18, yyxt19)
				} else {
					z.EncFallback(yy18)
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayEnd()
			} else {
				r.WriteMapEnd()
			}
		}
	}
}

func (x *Node) CodecDecodeSelf(d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	if false {
	} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
		z.DecExtension(x, yyxt1)
	} else {
		yyct2 := r.ContainerType()
		if yyct2 == codecSelferValueTypeMap9172 {
			yyl2 := r.ReadMapStart()
			if yyl2 == 0 {
				r.ReadMapEnd()
			} else {
				x.codecDecodeSelfFromMap(yyl2, d)
			}
		} else if yyct2 == codecSelferValueTypeArray9172 {
			yyl2 := r.ReadArrayStart()
			if yyl2 == 0 {
				r.ReadArrayEnd()
			} else {
				x.codecDecodeSelfFromArray(yyl2, d)
			}
		} else {
			panic(errCodecSelferOnlyMapOrArrayEncodeToStruct9172)
		}
	}
}

func (x *Node) codecDecodeSelfFromMap(l int, d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	var yyhl3 bool = l >= 0
	for yyj3 := 0; ; yyj3++ {
		if yyhl3 {
			if yyj3 >= l {
				break
			}
		} else {
			if r.CheckBreak() {
				break
			}
		}
		r.ReadMapElemKey()
		yys3 := z.StringView(r.DecodeStringAsBytes())
		r.ReadMapElemValue()
		switch yys3 {
		case "ID":
			if r.TryDecodeAsNil() {
				x.ID = ""
			} else {
				x.ID.CodecDecodeSelf(d)
			}
		case "Role":
			if r.TryDecodeAsNil() {
				x.Role = 0
			} else {
				x.Role.CodecDecodeSelf(d)
			}
		case "Addr":
			if r.TryDecodeAsNil() {
				x.Addr = ""
			} else {
				x.Addr = (string)(r.DecodeString())
			}
		case "PublicKey":
			if r.TryDecodeAsNil() {
				if true && x.PublicKey != nil {
					x.PublicKey = nil
				}
			} else {
				if x.PublicKey == nil {
					x.PublicKey = new(pkg1_asymmetric.PublicKey)
				}

				if false {
				} else if yyxt8 := z.Extension(z.I2Rtid(x.PublicKey)); yyxt8 != nil {
					z.DecExtension(x.PublicKey, yyxt8)
				} else if z.DecBinary() {
					z.DecBinaryUnmarshal(x.PublicKey)
				} else {
					z.DecFallback(x.PublicKey, false)
				}
			}
		case "Nonce":
			if r.TryDecodeAsNil() {
				x.Nonce = pkg4_cpuminer.Uint256{}
			} else {
				if false {
				} else if yyxt10 := z.Extension(z.I2Rtid(x.Nonce)); yyxt10 != nil {
					z.DecExtension(x.Nonce, yyxt10)
				} else {
					z.DecFallback(&x.Nonce, false)
				}
			}
		default:
			z.DecStructFieldNotFound(-1, yys3)
		} // end switch yys3
	} // end for yyj3
	r.ReadMapEnd()
}

func (x *Node) codecDecodeSelfFromArray(l int, d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	var yyj11 int
	var yyb11 bool
	var yyhl11 bool = l >= 0
	yyj11++
	if yyhl11 {
		yyb11 = yyj11 > l
	} else {
		yyb11 = r.CheckBreak()
	}
	if yyb11 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.ID = ""
	} else {
		x.ID.CodecDecodeSelf(d)
	}
	yyj11++
	if yyhl11 {
		yyb11 = yyj11 > l
	} else {
		yyb11 = r.CheckBreak()
	}
	if yyb11 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Role = 0
	} else {
		x.Role.CodecDecodeSelf(d)
	}
	yyj11++
	if yyhl11 {
		yyb11 = yyj11 > l
	} else {
		yyb11 = r.CheckBreak()
	}
	if yyb11 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Addr = ""
	} else {
		x.Addr = (string)(r.DecodeString())
	}
	yyj11++
	if yyhl11 {
		yyb11 = yyj11 > l
	} else {
		yyb11 = r.CheckBreak()
	}
	if yyb11 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		if true && x.PublicKey != nil {
			x.PublicKey = nil
		}
	} else {
		if x.PublicKey == nil {
			x.PublicKey = new(pkg1_asymmetric.PublicKey)
		}

		if false {
		} else if yyxt16 := z.Extension(z.I2Rtid(x.PublicKey)); yyxt16 != nil {
			z.DecExtension(x.PublicKey, yyxt16)
		} else if z.DecBinary() {
			z.DecBinaryUnmarshal(x.PublicKey)
		} else {
			z.DecFallback(x.PublicKey, false)
		}
	}
	yyj11++
	if yyhl11 {
		yyb11 = yyj11 > l
	} else {
		yyb11 = r.CheckBreak()
	}
	if yyb11 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.Nonce = pkg4_cpuminer.Uint256{}
	} else {
		if false {
		} else if yyxt18 := z.Extension(z.I2Rtid(x.Nonce)); yyxt18 != nil {
			z.DecExtension(x.Nonce, yyxt18)
		} else {
			z.DecFallback(&x.Nonce, false)
		}
	}
	for {
		yyj11++
		if yyhl11 {
			yyb11 = yyj11 > l
		} else {
			yyb11 = r.CheckBreak()
		}
		if yyb11 {
			break
		}
		r.ReadArrayElem()
		z.DecStructFieldNotFound(yyj11-1, "")
	}
	r.ReadArrayEnd()
}

func (x *AddrAndGas) CodecEncodeSelf(e *codec1978.Encoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperEncoder(e)
	_, _, _ = h, z, r
	if x == nil {
		r.EncodeNil()
	} else {
		if false {
		} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
			z.EncExtension(x, yyxt1)
		} else {
			yysep2 := !z.EncBinary()
			yy2arr2 := z.EncBasicHandle().StructToArray
			_, _ = yysep2, yy2arr2
			const yyr2 bool = false // struct tag has 'toArray'
			if yyr2 || yy2arr2 {
				r.WriteArrayStart(3)
			} else {
				r.WriteMapStart(3)
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				yy4 := &x.AccountAddress
				if false {
				} else if yyxt5 := z.Extension(z.I2Rtid(yy4)); yyxt5 != nil {
					z.EncExtension(yy4, yyxt5)
				} else {
					h.encAccountAddress((*AccountAddress)(yy4), e)
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `AccountAddress`)
				r.WriteMapElemValue()
				yy6 := &x.AccountAddress
				if false {
				} else if yyxt7 := z.Extension(z.I2Rtid(yy6)); yyxt7 != nil {
					z.EncExtension(yy6, yyxt7)
				} else {
					h.encAccountAddress((*AccountAddress)(yy6), e)
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				yy9 := &x.RawNodeID
				yy9.CodecEncodeSelf(e)
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `RawNodeID`)
				r.WriteMapElemValue()
				yy11 := &x.RawNodeID
				yy11.CodecEncodeSelf(e)
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayElem()
				if false {
				} else {
					r.EncodeUint(uint64(x.GasAmount))
				}
			} else {
				r.WriteMapElemKey()
				r.EncodeString(codecSelferCcUTF89172, `GasAmount`)
				r.WriteMapElemValue()
				if false {
				} else {
					r.EncodeUint(uint64(x.GasAmount))
				}
			}
			if yyr2 || yy2arr2 {
				r.WriteArrayEnd()
			} else {
				r.WriteMapEnd()
			}
		}
	}
}

func (x *AddrAndGas) CodecDecodeSelf(d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	if false {
	} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
		z.DecExtension(x, yyxt1)
	} else {
		yyct2 := r.ContainerType()
		if yyct2 == codecSelferValueTypeMap9172 {
			yyl2 := r.ReadMapStart()
			if yyl2 == 0 {
				r.ReadMapEnd()
			} else {
				x.codecDecodeSelfFromMap(yyl2, d)
			}
		} else if yyct2 == codecSelferValueTypeArray9172 {
			yyl2 := r.ReadArrayStart()
			if yyl2 == 0 {
				r.ReadArrayEnd()
			} else {
				x.codecDecodeSelfFromArray(yyl2, d)
			}
		} else {
			panic(errCodecSelferOnlyMapOrArrayEncodeToStruct9172)
		}
	}
}

func (x *AddrAndGas) codecDecodeSelfFromMap(l int, d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	var yyhl3 bool = l >= 0
	for yyj3 := 0; ; yyj3++ {
		if yyhl3 {
			if yyj3 >= l {
				break
			}
		} else {
			if r.CheckBreak() {
				break
			}
		}
		r.ReadMapElemKey()
		yys3 := z.StringView(r.DecodeStringAsBytes())
		r.ReadMapElemValue()
		switch yys3 {
		case "AccountAddress":
			if r.TryDecodeAsNil() {
				x.AccountAddress = AccountAddress{}
			} else {
				if false {
				} else if yyxt5 := z.Extension(z.I2Rtid(x.AccountAddress)); yyxt5 != nil {
					z.DecExtension(x.AccountAddress, yyxt5)
				} else {
					h.decAccountAddress((*AccountAddress)(&x.AccountAddress), d)
				}
			}
		case "RawNodeID":
			if r.TryDecodeAsNil() {
				x.RawNodeID = RawNodeID{}
			} else {
				x.RawNodeID.CodecDecodeSelf(d)
			}
		case "GasAmount":
			if r.TryDecodeAsNil() {
				x.GasAmount = 0
			} else {
				x.GasAmount = (uint64)(r.DecodeUint64())
			}
		default:
			z.DecStructFieldNotFound(-1, yys3)
		} // end switch yys3
	} // end for yyj3
	r.ReadMapEnd()
}

func (x *AddrAndGas) codecDecodeSelfFromArray(l int, d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	var yyj8 int
	var yyb8 bool
	var yyhl8 bool = l >= 0
	yyj8++
	if yyhl8 {
		yyb8 = yyj8 > l
	} else {
		yyb8 = r.CheckBreak()
	}
	if yyb8 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.AccountAddress = AccountAddress{}
	} else {
		if false {
		} else if yyxt10 := z.Extension(z.I2Rtid(x.AccountAddress)); yyxt10 != nil {
			z.DecExtension(x.AccountAddress, yyxt10)
		} else {
			h.decAccountAddress((*AccountAddress)(&x.AccountAddress), d)
		}
	}
	yyj8++
	if yyhl8 {
		yyb8 = yyj8 > l
	} else {
		yyb8 = r.CheckBreak()
	}
	if yyb8 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.RawNodeID = RawNodeID{}
	} else {
		x.RawNodeID.CodecDecodeSelf(d)
	}
	yyj8++
	if yyhl8 {
		yyb8 = yyj8 > l
	} else {
		yyb8 = r.CheckBreak()
	}
	if yyb8 {
		r.ReadArrayEnd()
		return
	}
	r.ReadArrayElem()
	if r.TryDecodeAsNil() {
		x.GasAmount = 0
	} else {
		x.GasAmount = (uint64)(r.DecodeUint64())
	}
	for {
		yyj8++
		if yyhl8 {
			yyb8 = yyj8 > l
		} else {
			yyb8 = r.CheckBreak()
		}
		if yyb8 {
			break
		}
		r.ReadArrayElem()
		z.DecStructFieldNotFound(yyj8-1, "")
	}
	r.ReadArrayEnd()
}

func (x ServerRole) CodecEncodeSelf(e *codec1978.Encoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperEncoder(e)
	_, _, _ = h, z, r
	if false {
	} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
		z.EncExtension(x, yyxt1)
	} else {
		r.EncodeInt(int64(x))
	}
}

func (x *ServerRole) CodecDecodeSelf(d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	if false {
	} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
		z.DecExtension(x, yyxt1)
	} else {
		*x = (ServerRole)(z.C.IntV(r.DecodeInt64(), codecSelferBitsize9172))
	}
}

func (x ServerRoles) CodecEncodeSelf(e *codec1978.Encoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperEncoder(e)
	_, _, _ = h, z, r
	if x == nil {
		r.EncodeNil()
	} else {
		if false {
		} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
			z.EncExtension(x, yyxt1)
		} else {
			h.encServerRoles((ServerRoles)(x), e)
		}
	}
}

func (x *ServerRoles) CodecDecodeSelf(d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	if false {
	} else if yyxt1 := z.Extension(z.I2Rtid(x)); yyxt1 != nil {
		z.DecExtension(x, yyxt1)
	} else {
		h.decServerRoles((*ServerRoles)(x), d)
	}
}

func (x codecSelfer9172) encSliceSliceuint8(v [][]uint8, e *codec1978.Encoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperEncoder(e)
	_, _, _ = h, z, r
	r.WriteArrayStart(len(v))
	for _, yyv1 := range v {
		r.WriteArrayElem()
		if yyv1 == nil {
			r.EncodeNil()
		} else {
			if false {
			} else {
				r.EncodeStringBytes(codecSelferCcRAW9172, []byte(yyv1))
			}
		}
	}
	r.WriteArrayEnd()
}

func (x codecSelfer9172) decSliceSliceuint8(v *[][]uint8, d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r

	yyv1 := *v
	yyh1, yyl1 := z.DecSliceHelperStart()
	var yyc1 bool
	_ = yyc1
	if yyl1 == 0 {
		if yyv1 == nil {
			yyv1 = [][]uint8{}
			yyc1 = true
		} else if len(yyv1) != 0 {
			yyv1 = yyv1[:0]
			yyc1 = true
		}
	} else {
		yyhl1 := yyl1 > 0
		var yyrl1 int
		_ = yyrl1
		if yyhl1 {
			if yyl1 > cap(yyv1) {
				yyrl1 = z.DecInferLen(yyl1, z.DecBasicHandle().MaxInitLen, 24)
				if yyrl1 <= cap(yyv1) {
					yyv1 = yyv1[:yyrl1]
				} else {
					yyv1 = make([][]uint8, yyrl1)
				}
				yyc1 = true
			} else if yyl1 != len(yyv1) {
				yyv1 = yyv1[:yyl1]
				yyc1 = true
			}
		}
		var yyj1 int
		// var yydn1 bool
		for ; (yyhl1 && yyj1 < yyl1) || !(yyhl1 || r.CheckBreak()); yyj1++ {
			if yyj1 == 0 && yyv1 == nil {
				if yyhl1 {
					yyrl1 = z.DecInferLen(yyl1, z.DecBasicHandle().MaxInitLen, 24)
				} else {
					yyrl1 = 8
				}
				yyv1 = make([][]uint8, yyrl1)
				yyc1 = true
			}
			yyh1.ElemContainerState(yyj1)

			var yydb1 bool
			if yyj1 >= len(yyv1) {
				yyv1 = append(yyv1, nil)
				yyc1 = true

			}
			if yydb1 {
				z.DecSwallow()
			} else {
				if r.TryDecodeAsNil() {
					yyv1[yyj1] = nil
				} else {
					if false {
					} else {
						yyv1[yyj1] = r.DecodeBytes(([]byte)(yyv1[yyj1]), false)
					}
				}

			}

		}
		if yyj1 < len(yyv1) {
			yyv1 = yyv1[:yyj1]
			yyc1 = true
		} else if yyj1 == 0 && yyv1 == nil {
			yyv1 = make([][]uint8, 0)
			yyc1 = true
		}
	}
	yyh1.End()
	if yyc1 {
		*v = yyv1
	}
}

func (x codecSelfer9172) encSliceuint8(v []uint8, e *codec1978.Encoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperEncoder(e)
	_, _, _ = h, z, r
	r.EncodeStringBytes(codecSelferCcRAW9172, []byte(v))
}

func (x codecSelfer9172) decSliceuint8(v *[]uint8, d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	*v = r.DecodeBytes(*((*[]byte)(v)), false)
}

func (x codecSelfer9172) encSliceServerRole(v []ServerRole, e *codec1978.Encoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperEncoder(e)
	_, _, _ = h, z, r
	r.WriteArrayStart(len(v))
	for _, yyv1 := range v {
		r.WriteArrayElem()
		yyv1.CodecEncodeSelf(e)
	}
	r.WriteArrayEnd()
}

func (x codecSelfer9172) decSliceServerRole(v *[]ServerRole, d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r

	yyv1 := *v
	yyh1, yyl1 := z.DecSliceHelperStart()
	var yyc1 bool
	_ = yyc1
	if yyl1 == 0 {
		if yyv1 == nil {
			yyv1 = []ServerRole{}
			yyc1 = true
		} else if len(yyv1) != 0 {
			yyv1 = yyv1[:0]
			yyc1 = true
		}
	} else {
		yyhl1 := yyl1 > 0
		var yyrl1 int
		_ = yyrl1
		if yyhl1 {
			if yyl1 > cap(yyv1) {
				yyrl1 = z.DecInferLen(yyl1, z.DecBasicHandle().MaxInitLen, 8)
				if yyrl1 <= cap(yyv1) {
					yyv1 = yyv1[:yyrl1]
				} else {
					yyv1 = make([]ServerRole, yyrl1)
				}
				yyc1 = true
			} else if yyl1 != len(yyv1) {
				yyv1 = yyv1[:yyl1]
				yyc1 = true
			}
		}
		var yyj1 int
		// var yydn1 bool
		for ; (yyhl1 && yyj1 < yyl1) || !(yyhl1 || r.CheckBreak()); yyj1++ {
			if yyj1 == 0 && yyv1 == nil {
				if yyhl1 {
					yyrl1 = z.DecInferLen(yyl1, z.DecBasicHandle().MaxInitLen, 8)
				} else {
					yyrl1 = 8
				}
				yyv1 = make([]ServerRole, yyrl1)
				yyc1 = true
			}
			yyh1.ElemContainerState(yyj1)

			var yydb1 bool
			if yyj1 >= len(yyv1) {
				yyv1 = append(yyv1, 0)
				yyc1 = true

			}
			if yydb1 {
				z.DecSwallow()
			} else {
				if r.TryDecodeAsNil() {
					yyv1[yyj1] = 0
				} else {
					yyv1[yyj1].CodecDecodeSelf(d)
				}

			}

		}
		if yyj1 < len(yyv1) {
			yyv1 = yyv1[:yyj1]
			yyc1 = true
		} else if yyj1 == 0 && yyv1 == nil {
			yyv1 = make([]ServerRole, 0)
			yyc1 = true
		}
	}
	yyh1.End()
	if yyc1 {
		*v = yyv1
	}
}

func (x codecSelfer9172) encSliceNode(v []Node, e *codec1978.Encoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperEncoder(e)
	_, _, _ = h, z, r
	r.WriteArrayStart(len(v))
	for _, yyv1 := range v {
		r.WriteArrayElem()
		yy2 := &yyv1
		yy2.CodecEncodeSelf(e)
	}
	r.WriteArrayEnd()
}

func (x codecSelfer9172) decSliceNode(v *[]Node, d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r

	yyv1 := *v
	yyh1, yyl1 := z.DecSliceHelperStart()
	var yyc1 bool
	_ = yyc1
	if yyl1 == 0 {
		if yyv1 == nil {
			yyv1 = []Node{}
			yyc1 = true
		} else if len(yyv1) != 0 {
			yyv1 = yyv1[:0]
			yyc1 = true
		}
	} else {
		yyhl1 := yyl1 > 0
		var yyrl1 int
		_ = yyrl1
		if yyhl1 {
			if yyl1 > cap(yyv1) {
				yyrl1 = z.DecInferLen(yyl1, z.DecBasicHandle().MaxInitLen, 80)
				if yyrl1 <= cap(yyv1) {
					yyv1 = yyv1[:yyrl1]
				} else {
					yyv1 = make([]Node, yyrl1)
				}
				yyc1 = true
			} else if yyl1 != len(yyv1) {
				yyv1 = yyv1[:yyl1]
				yyc1 = true
			}
		}
		var yyj1 int
		// var yydn1 bool
		for ; (yyhl1 && yyj1 < yyl1) || !(yyhl1 || r.CheckBreak()); yyj1++ {
			if yyj1 == 0 && yyv1 == nil {
				if yyhl1 {
					yyrl1 = z.DecInferLen(yyl1, z.DecBasicHandle().MaxInitLen, 80)
				} else {
					yyrl1 = 8
				}
				yyv1 = make([]Node, yyrl1)
				yyc1 = true
			}
			yyh1.ElemContainerState(yyj1)

			var yydb1 bool
			if yyj1 >= len(yyv1) {
				yyv1 = append(yyv1, Node{})
				yyc1 = true

			}
			if yydb1 {
				z.DecSwallow()
			} else {
				if r.TryDecodeAsNil() {
					yyv1[yyj1] = Node{}
				} else {
					yyv1[yyj1].CodecDecodeSelf(d)
				}

			}

		}
		if yyj1 < len(yyv1) {
			yyv1 = yyv1[:yyj1]
			yyc1 = true
		} else if yyj1 == 0 && yyv1 == nil {
			yyv1 = make([]Node, 0)
			yyc1 = true
		}
	}
	yyh1.End()
	if yyc1 {
		*v = yyv1
	}
}

func (x codecSelfer9172) enchash_Hash(v *pkg5_hash.Hash, e *codec1978.Encoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperEncoder(e)
	_, _, _ = h, z, r
	r.EncodeStringBytes(codecSelferCcRAW9172, ((*[32]byte)(v))[:])
}

func (x codecSelfer9172) dechash_Hash(v *pkg5_hash.Hash, d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	r.DecodeBytes(((*[32]byte)(v))[:], true)
}

func (x codecSelfer9172) encAccountAddress(v *AccountAddress, e *codec1978.Encoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperEncoder(e)
	_, _, _ = h, z, r
	r.EncodeStringBytes(codecSelferCcRAW9172, ((*[32]byte)(v))[:])
}

func (x codecSelfer9172) decAccountAddress(v *AccountAddress, d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r
	r.DecodeBytes(((*[32]byte)(v))[:], true)
}

func (x codecSelfer9172) encServerRoles(v ServerRoles, e *codec1978.Encoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperEncoder(e)
	_, _, _ = h, z, r
	r.WriteArrayStart(len(v))
	for _, yyv1 := range v {
		r.WriteArrayElem()
		yyv1.CodecEncodeSelf(e)
	}
	r.WriteArrayEnd()
}

func (x codecSelfer9172) decServerRoles(v *ServerRoles, d *codec1978.Decoder) {
	var h codecSelfer9172
	z, r := codec1978.GenHelperDecoder(d)
	_, _, _ = h, z, r

	yyv1 := *v
	yyh1, yyl1 := z.DecSliceHelperStart()
	var yyc1 bool
	_ = yyc1
	if yyl1 == 0 {
		if yyv1 == nil {
			yyv1 = []ServerRole{}
			yyc1 = true
		} else if len(yyv1) != 0 {
			yyv1 = yyv1[:0]
			yyc1 = true
		}
	} else {
		yyhl1 := yyl1 > 0
		var yyrl1 int
		_ = yyrl1
		if yyhl1 {
			if yyl1 > cap(yyv1) {
				yyrl1 = z.DecInferLen(yyl1, z.DecBasicHandle().MaxInitLen, 8)
				if yyrl1 <= cap(yyv1) {
					yyv1 = yyv1[:yyrl1]
				} else {
					yyv1 = make([]ServerRole, yyrl1)
				}
				yyc1 = true
			} else if yyl1 != len(yyv1) {
				yyv1 = yyv1[:yyl1]
				yyc1 = true
			}
		}
		var yyj1 int
		// var yydn1 bool
		for ; (yyhl1 && yyj1 < yyl1) || !(yyhl1 || r.CheckBreak()); yyj1++ {
			if yyj1 == 0 && yyv1 == nil {
				if yyhl1 {
					yyrl1 = z.DecInferLen(yyl1, z.DecBasicHandle().MaxInitLen, 8)
				} else {
					yyrl1 = 8
				}
				yyv1 = make([]ServerRole, yyrl1)
				yyc1 = true
			}
			yyh1.ElemContainerState(yyj1)

			var yydb1 bool
			if yyj1 >= len(yyv1) {
				yyv1 = append(yyv1, 0)
				yyc1 = true

			}
			if yydb1 {
				z.DecSwallow()
			} else {
				if r.TryDecodeAsNil() {
					yyv1[yyj1] = 0
				} else {
					yyv1[yyj1].CodecDecodeSelf(d)
				}

			}

		}
		if yyj1 < len(yyv1) {
			yyv1 = yyv1[:yyj1]
			yyc1 = true
		} else if yyj1 == 0 && yyv1 == nil {
			yyv1 = make([]ServerRole, 0)
			yyc1 = true
		}
	}
	yyh1.End()
	if yyc1 {
		*v = yyv1
	}
}
