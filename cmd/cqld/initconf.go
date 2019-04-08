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
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

func initNodePeers(nodeID proto.NodeID, publicKeystorePath string) (nodes *[]proto.Node, peers *proto.Peers, thisNode *proto.Node, err error) {
	privateKey, err := kms.GetLocalPrivateKey()
	if err != nil {
		log.WithError(err).Fatal("get local private key failed")
	}

	peers = &proto.Peers{
		PeersHeader: proto.PeersHeader{
			Term:   1,
			Leader: conf.GConf.BP.NodeID,
		},
	}

	if conf.GConf.KnownNodes != nil {
		for i, n := range conf.GConf.KnownNodes {
			if n.Role == proto.Leader || n.Role == proto.Follower {
				//FIXME all KnownNodes
				conf.GConf.KnownNodes[i].PublicKey = kms.BP.PublicKey
				peers.Servers = append(peers.Servers, n.ID)
			}
		}
	}

	log.Debugf("AllNodes:\n %#v\n", conf.GConf.KnownNodes)

	err = peers.Sign(privateKey)
	if err != nil {
		log.WithError(err).Error("sign peers failed")
		return nil, nil, nil, err
	}
	log.Debugf("peers:\n %#v\n", peers)

	//route.initResolver()
	kms.InitPublicKeyStore(publicKeystorePath, nil)

	// set p route and public keystore
	if conf.GConf.KnownNodes != nil {
		for i, p := range conf.GConf.KnownNodes {
			rawNodeIDHash, err := hash.NewHashFromStr(string(p.ID))
			if err != nil {
				log.WithError(err).Error("load hash from node id failed")
				return nil, nil, nil, err
			}
			log.WithFields(log.Fields{
				"node": rawNodeIDHash.String(),
				"addr": p.Addr,
			}).Debug("set node addr")
			rawNodeID := &proto.RawNodeID{Hash: *rawNodeIDHash}
			route.SetNodeAddrCache(rawNodeID, p.Addr)
			node := &proto.Node{
				ID:         p.ID,
				Addr:       p.Addr,
				DirectAddr: p.DirectAddr,
				PublicKey:  p.PublicKey,
				Nonce:      p.Nonce,
				Role:       p.Role,
			}
			err = kms.SetNode(node)
			if err != nil {
				log.WithField("node", node).WithError(err).Error("set node failed")
			}
			if p.ID == nodeID {
				kms.SetLocalNodeIDNonce(rawNodeID.CloneBytes(), &p.Nonce)
				thisNode = &conf.GConf.KnownNodes[i]
			}
		}
	}

	return
}
