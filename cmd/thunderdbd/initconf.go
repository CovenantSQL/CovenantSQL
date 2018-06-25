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

package main

import (
	"encoding/hex"

	log "github.com/sirupsen/logrus"
	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/kayak"
	"gitlab.com/thunderdb/ThunderDB/pow/cpuminer"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/route"
)

var (
	// TODO(auxten): move these BP info into conf
	AllNodes = []NodeInfo{
		{
			ID:        kms.BPNodeID,
			Nonce:     kms.BPNonce,
			PublicKey: nil,
			Addr:      "127.0.0.1:2122",
			Role:      kayak.Leader,
		},
		{
			ID:        proto.NodeID("000000000013fd4b3180dd424d5a895bc57b798e5315087b7198c926d8893f98"),
			PublicKey: nil,
			Nonce:     cpuminer.Uint256{789554103, 0, 0, 8070450536379825883},
			Addr:      "127.0.0.1:2121",
			Role:      kayak.Follower,
		},
		{
			ID:        proto.NodeID("0000000000293f7216362791b6b1c9772184d6976cb34310c42547735410186c"),
			PublicKey: nil,
			Nonce:     cpuminer.Uint256{746598970, 0, 0, 10808639108098016056},
			Addr:      "127.0.0.1:2120",
			Role:      kayak.Follower,
		},
		{
			//{{22403 0 0 0} 20 00000f3b43288fe99831eb533ab77ec455d13e11fc38ec35a42d4edd17aa320d}
			ID:        proto.NodeID("00000f3b43288fe99831eb533ab77ec455d13e11fc38ec35a42d4edd17aa320d"),
			Nonce:     cpuminer.Uint256{22403, 0, 0, 0},
			PublicKey: nil,
			Addr:      "",
			Role:      kayak.Client,
		},
	}
)

// NodeInfo for conf generation and load purpose.
type NodeInfo struct {
	ID        proto.NodeID
	Nonce     cpuminer.Uint256
	PublicKey *asymmetric.PublicKey
	Addr      string
	Role      kayak.ServerRole
}

func initNodePeers(idx int, publicKeystorePath string) (nodes *[]NodeInfo, peers *kayak.Peers, err error) {
	privateKey, err := kms.GetLocalPrivateKey()
	if err != nil {
		log.Fatalf("get local private key failed: %s", err)
	}
	publicKey, err := kms.GetLocalPublicKey()
	if err != nil {
		log.Fatalf("get local public key failed: %s", err)
	}

	leader := &kayak.Server{
		Role:   kayak.Leader,
		ID:     AllNodes[0].ID,
		PubKey: publicKey,
	}

	peers = &kayak.Peers{
		Term:    1,
		Leader:  leader,
		Servers: []*kayak.Server{},
		PubKey:  publicKey,
	}

	for i, n := range AllNodes[:] {
		if n.Role == kayak.Leader || n.Role == kayak.Follower {
			AllNodes[i].PublicKey = kms.BPPublicKey
			peers.Servers = append(peers.Servers, &kayak.Server{
				Role:   n.Role,
				ID:     n.ID,
				PubKey: publicKey,
			})
		}
		if n.Role == kayak.Client {
			var publicKeyBytes []byte
			var clientPublicKey *asymmetric.PublicKey
			//02ec784ca599f21ef93fe7abdc68d78817ab6c9b31f2324d15ea174d9da498b4c4
			publicKeyBytes, err = hex.DecodeString("02ec784ca599f21ef93fe7abdc68d78817ab6c9b31f2324d15ea174d9da498b4c4")
			if err != nil {
				log.Errorf("hex decode clientPublicKey error: %s", err)
				return
			}
			clientPublicKey, err = asymmetric.ParsePubKey(publicKeyBytes)
			if err != nil {
				log.Errorf("parse clientPublicKey error: %s", err)
				return
			}
			AllNodes[i].PublicKey = clientPublicKey
		}
	}
	nodes = &AllNodes
	log.Debugf("AllNodes:\n %v\n", nodes)

	err = peers.Sign(privateKey)
	if err != nil {
		log.Errorf("sign peers failed: %s", err)
		return nil, nil, err
	}
	log.Debugf("peers:\n %v\n", peers)

	route.InitResolver()
	kms.InitPublicKeyStore(publicKeystorePath, nil)
	// set p route and public keystore
	for i, p := range AllNodes[:] {
		rawNodeIDHash, err := hash.NewHashFromStr(string(p.ID))
		if err != nil {
			log.Errorf("load hash from node id failed: %s", err)
			return nil, nil, err
		}
		log.Debugf("set node addr: %v, %v", rawNodeIDHash, p.Addr)
		rawNodeID := &proto.RawNodeID{Hash: *rawNodeIDHash}
		route.SetNodeAddr(rawNodeID, p.Addr)
		node := &proto.Node{
			ID:        p.ID,
			Addr:      p.Addr,
			PublicKey: p.PublicKey,
			Nonce:     p.Nonce,
		}
		err = kms.SetNode(node)
		if err != nil {
			log.Errorf("set node failed: %v\n %s", node, err)
		}
		if i == idx {
			kms.SetLocalNodeIDNonce(rawNodeID.CloneBytes(), &p.Nonce)
		}
	}

	return
}
