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
	"testing"
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"
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

			Convey("header change without signing", func() {
				req.Header.Timestamp = req.Header.Timestamp.Add(time.Second)

				buildHash(&req.Header.RequestHeader, &req.Header.Hash)
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
	privKey, _ := getCommKeys()

	Convey("sign", t, func() {
		res := &Response{
			Header: SignedResponseHeader{
				ResponseHeader: ResponseHeader{
					Request: SignedRequestHeader{
						RequestHeader: RequestHeader{
							QueryType:    WriteQuery,
							NodeID:       proto.NodeID("0000000000000000000000000000000000000000000000000000000000001111"),
							DatabaseID:   proto.DatabaseID("db1"),
							ConnectionID: uint64(1),
							SeqNo:        uint64(2),
							Timestamp:    time.Now().UTC(),
						},
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
				},
				DeclTypes: []string{
					"INTEGER",
					"BOOLEAN",
					"DATETIME",
					"INTEGER",
					"FLOAT",
					"BLOB",
					"TEXT",
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
						},
					},
				},
			},
		}

		var err error

		// sign directly, embedded original request is not filled
		err = res.Sign(privKey)
		So(err, ShouldNotBeNil)
		So(errors.Cause(err), ShouldBeIn, []error{
			ErrSignVerification,
			ErrHashVerification,
		})

		// sign original request first
		err = res.Header.Request.Sign(privKey)
		So(err, ShouldBeNil)

		// sign again
		err = res.Sign(privKey)
		So(err, ShouldBeNil)

		// test hash
		err = verifyHash(&res.Payload, &res.Header.DataHash)
		So(err, ShouldBeNil)

		// verify
		Convey("verify", func() {
			err = res.Verify()
			So(err, ShouldBeNil)

			Convey("request change", func() {
				res.Header.Request.BatchCount = 200

				err = res.Verify()
				So(err, ShouldNotBeNil)
			})
			Convey("payload change", func() {
				res.Payload.DeclTypes[0] = "INT"

				err = res.Verify()
				So(err, ShouldNotBeNil)
			})
			Convey("header change", func() {
				res.Header.Timestamp = res.Header.Timestamp.Add(time.Second)

				err = res.Verify()
				So(err, ShouldNotBeNil)
			})
			Convey("header change without signing", func() {
				res.Header.Timestamp = res.Header.Timestamp.Add(time.Second)
				buildHash(&res.Header.ResponseHeader, &res.Header.Hash)

				err = res.Verify()
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
					Response: SignedResponseHeader{
						ResponseHeader: ResponseHeader{
							Request: SignedRequestHeader{
								RequestHeader: RequestHeader{
									QueryType:    WriteQuery,
									NodeID:       proto.NodeID("0000000000000000000000000000000000000000000000000000000000001111"),
									DatabaseID:   proto.DatabaseID("db1"),
									ConnectionID: uint64(1),
									SeqNo:        uint64(2),
									Timestamp:    time.Now().UTC(),
								},
							},
							NodeID:    proto.NodeID("0000000000000000000000000000000000000000000000000000000000002222"),
							Timestamp: time.Now().UTC(),
							RowCount:  uint64(1),
						},
					},
					NodeID:    proto.NodeID("0000000000000000000000000000000000000000000000000000000000001111"),
					Timestamp: time.Now().UTC(),
				},
			},
		}

		var err error

		Convey("get query key", func() {
			key := ack.Header.SignedRequestHeader().GetQueryKey()
			So(key.NodeID, ShouldEqual, ack.Header.SignedRequestHeader().NodeID)
			So(key.ConnectionID, ShouldEqual, ack.Header.SignedRequestHeader().ConnectionID)
			So(key.SeqNo, ShouldEqual, ack.Header.SignedRequestHeader().SeqNo)
		})

		// sign directly, embedded original response is not filled
		err = ack.Sign(privKey, false)
		So(err, ShouldBeNil)
		err = ack.Sign(privKey, true)
		So(err, ShouldNotBeNil)
		So(err, ShouldBeIn, []error{
			ErrSignVerification,
			ErrHashVerification,
		})

		// sign nested structure, step by step
		// this is not required during runtime
		// during runtime, nested structures is signed and provided by peers
		err = ack.Header.Response.Request.Sign(privKey)
		So(err, ShouldBeNil)
		err = ack.Header.Response.Sign(privKey)
		So(err, ShouldBeNil)
		err = ack.Sign(privKey, true)
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
			Convey("header change without signing", func() {
				ack.Header.Timestamp = ack.Header.Timestamp.Add(time.Second)

				buildHash(&ack.Header.AckHeader, &ack.Header.Hash)

				err = ack.Verify()
				So(err, ShouldNotBeNil)
			})
		})
	})
}

func TestNoAckReport_Sign(t *testing.T) {
	privKey, _ := getCommKeys()

	Convey("sign", t, func() {
		noAck := &NoAckReport{
			Header: SignedNoAckReportHeader{
				NoAckReportHeader: NoAckReportHeader{
					NodeID:    proto.NodeID("0000000000000000000000000000000000000000000000000000000000002222"),
					Timestamp: time.Now().UTC(),
					Response: SignedResponseHeader{
						ResponseHeader: ResponseHeader{
							Request: SignedRequestHeader{
								RequestHeader: RequestHeader{
									QueryType:    WriteQuery,
									NodeID:       proto.NodeID("0000000000000000000000000000000000000000000000000000000000001111"),
									DatabaseID:   proto.DatabaseID("db1"),
									ConnectionID: uint64(1),
									SeqNo:        uint64(2),
									Timestamp:    time.Now().UTC(),
								},
							},
							NodeID:    proto.NodeID("0000000000000000000000000000000000000000000000000000000000002222"),
							Timestamp: time.Now().UTC(),
							RowCount:  uint64(1),
						},
					},
				},
			},
		}

		var err error

		// sign directly, embedded original response/request is not filled
		err = noAck.Sign(privKey)
		So(err, ShouldNotBeNil)
		So(err, ShouldBeIn, []error{
			ErrSignVerification,
			ErrHashVerification,
		})

		// sign nested structure
		err = noAck.Header.Response.Request.Sign(privKey)
		So(err, ShouldBeNil)
		err = noAck.Header.Response.Sign(privKey)
		So(err, ShouldBeNil)
		err = noAck.Sign(privKey)
		So(err, ShouldBeNil)

		Convey("verify", func() {
			err = noAck.Verify()
			So(err, ShouldBeNil)

			Convey("request change", func() {
				noAck.Header.Response.Request.QueryType = ReadQuery

				err = noAck.Verify()
				So(err, ShouldNotBeNil)
			})

			Convey("response change", func() {
				noAck.Header.Response.RowCount = 100

				err = noAck.Verify()
				So(err, ShouldNotBeNil)
			})

			Convey("header change", func() {
				noAck.Header.Timestamp = noAck.Header.Timestamp.Add(time.Second)

				err = noAck.Verify()
				So(err, ShouldNotBeNil)
			})

			Convey("header change without signing", func() {
				noAck.Header.Timestamp = noAck.Header.Timestamp.Add(time.Second)

				buildHash(&noAck.Header.NoAckReportHeader, &noAck.Header.Hash)

				err = noAck.Verify()
				So(err, ShouldNotBeNil)
			})
		})
	})
}

func TestAggrNoAckReport_Sign(t *testing.T) {
	privKey, _ := getCommKeys()

	Convey("sign", t, func() {
		aggrNoAck := &AggrNoAckReport{
			Header: SignedAggrNoAckReportHeader{
				AggrNoAckReportHeader: AggrNoAckReportHeader{
					NodeID:    proto.NodeID("0000000000000000000000000000000000000000000000000000000000003333"),
					Timestamp: time.Now().UTC(),
					Reports: []SignedNoAckReportHeader{
						{
							NoAckReportHeader: NoAckReportHeader{
								NodeID:    proto.NodeID("0000000000000000000000000000000000000000000000000000000000002222"),
								Timestamp: time.Now().UTC(),
								Response: SignedResponseHeader{
									ResponseHeader: ResponseHeader{
										Request: SignedRequestHeader{
											RequestHeader: RequestHeader{
												QueryType:    WriteQuery,
												NodeID:       proto.NodeID("0000000000000000000000000000000000000000000000000000000000001111"),
												DatabaseID:   proto.DatabaseID("db1"),
												ConnectionID: uint64(1),
												SeqNo:        uint64(2),
												Timestamp:    time.Now().UTC(),
											},
										},
										NodeID:    proto.NodeID("0000000000000000000000000000000000000000000000000000000000002222"),
										Timestamp: time.Now().UTC(),
										RowCount:  uint64(1),
									},
								},
							},
						},
						{
							NoAckReportHeader: NoAckReportHeader{
								NodeID:    proto.NodeID("0000000000000000000000000000000000000000000000000000000000003333"),
								Timestamp: time.Now().UTC(),
								Response: SignedResponseHeader{
									ResponseHeader: ResponseHeader{
										Request: SignedRequestHeader{
											RequestHeader: RequestHeader{
												QueryType:    WriteQuery,
												NodeID:       proto.NodeID("0000000000000000000000000000000000000000000000000000000000001111"),
												DatabaseID:   proto.DatabaseID("db1"),
												ConnectionID: uint64(1),
												SeqNo:        uint64(2),
												Timestamp:    time.Now().UTC(),
											},
										},
										NodeID:    proto.NodeID("0000000000000000000000000000000000000000000000000000000000003333"),
										Timestamp: time.Now().UTC(),
										RowCount:  uint64(1),
									},
								},
							},
						},
					},
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
				},
			},
		}

		var err error

		// sign directly, embedded original response/request is not filled
		err = aggrNoAck.Sign(privKey)
		So(err, ShouldNotBeNil)
		So(err, ShouldBeIn, []error{
			ErrSignVerification,
			ErrHashVerification,
		})

		// sign nested structure
		err = aggrNoAck.Header.Reports[0].Response.Request.Sign(privKey)
		So(err, ShouldBeNil)
		err = aggrNoAck.Header.Reports[1].Response.Request.Sign(privKey)
		So(err, ShouldBeNil)
		err = aggrNoAck.Header.Reports[0].Response.Sign(privKey)
		So(err, ShouldBeNil)
		err = aggrNoAck.Header.Reports[1].Response.Sign(privKey)
		So(err, ShouldBeNil)
		err = aggrNoAck.Header.Reports[0].Sign(privKey)
		So(err, ShouldBeNil)
		err = aggrNoAck.Header.Reports[1].Sign(privKey)
		So(err, ShouldBeNil)
		err = aggrNoAck.Sign(privKey)
		So(err, ShouldBeNil)

		Convey("verify", func() {
			err = aggrNoAck.Verify()
			So(err, ShouldBeNil)

			Convey("request change", func() {
				aggrNoAck.Header.Reports[0].Response.Request.QueryType = ReadQuery

				err = aggrNoAck.Verify()
				So(err, ShouldNotBeNil)
			})

			Convey("response change", func() {
				aggrNoAck.Header.Reports[0].Response.RowCount = 1000

				err = aggrNoAck.Verify()
				So(err, ShouldNotBeNil)
			})

			Convey("report change", func() {
				aggrNoAck.Header.Reports[0].Timestamp = aggrNoAck.Header.Reports[0].Timestamp.Add(time.Second)

				err = aggrNoAck.Verify()
				So(err, ShouldNotBeNil)
			})

			Convey("header change", func() {
				aggrNoAck.Header.Timestamp = aggrNoAck.Header.Timestamp.Add(time.Second)

				err = aggrNoAck.Verify()
				So(err, ShouldNotBeNil)
			})

			Convey("header change without signing", func() {
				aggrNoAck.Header.Timestamp = aggrNoAck.Header.Timestamp.Add(time.Second)

				buildHash(&aggrNoAck.Header.AggrNoAckReportHeader, &aggrNoAck.Header.Hash)

				err = aggrNoAck.Verify()
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

			Convey("header change without signing", func() {
				initServiceResponse.Header.Instances[0].DatabaseID = proto.DatabaseID("db2")

				buildHash(&initServiceResponse.Header.InitServiceResponseHeader, &initServiceResponse.Header.Hash)

				s, err := initServiceResponse.Header.InitServiceResponseHeader.MarshalHash()
				So(err, ShouldBeNil)
				So(s, ShouldNotBeEmpty)

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

			Convey("header change without signing", func() {
				updateServiceReq.Header.Instance.DatabaseID = proto.DatabaseID("db2")
				buildHash(&updateServiceReq.Header.UpdateServiceHeader, &updateServiceReq.Header.Hash)

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
