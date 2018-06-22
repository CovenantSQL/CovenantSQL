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
	log "github.com/sirupsen/logrus"
	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/kayak"
	"gitlab.com/thunderdb/ThunderDB/pow/cpuminer"
	"gitlab.com/thunderdb/ThunderDB/proto"
)

var (
	BP1 = proto.Node{
		ID:        proto.NodeID("000000000013fd4b3180dd424d5a895bc57b798e5315087b7198c926d8893f98"),
		Addr:      "127.0.0.1:2121",
		PublicKey: nil,
		Nonce:     cpuminer.Uint256{789554103, 0, 0, 8070450536379825883},
	}
	BP2 = proto.Node{
		ID:        proto.NodeID("0000000000293f7216362791b6b1c9772184d6976cb34310c42547735410186c"),
		Addr:      "127.0.0.1:2120",
		PublicKey: nil,
		Nonce:     cpuminer.Uint256{746598970, 0, 0, 10808639108098016056},
	}
)

func InitKayakPeers(privateKey *asymmetric.PrivateKey, publicKey *asymmetric.PublicKey) (peers *kayak.Peers, err error) {
	leader := &kayak.Server{
		Role:   kayak.Leader,
		ID:     kms.BPNodeID,
		PubKey: publicKey,
	}

	peers = &kayak.Peers{
		Term:   1,
		Leader: leader,
		Servers: []*kayak.Server{
			leader,
			{
				Role:   kayak.Follower,
				ID:     BP1.ID,
				PubKey: publicKey,
			},
			{
				Role:   kayak.Follower,
				ID:     BP2.ID,
				PubKey: publicKey,
			},
		},
		PubKey: publicKey,
	}
	err = peers.Sign(privateKey)
	if err != nil {
		log.Errorf("sign peers failed: %s", err)
		return nil, err
	}

	return
}
