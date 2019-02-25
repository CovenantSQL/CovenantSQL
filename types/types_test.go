/*
 * Copyright 2018 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package types

import (
	"bytes"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/ugorji/go/codec"
)

func getCommKeys() (*asymmetric.PrivateKey, *asymmetric.PublicKey) {
	testPriv := []byte{
		0xea, 0xf0, 0x2c, 0xa3, 0x48, 0xc5, 0x24, 0xe6,
		0x39, 0x26, 0x55, 0xba, 0x4d, 0x29, 0x60, 0x3c,
		0xd1, 0xa7, 0x34, 0x7d, 0x9d, 0x65, 0xcf, 0xe9,
		0x3c, 0xe1, 0xeb, 0xff, 0xdc, 0xa2, 0x26, 0x94,
	}
	return asymmetric.PrivKeyFromBytes(testPriv)
}

func TestSignedRequestHeader_Sign(t *testing.T) {
	privKey, _ := getCommKeys()

	Convey("sign", t, func() {
		req := &SignedRequestHeader{
			RequestHeader: RequestHeader{
				QueryType:    WriteQuery,
				NodeID:       proto.NodeID("node"),
				DatabaseID:   proto.DatabaseID("db1"),
				ConnectionID: uint64(1),
				SeqNo:        uint64(2),
				Timestamp:    time.Now().UTC(),
			},
		}

		var err error

		err = req.Sign(privKey)
		So(err, ShouldBeNil)

		Convey("verify", func() {
			err = req.Verify()
			So(err, ShouldBeNil)

			// modify structure
			req.Timestamp = req.Timestamp.Add(time.Second)

			err = req.Verify()
			So(err, ShouldNotBeNil)

			s, err := req.MarshalHash()
			So(err, ShouldBeNil)
			So(s, ShouldNotBeEmpty)
		})
	})
}

func TestRequest_Sign(t *testing.T) {
	privKey, _ := getCommKeys()

	Convey("sign", t, func() {
		req := &Request{
			Header: SignedRequestHeader{
				RequestHeader: RequestHeader{
					QueryType:    WriteQuery,
					NodeID:       proto.NodeID("node"),
					DatabaseID:   proto.DatabaseID("db1"),
					ConnectionID: uint64(1),
					SeqNo:        uint64(2),
					Timestamp:    time.Now().UTC(),
				},
			},
			Payload: RequestPayload{
				Queries: []Query{
					{
						Pattern: "INSERT INTO test VALUES(?)",
						Args: []NamedArg{
							{
								Value: 1,
							},
						},
					},
					{
						Pattern: "INSERT INTO test VALUES(?)",
						Args: []NamedArg{
							{
								Value: "happy",
							},
						},
					},
				},
			},
		}

		var err error

		// sign
		err = req.Sign(privKey)
		So(err, ShouldBeNil)
		So(req.Header.BatchCount, ShouldEqual, uint64(len(req.Payload.Queries)))

		// test queries hash
		err = verifyHash(&req.Payload, &req.Header.QueriesHash)
		So(err, ShouldBeNil)

		Convey("verify", func() {
			err = req.Verify()
			So(err, ShouldBeNil)

			Convey("header change", func() {
				// modify structure
				req.Header.Timestamp = req.Header.Timestamp.Add(time.Second)

				err = req.Verify()
				So(err, ShouldNotBeNil)
			})

			Convey("header change with invalid queries hash", func() {
				req.Payload.Queries = append(req.Payload.Queries,
					Query{
						Pattern: "select 1",
					},
				)

				err = req.Verify()
				So(err, ShouldNotBeNil)
			})
		})
	})
}

func TestResponse_Sign(t *testing.T) {
	Convey("sign", t, func() {
		res := &Response{
			Header: SignedResponseHeader{
				ResponseHeader: ResponseHeader{
					Request: RequestHeader{
						QueryType:    WriteQuery,
						NodeID:       proto.NodeID("0000000000000000000000000000000000000000000000000000000000001111"),
						DatabaseID:   proto.DatabaseID("db1"),
						ConnectionID: uint64(1),
						SeqNo:        uint64(2),
						Timestamp:    time.Now().UTC(),
					},
					NodeID:    proto.NodeID("0000000000000000000000000000000000000000000000000000000000002222"),
					Timestamp: time.Now().UTC(),
					RowCount:  uint64(1),
				},
			},
			Payload: ResponsePayload{
				Columns: []string{
					"test_integer",
					"test_boolean",
					"test_time",
					"test_nil",
					"test_float",
					"test_binary_string",
					"test_string",
					"test_empty_time",
				},
				DeclTypes: []string{
					"INTEGER",
					"BOOLEAN",
					"DATETIME",
					"INTEGER",
					"FLOAT",
					"BLOB",
					"TEXT",
					"DATETIME",
				},
				Rows: []ResponseRow{
					{
						Values: []interface{}{
							int(1),
							true,
							time.Now().UTC(),
							nil,
							float64(1.0001),
							"11111\0001111111",
							"11111111111111",
							time.Time{},
						},
					},
				},
			},
		}

		var err error

		// sign
		err = res.BuildHash()
		So(err, ShouldBeNil)

		// test hash
		err = verifyHash(&res.Payload, &res.Header.PayloadHash)
		So(err, ShouldBeNil)

		// verify
		Convey("verify", func() {
			err = res.BuildHash()
			So(err, ShouldBeNil)

			Convey("encode/decode verify", func() {
				buf, err := utils.EncodeMsgPack(res)
				So(err, ShouldBeNil)
				var r *Response
				err = utils.DecodeMsgPack(buf.Bytes(), &r)
				So(err, ShouldBeNil)
				err = r.VerifyHash()
				So(err, ShouldBeNil)
			})
			Convey("request change", func() {
				res.Header.Request.BatchCount = 200

				err = res.VerifyHash()
				So(err, ShouldNotBeNil)
			})
			Convey("payload change", func() {
				res.Payload.DeclTypes[0] = "INT"

				err = res.VerifyHash()
				So(err, ShouldNotBeNil)
			})
			Convey("header change", func() {
				res.Header.Timestamp = res.Header.Timestamp.Add(time.Second)

				err = res.VerifyHash()
				So(err, ShouldNotBeNil)
			})
		})
	})
}

func TestAck_Sign(t *testing.T) {
	privKey, _ := getCommKeys()

	Convey("sign", t, func() {
		ack := &Ack{
			Header: SignedAckHeader{
				AckHeader: AckHeader{
					Response: ResponseHeader{
						Request: RequestHeader{
							QueryType:    WriteQuery,
							NodeID:       proto.NodeID("0000000000000000000000000000000000000000000000000000000000001111"),
							DatabaseID:   proto.DatabaseID("db1"),
							ConnectionID: uint64(1),
							SeqNo:        uint64(2),
							Timestamp:    time.Now().UTC(),
						},
						NodeID:    proto.NodeID("0000000000000000000000000000000000000000000000000000000000002222"),
						Timestamp: time.Now().UTC(),
						RowCount:  uint64(1),
					},
					NodeID:    proto.NodeID("0000000000000000000000000000000000000000000000000000000000001111"),
					Timestamp: time.Now().UTC(),
				},
			},
		}

		var err error

		Convey("get query key", func() {
			key := ack.Header.GetQueryKey()
			So(key.NodeID, ShouldEqual, ack.Header.GetQueryKey().NodeID)
			So(key.ConnectionID, ShouldEqual, ack.Header.GetQueryKey().ConnectionID)
			So(key.SeqNo, ShouldEqual, ack.Header.GetQueryKey().SeqNo)
		})

		err = ack.Sign(privKey)
		So(err, ShouldBeNil)

		Convey("verify", func() {
			err = ack.Verify()
			So(err, ShouldBeNil)

			Convey("request change", func() {
				ack.Header.Response.Request.QueryType = ReadQuery

				err = ack.Verify()
				So(err, ShouldNotBeNil)
			})
			Convey("response change", func() {
				ack.Header.Response.RowCount = 100

				err = ack.Verify()
				So(err, ShouldNotBeNil)
			})
			Convey("header change", func() {
				ack.Header.Timestamp = ack.Header.Timestamp.Add(time.Second)

				err = ack.Verify()
				So(err, ShouldNotBeNil)
			})
		})
	})
}

func TestInitServiceResponse_Sign(t *testing.T) {
	privKey, _ := getCommKeys()

	Convey("sign", t, func() {
		var err error

		initServiceResponse := &InitServiceResponse{
			Header: SignedInitServiceResponseHeader{
				InitServiceResponseHeader: InitServiceResponseHeader{
					Instances: []ServiceInstance{
						{
							DatabaseID: proto.DatabaseID("db1"),
							Peers: &proto.Peers{
								PeersHeader: proto.PeersHeader{
									Term:   uint64(1),
									Leader: proto.NodeID("0000000000000000000000000000000000000000000000000000000000003333"),
									Servers: []proto.NodeID{
										proto.NodeID("0000000000000000000000000000000000000000000000000000000000003333"),
										proto.NodeID("0000000000000000000000000000000000000000000000000000000000002222"),
									},
								},
							},
							// TODO(xq262144), should integrated with genesis block serialization test
							GenesisBlock: nil,
						},
					},
				},
			},
		}

		// sign
		err = initServiceResponse.Sign(privKey)

		Convey("verify", func() {
			err = initServiceResponse.Verify()
			So(err, ShouldBeNil)

			Convey("header change", func() {
				initServiceResponse.Header.Instances[0].DatabaseID = proto.DatabaseID("db2")

				err = initServiceResponse.Verify()
				So(err, ShouldNotBeNil)
			})
		})
	})
}

func TestUpdateService_Sign(t *testing.T) {
	privKey, _ := getCommKeys()

	Convey("sign", t, func() {
		var err error

		updateServiceReq := &UpdateService{
			Header: SignedUpdateServiceHeader{
				UpdateServiceHeader: UpdateServiceHeader{
					Op: CreateDB,
					Instance: ServiceInstance{
						DatabaseID: proto.DatabaseID("db1"),
						Peers: &proto.Peers{
							PeersHeader: proto.PeersHeader{
								Term:   uint64(1),
								Leader: proto.NodeID("0000000000000000000000000000000000000000000000000000000000003333"),
								Servers: []proto.NodeID{
									proto.NodeID("0000000000000000000000000000000000000000000000000000000000003333"),
									proto.NodeID("0000000000000000000000000000000000000000000000000000000000002222"),
								},
							},
						},
						// TODO(xq262144), should integrated with genesis block serialization test
						GenesisBlock: nil,
					},
				},
			},
		}

		// sign
		err = updateServiceReq.Sign(privKey)

		Convey("verify", func() {
			err = updateServiceReq.Verify()
			So(err, ShouldBeNil)

			Convey("header change", func() {
				updateServiceReq.Header.Instance.DatabaseID = proto.DatabaseID("db2")

				err = updateServiceReq.Verify()
				So(err, ShouldNotBeNil)
			})
		})
	})
}

func TestOther_MarshalHash(t *testing.T) {
	Convey("marshal hash", t, func() {
		tm := UpdateType(1)
		s, err := tm.MarshalHash()
		So(err, ShouldBeNil)
		So(s, ShouldNotBeEmpty)

		tm2 := QueryType(1)
		s, err = tm2.MarshalHash()
		So(err, ShouldBeNil)
		So(s, ShouldNotBeEmpty)
	})
}

func TestQueryTypeStringer(t *testing.T) {
	Convey("Query type stringer should return expected string", t, func() {
		var cases = [...]struct {
			i fmt.Stringer
			s string
		}{
			{
				i: ReadQuery,
				s: "read",
			}, {
				i: WriteQuery,
				s: "write",
			}, {
				i: QueryType(0xffff),
				s: "unknown",
			},
		}
		for _, v := range cases {
			So(v.s, ShouldEqual, fmt.Sprintf("%v", v.i))
		}
	})
}

func benchmarkEnc(b *testing.B, v interface{}) {
	var (
		h = &codec.MsgpackHandle{
			WriteExt:    true,
			RawToString: true,
		}
		err error
	)
	for i := 0; i < b.N; i++ {
		var enc = codec.NewEncoder(bytes.NewBuffer(nil), h)
		if err = enc.Encode(v); err != nil {
			b.Error(err)
		}
	}
}

func benchmarkDec(b *testing.B, v interface{}) {
	var (
		r   []byte
		err error

		h = &codec.MsgpackHandle{
			WriteExt:    true,
			RawToString: true,
		}
		enc   = codec.NewEncoderBytes(&r, h)
		recvt = reflect.ValueOf(v).Elem().Type()
		recvs = make([]interface{}, b.N)
	)
	// Encode v and make receivers
	if err = enc.Encode(v); err != nil {
		b.Error(err)
	}
	for i := range recvs {
		recvs[i] = reflect.New(recvt).Interface()
	}
	b.ResetTimer()
	// Start benchmark
	for i := 0; i < b.N; i++ {
		var dec = codec.NewDecoder(bytes.NewReader(r), h)
		if err = dec.Decode(recvs[i]); err != nil {
			b.Error(err)
		}
	}
}

type ver interface {
	Verify() error
}

type ser interface {
	Sign(*asymmetric.PrivateKey) error
}

func benchmarkVerify(b *testing.B, v ver) {
	var err error
	for i := 0; i < b.N; i++ {
		if err = v.Verify(); err != nil {
			b.Error(err)
		}
	}
}

func benchmarkSign(b *testing.B, s ser) {
	var err error
	for i := 0; i < b.N; i++ {
		if err = s.Sign(testingPrivateKey); err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkTypes(b *testing.B) {
	var (
		// Build a approximate 1KB request
		req = buildRequest(WriteQuery, []Query{
			buildQuery(
				`INSERT INTO "t1" VALUES (?, ?, ?, ?)`,
				rand.Int(),
				randBytes(333),
				randBytes(333),
				randBytes(333),
			),
		})
		// Build a approximate 1KB response with the previous request header
		resp = buildResponse(
			&req.Header,
			[]string{"k", "v1", "v2", "v3"},
			[]string{"INT", "TEXT", "TEXT", "TEXT"},
			[]ResponseRow{
				{
					Values: []interface{}{
						rand.Int(),
						randBytes(333),
						randBytes(333),
						randBytes(333),
					},
				},
			},
		)
	)
	var subjects = [...]interface{}{req, resp}
	for _, v := range subjects {
		var name = reflect.ValueOf(v).Elem().Type().Name()
		b.Run(fmt.Sprint(name, "Enc"), func(b *testing.B) { benchmarkEnc(b, v) })
		b.Run(fmt.Sprint(name, "Dec"), func(b *testing.B) { benchmarkDec(b, v) })
		if x, ok := v.(ver); ok {
			b.Run(fmt.Sprint(name, "Verify"), func(b *testing.B) { benchmarkVerify(b, x) })
		}
		if x, ok := v.(ser); ok {
			b.Run(fmt.Sprint(name, "Sign"), func(b *testing.B) { benchmarkSign(b, x) })
		}
	}
}
