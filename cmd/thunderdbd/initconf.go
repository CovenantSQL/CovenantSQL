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
	"gitlab.com/thunderdb/ThunderDB/conf"
	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/kayak"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/route"
)

var (
// TODO(auxten): move these BP info into conf

// AllNodes holds all BP nodes and client node info
//AllNodes = []NodeInfo{}
)

func initNodePeers(nodeID proto.NodeID, publicKeystorePath string) (nodes *[]conf.NodeInfo, peers *kayak.Peers, thisNode *conf.NodeInfo, err error) {
	privateKey, err := kms.GetLocalPrivateKey()
	if err != nil {
		log.Fatalf("get local private key failed: %s", err)
	}
	publicKey, err := kms.GetLocalPublicKey()
	if err != nil {
		log.Fatalf("get local public key failed: %s", err)
	}

	leader := &kayak.Server{
		Role:   conf.Leader,
		ID:     conf.GConf.BP.NodeID,
		PubKey: publicKey,
	}

	peers = &kayak.Peers{
		Term:    1,
		Leader:  leader,
		Servers: []*kayak.Server{},
		PubKey:  publicKey,
	}

	for i, n := range (*conf.GConf.KnownNodes)[:] {
		if n.Role == conf.Leader || n.Role == conf.Follower {
			//FIXME all KnownNodes
			(*conf.GConf.KnownNodes)[i].PublicKey = kms.BP.PublicKey
			peers.Servers = append(peers.Servers, &kayak.Server{
				Role:   n.Role,
				ID:     n.ID,
				PubKey: publicKey,
			})
		}
		if n.Role == conf.Client {
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
			//FIXME
			(*conf.GConf.KnownNodes)[i].PublicKey = clientPublicKey
		}
	}
	log.Debugf("AllNodes:\n %v\n", conf.GConf.KnownNodes)

	err = peers.Sign(privateKey)
	if err != nil {
		log.Errorf("sign peers failed: %s", err)
		return nil, nil, nil, err
	}
	log.Debugf("peers:\n %v\n", peers)

	route.InitResolver()
	kms.InitPublicKeyStore(publicKeystorePath, nil)
	// set p route and public keystore
	for _, p := range (*conf.GConf.KnownNodes)[:] {
		rawNodeIDHash, err := hash.NewHashFromStr(string(p.ID))
		if err != nil {
			log.Errorf("load hash from node id failed: %s", err)
			return nil, nil, nil, err
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
		if p.ID == nodeID {
			kms.SetLocalNodeIDNonce(rawNodeID.CloneBytes(), &p.Nonce)
			thisNode = &p
		}
	}

	return
}
