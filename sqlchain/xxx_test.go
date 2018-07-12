/*
 * Copyright 2018 The ThunderDB Authors.
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

package sqlchain

import (
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.com/thunderdb/ThunderDB/conf"
	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/kayak"
	"gitlab.com/thunderdb/ThunderDB/pow/cpuminer"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/sqlchain/storage"
	ct "gitlab.com/thunderdb/ThunderDB/sqlchain/types"
	wt "gitlab.com/thunderdb/ThunderDB/worker/types"
)

var (
	genesisHash           = hash.Hash{}
	testHeight      int32 = 50
	testDifficulty        = 4
	testMasterKey         = []byte(".9K.sgch!3;C>w0v")
	testDataDir     string
	testPrivKeyFile string
	testPubKeysFile string
	testPrivKey     *asymmetric.PrivateKey
	testPubKey      *asymmetric.PublicKey
)

type nodeProfile struct {
	NodeID     proto.NodeID
	PrivateKey *asymmetric.PrivateKey
	PublicKey  *asymmetric.PublicKey
}

func newRandomNode() (node *nodeProfile, err error) {
	priv, pub, err := asymmetric.GenSecp256k1KeyPair()

	if err != nil {
		return
	}

	node = &nodeProfile{
		PrivateKey: priv,
		PublicKey:  pub,
	}

	createRandomString(10, 10, (*string)(&node.NodeID))
	return
}

func newRandomNodes(n int) (nodes []*nodeProfile, err error) {
	nodes = make([]*nodeProfile, n)

	for i := range nodes {
		if nodes[i], err = newRandomNode(); err != nil {
			return
		}
	}

	return
}

func createRandomString(offset, length int, s *string) {
	buff := make([]byte, rand.Intn(length)+offset)
	rand.Read(buff)
	*s = string(buff)
}

func createRandomStrings(offset, length, soffset, slength int) (s []string) {
	s = make([]string, rand.Intn(length)+offset)

	for i := range s {
		createRandomString(soffset, slength, &s[i])
	}

	return
}

func createRandomStorageQueries(offset, length, soffset, slength int) (qs []storage.Query) {
	qs = make([]storage.Query, rand.Intn(length)+offset)

	for i := range qs {
		createRandomString(soffset, slength, &qs[i].Pattern)
	}

	return
}

func createRandomTimeAfter(now time.Time, maxDelayMillisecond int) time.Time {
	return now.Add(time.Duration(rand.Intn(maxDelayMillisecond)+1) * time.Millisecond)
}

func createRandomQueryRequest(cli *nodeProfile) (r *wt.SignedRequestHeader, err error) {
	req := &wt.Request{
		Header: wt.SignedRequestHeader{
			RequestHeader: wt.RequestHeader{
				QueryType:    wt.QueryType(rand.Intn(2)),
				NodeID:       cli.NodeID,
				ConnectionID: uint64(rand.Int63()),
				SeqNo:        uint64(rand.Int63()),
				Timestamp:    time.Now().UTC(),
				// BatchCount and QueriesHash will be set by req.Sign()
			},
			Signee:    cli.PublicKey,
			Signature: nil,
		},
		Payload: wt.RequestPayload{
			Queries: createRandomStorageQueries(10, 10, 10, 10),
		},
	}

	createRandomString(10, 10, (*string)(&req.Header.DatabaseID))

	if err = req.Sign(cli.PrivateKey); err != nil {
		return
	}

	r = &req.Header
	return
}

func createRandomQueryResponse(cli, worker *nodeProfile) (
	r *wt.SignedResponseHeader, err error,
) {
	req, err := createRandomQueryRequest(cli)

	if err != nil {
		return
	}

	resp := &wt.Response{
		Header: wt.SignedResponseHeader{
			ResponseHeader: wt.ResponseHeader{
				Request:   *req,
				NodeID:    worker.NodeID,
				Timestamp: createRandomTimeAfter(req.Timestamp, 100),
			},
			Signee:    worker.PublicKey,
			Signature: nil,
		},
		Payload: wt.ResponsePayload{
			Columns:   createRandomStrings(10, 10, 10, 10),
			DeclTypes: createRandomStrings(10, 10, 10, 10),
			Rows:      make([]wt.ResponseRow, rand.Intn(10)+10),
		},
	}

	for i := range resp.Payload.Rows {
		s := createRandomStrings(10, 10, 10, 10)
		resp.Payload.Rows[i].Values = make([]interface{}, len(s))
		for j := range resp.Payload.Rows[i].Values {
			resp.Payload.Rows[i].Values[j] = s[j]
		}
	}

	if err = resp.Sign(worker.PrivateKey); err != nil {
		return
	}

	r = &resp.Header
	return
}

func createRandomQueryResponseWithRequest(req *wt.SignedRequestHeader, worker *nodeProfile) (
	r *wt.SignedResponseHeader, err error,
) {
	resp := &wt.Response{
		Header: wt.SignedResponseHeader{
			ResponseHeader: wt.ResponseHeader{
				Request:   *req,
				NodeID:    worker.NodeID,
				Timestamp: createRandomTimeAfter(req.Timestamp, 100),
			},
			Signee:    worker.PublicKey,
			Signature: nil,
		},
		Payload: wt.ResponsePayload{
			Columns:   createRandomStrings(10, 10, 10, 10),
			DeclTypes: createRandomStrings(10, 10, 10, 10),
			Rows:      make([]wt.ResponseRow, rand.Intn(10)+10),
		},
	}

	for i := range resp.Payload.Rows {
		s := createRandomStrings(10, 10, 10, 10)
		resp.Payload.Rows[i].Values = make([]interface{}, len(s))
		for j := range resp.Payload.Rows[i].Values {
			resp.Payload.Rows[i].Values[j] = s[j]
		}
	}

	if err = resp.Sign(worker.PrivateKey); err != nil {
		return
	}

	r = &resp.Header
	return
}

func createRandomQueryAckWithResponse(resp *wt.SignedResponseHeader, cli *nodeProfile) (
	r *wt.SignedAckHeader, err error,
) {
	ack := &wt.Ack{
		Header: wt.SignedAckHeader{
			AckHeader: wt.AckHeader{
				Response:  *resp,
				NodeID:    cli.NodeID,
				Timestamp: createRandomTimeAfter(resp.Timestamp, 100),
			},
			Signee:    cli.PublicKey,
			Signature: nil,
		},
	}

	if err = ack.Sign(cli.PrivateKey); err != nil {
		return
	}

	r = &ack.Header
	return
}

func createRandomQueryAck(cli, worker *nodeProfile) (r *wt.SignedAckHeader, err error) {
	resp, err := createRandomQueryResponse(cli, worker)

	if err != nil {
		return
	}

	ack := &wt.Ack{
		Header: wt.SignedAckHeader{
			AckHeader: wt.AckHeader{
				Response:  *resp,
				NodeID:    cli.NodeID,
				Timestamp: createRandomTimeAfter(resp.Timestamp, 100),
			},
			Signee:    cli.PublicKey,
			Signature: nil,
		},
	}

	if err = ack.Sign(cli.PrivateKey); err != nil {
		return
	}

	r = &ack.Header
	return
}

func createRandomNodesAndAck() (r *wt.SignedAckHeader, err error) {
	cli, err := newRandomNode()

	if err != nil {
		return
	}

	worker, err := newRandomNode()

	if err != nil {
		return
	}

	return createRandomQueryAck(cli, worker)
}

func registerNodesWithPublicKey(pub *asymmetric.PublicKey, diff int, num int) (
	nis []cpuminer.NonceInfo, err error) {
	nis = make([]cpuminer.NonceInfo, num)
	nCh := make(chan cpuminer.NonceInfo)
	qCh := make(chan struct{})
	miner := cpuminer.NewCPUMiner(qCh)

	defer close(qCh)
	go func() {
		defer close(nCh)
		miner.ComputeBlockNonce(
			cpuminer.MiningBlock{Data: pub.Serialize(), NonceChan: nCh, Stop: nil},
			cpuminer.Uint256{A: 0, B: 0, C: 0, D: 0},
			diff)
	}()

	for i := range nis {
		n := <-nCh
		nis[i] = n

		if err = kms.SetPublicKey(proto.NodeID(n.Hash.String()), n.Nonce, pub); err != nil {
			return
		}
	}

	// Register a local nonce, don't know what is the matter though
	kms.SetLocalNodeIDNonce(nis[0].Hash[:], &nis[0].Nonce)
	return
}

func createRandomBlock(parent hash.Hash, isGenesis bool) (b *ct.Block, err error) {
	// Generate key pair
	priv, pub, err := asymmetric.GenSecp256k1KeyPair()

	if err != nil {
		return
	}

	h := hash.Hash{}
	rand.Read(h[:])

	b = &ct.Block{
		SignedHeader: ct.SignedHeader{
			Header: ct.Header{
				Version:     0x01000000,
				Producer:    proto.NodeID(h.String()),
				GenesisHash: genesisHash,
				ParentHash:  parent,
				Timestamp:   time.Now().UTC(),
			},
			Signee:    pub,
			Signature: nil,
		},
	}

	for i, n := 0, rand.Intn(10)+10; i < n; i++ {
		h := &hash.Hash{}
		rand.Read(h[:])
		b.PushAckedQuery(h)
	}

	if isGenesis {
		// Register node for genesis verification
		var nis []cpuminer.NonceInfo
		nis, err = registerNodesWithPublicKey(pub, testDifficulty, 1)

		if err != nil {
			return
		}

		b.SignedHeader.GenesisHash = hash.Hash{}
		b.SignedHeader.Header.Producer = proto.NodeID(nis[0].Hash.String())
	}

	err = b.PackAndSignBlock(priv)
	return
}

func createRandomQueries(x int) (acks []*wt.SignedAckHeader, err error) {
	n := rand.Intn(x)
	acks = make([]*wt.SignedAckHeader, n)

	for i := range acks {
		if acks[i], err = createRandomNodesAndAck(); err != nil {
			return
		}
	}

	return
}

func createRandomBlockWithQueries(genesis, parent hash.Hash, acks []*wt.SignedAckHeader) (
	b *ct.Block, err error,
) {
	// Generate key pair
	priv, pub, err := asymmetric.GenSecp256k1KeyPair()

	if err != nil {
		return
	}

	h := hash.Hash{}
	rand.Read(h[:])

	b = &ct.Block{
		SignedHeader: ct.SignedHeader{
			Header: ct.Header{
				Version:     0x01000000,
				Producer:    proto.NodeID(h.String()),
				GenesisHash: genesis,
				ParentHash:  parent,
				Timestamp:   time.Now().UTC(),
			},
			Signee:    pub,
			Signature: nil,
		},
	}

	for _, ack := range acks {
		b.PushAckedQuery(&ack.HeaderHash)
	}

	err = b.PackAndSignBlock(priv)
	return
}

func createTestPeers(num int) (nis []cpuminer.NonceInfo, p *kayak.Peers, err error) {
	if num <= 0 {
		return
	}

	// Use a same key pair for all the servers, so that we can run multiple instances of sql-chain
	// locally without breaking the LocalKeyStore
	pub, err := kms.GetLocalPublicKey()

	if err != nil {
		return
	}

	priv, err := kms.GetLocalPrivateKey()

	if err != nil {
		return
	}

	nis, err = registerNodesWithPublicKey(pub, testDifficulty, num)

	if err != nil {
		return
	}

	s := make([]*kayak.Server, num)
	h := &hash.Hash{}

	for i := range s {
		rand.Read(h[:])
		s[i] = &kayak.Server{
			Role: func() conf.ServerRole {
				if i == 0 {
					return conf.Leader
				}
				return conf.Follower
			}(),
			ID:     proto.NodeID(nis[i].Hash.String()),
			PubKey: pub,
		}
	}

	p = &kayak.Peers{
		Term:      0,
		Leader:    s[0],
		Servers:   s,
		PubKey:    pub,
		Signature: nil,
	}

	if err = p.Sign(priv); err != nil {
		return
	}

	return
}

func setup() {
	// Setup RNG
	rand.Seed(time.Now().UnixNano())
	rand.Read(genesisHash[:])

	// Create temp dir
	var err error
	testDataDir, err = ioutil.TempDir("", "thunderdb")

	if err != nil {
		panic(err)
	}

	testPubKeysFile = path.Join(testDataDir, "pub")
	testPrivKeyFile = path.Join(testDataDir, "priv")

	// Setup public key store
	if err = kms.InitPublicKeyStore(testPubKeysFile, nil); err != nil {
		panic(err)
	}

	// Setup local key store
	kms.Unittest = true
	testPrivKey, testPubKey, err = asymmetric.GenSecp256k1KeyPair()

	if err != nil {
		panic(err)
	}

	kms.SetLocalKeyPair(testPrivKey, testPubKey)

	if err = kms.SavePrivateKey(testPrivKeyFile, testPrivKey, testMasterKey); err != nil {
		panic(err)
	}

	// Setup logging
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
}

func TestMain(m *testing.M) {
	setup()
	os.Exit(m.Run())
}
