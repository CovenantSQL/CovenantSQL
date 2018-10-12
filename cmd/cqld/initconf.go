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

package main

import (
	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/kayak"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

func initNodePeers(nodeID proto.NodeID, publicKeystorePath string) (nodes *[]proto.Node, peers *kayak.Peers, thisNode *proto.Node, err error) {
	privateKey, err := kms.GetLocalPrivateKey()
	if err != nil {
		log.Fatalf("get local private key failed: %s", err)
	}
	publicKey, err := kms.GetLocalPublicKey()
	if err != nil {
		log.Fatalf("get local public key failed: %s", err)
	}

	leader := &kayak.Server{
		Role:   proto.Leader,
		ID:     conf.GConf.BP.NodeID,
		PubKey: publicKey,
	}

	peers = &kayak.Peers{
		Term:    1,
		Leader:  leader,
		Servers: []*kayak.Server{},
		PubKey:  publicKey,
	}

	if conf.GConf.KnownNodes != nil {
		for i, n := range conf.GConf.KnownNodes {
			if n.Role == proto.Leader || n.Role == proto.Follower {
				//FIXME all KnownNodes
				conf.GConf.KnownNodes[i].PublicKey = kms.BP.PublicKey
				peers.Servers = append(peers.Servers, &kayak.Server{
					Role:   n.Role,
					ID:     n.ID,
					PubKey: publicKey,
				})
			}
		}
	}

	log.Debugf("AllNodes:\n %v\n", conf.GConf.KnownNodes)

	err = peers.Sign(privateKey)
	if err != nil {
		log.Errorf("sign peers failed: %s", err)
		return nil, nil, nil, err
	}
	log.Debugf("peers:\n %v\n", peers)

	//route.initResolver()
	kms.InitPublicKeyStore(publicKeystorePath, nil)

	// set p route and public keystore
	if conf.GConf.KnownNodes != nil {
		for _, p := range conf.GConf.KnownNodes {
			rawNodeIDHash, err := hash.NewHashFromStr(string(p.ID))
			if err != nil {
				log.Errorf("load hash from node id failed: %s", err)
				return nil, nil, nil, err
			}
			log.Debugf("set node addr: %v, %v", rawNodeIDHash, p.Addr)
			rawNodeID := &proto.RawNodeID{Hash: *rawNodeIDHash}
			route.SetNodeAddrCache(rawNodeID, p.Addr)
			node := &proto.Node{
				ID:        p.ID,
				Addr:      p.Addr,
				PublicKey: p.PublicKey,
				Nonce:     p.Nonce,
				Role:      p.Role,
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
	}

	return
}
