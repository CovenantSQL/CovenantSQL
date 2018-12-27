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
	"bytes"
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	ct "github.com/CovenantSQL/CovenantSQL/sqlchain/otypes"
	"github.com/CovenantSQL/CovenantSQL/storage"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	wt "github.com/CovenantSQL/CovenantSQL/worker/otypes"
)

var (
	dhtFile    string
	privateKey string
)

func init() {
	flag.StringVar(&dhtFile, "dhtFile", "dht.db", "dht database file to fix")
	flag.StringVar(&privateKey, "private", "private.key", "private key to use for signing")
}

// OldBlock type mocks current sqlchain block type for custom serialization.
type OldBlock ct.Block

// MarshalBinary implements custom binary marshaller for OldBlock.
func (b *OldBlock) MarshalBinary() ([]byte, error) {
	return nil, nil
}

// UnmarshalBinary implements custom binary unmarshaller for OldBlock.
func (b *OldBlock) UnmarshalBinary(data []byte) (err error) {
	reader := bytes.NewReader(data)
	var headerBuf []byte

	if err = ReadElements(reader, binary.BigEndian, &headerBuf); err != nil {
		return
	}

	if err = ReadElements(bytes.NewReader(headerBuf), binary.BigEndian,
		&b.SignedHeader.Version,
		&b.SignedHeader.Producer,
		&b.SignedHeader.GenesisHash,
		&b.SignedHeader.ParentHash,
		&b.SignedHeader.MerkleRoot,
		&b.SignedHeader.Timestamp,
		&b.SignedHeader.BlockHash,
		&b.SignedHeader.Signee,
		&b.SignedHeader.Signature); err != nil {
		return
	}

	err = ReadElements(reader, binary.BigEndian, &b.Queries)
	return
}

// OldServer ports back the original kayak server structure.
type OldServer struct {
	Role   proto.ServerRole
	ID     proto.NodeID
	PubKey *asymmetric.PublicKey
}

// OldPeers ports back the original kayak peers structure.
type OldPeers struct {
	Term      uint64
	Leader    *OldServer
	Servers   []*OldServer
	PubKey    *asymmetric.PublicKey
	Signature *asymmetric.Signature
}

// ServiceInstanceV0 defines the version 0 service instance type before marshaller updates.
type ServiceInstanceV0 struct {
	DatabaseID   proto.DatabaseID
	Peers        *OldPeers
	ResourceMeta wt.ResourceMeta
	GenesisBlock *OldBlock
}

// ServiceInstanceV1 defines the version 1 service instance type before marshaller updates.
type ServiceInstanceV1 struct {
	DatabaseID   proto.DatabaseID
	Peers        *OldPeers
	ResourceMeta wt.ResourceMeta
	GenesisBlock *ct.Block
}

// ServiceInstanceV2 defines the version 2 service instance type before marshaller updates.
type ServiceInstanceV2 wt.ServiceInstance

func convertPeers(oldPeers *OldPeers) (newPeers *proto.Peers) {
	if oldPeers == nil {
		return
	}

	newPeers = new(proto.Peers)
	for _, s := range oldPeers.Servers {
		newPeers.Servers = append(newPeers.Servers, s.ID)
	}
	newPeers.Leader = oldPeers.Leader.ID
	newPeers.Term = oldPeers.Term

	return
}

func convertResourceMeta(oldMeta wt.ResourceMeta) (newMeta types.ResourceMeta) {
	newMeta.Node = oldMeta.Node
	newMeta.Space = oldMeta.Space
	newMeta.Memory = oldMeta.Memory
	newMeta.LoadAvgPerCPU = oldMeta.LoadAvgPerCPU
	newMeta.EncryptionKey = oldMeta.EncryptionKey
	return
}

func convertBlockHeader(oldHeader ct.Header) (newHeader types.Header) {
	newHeader.Version = oldHeader.Version
	newHeader.Producer = oldHeader.Producer
	newHeader.GenesisHash = oldHeader.GenesisHash
	newHeader.ParentHash = oldHeader.ParentHash
	newHeader.MerkleRoot = oldHeader.MerkleRoot
	newHeader.Timestamp = oldHeader.Timestamp
	return
}

func main() {
	flag.Parse()

	log.Info("start hotfix")

	// load private key
	privateKey, err := kms.LoadPrivateKey(privateKey, []byte(""))
	if err != nil {
		log.WithError(err).Fatal("load private key failed")
		return
	}

	// backup dht file
	if err := exec.Command("cp", "-av", dhtFile, dhtFile+".bak").Run(); err != nil {
		log.WithError(err).Fatal("backup database failed")
		return
	}

	st, err := storage.New(dhtFile)
	if err != nil {
		log.WithError(err).Fatal("open database failed")
		return
	}
	defer st.Close()

	_, _, rows, err := st.Query(context.Background(),
		[]storage.Query{{Pattern: "SELECT `id`, `meta` FROM `databases`"}})

	if err != nil {
		log.WithError(err).Fatal("select database failed")
		return
	}

	for _, row := range rows {
		if len(row) <= 0 {
			continue
		}

		id := string(row[0].([]byte))
		rawInstance := row[1].([]byte)

		// test decode
		var testDecode interface{}

		// copy instance to new type
		var newInstance types.ServiceInstance

		if err := utils.DecodeMsgPackPlain(rawInstance, &testDecode); err != nil {
			log.WithError(err).Fatal("test decode failed")
		} else {
			// detect if the genesis block is in old version
			if strings.Contains(fmt.Sprintf("%#v", testDecode), "\"GenesisBlock\":[]uint8") {
				log.Info("detected version 0 (without msgpack tag and use custom serializer)")
				var instance ServiceInstanceV0

				if err := utils.DecodeMsgPackPlain(rawInstance, &instance); err != nil {
					log.WithError(err).Fatal("decode msgpack failed")
					return
				}

				newInstance.DatabaseID = instance.DatabaseID
				newInstance.Peers = convertPeers(instance.Peers)
				newInstance.ResourceMeta = convertResourceMeta(instance.ResourceMeta)
				newInstance.GenesisBlock = &types.Block{}
				newInstance.GenesisBlock.SignedHeader.Header = convertBlockHeader(instance.GenesisBlock.SignedHeader.Header)
			} else if strings.Contains(fmt.Sprintf("%#v", testDecode), "\"PubKey\"") {
				log.Info("detected version 1 (old kayak implementation [called as kaar])")
				var instance ServiceInstanceV1

				if err := utils.DecodeMsgPack(rawInstance, &instance); err != nil {
					log.WithError(err).Fatal("decode msgpack failed")
					return
				}

				newInstance.DatabaseID = instance.DatabaseID
				newInstance.Peers = convertPeers(instance.Peers)
				newInstance.ResourceMeta = convertResourceMeta(instance.ResourceMeta)
				newInstance.GenesisBlock = &types.Block{}
				newInstance.GenesisBlock.SignedHeader.Header = convertBlockHeader(instance.GenesisBlock.SignedHeader.Header)
			} else if !strings.Contains(fmt.Sprintf("%#v", testDecode), "\"HSV\"") {
				log.Info("detected version 2 (old block format without HSV)")
				var instance ServiceInstanceV2

				if err := utils.DecodeMsgPack(rawInstance, &instance); err != nil {
					log.WithError(err).Fatal("decode msgpack failed")
					return
				}

				newInstance.DatabaseID = instance.DatabaseID
				newInstance.Peers = instance.Peers
				newInstance.ResourceMeta = convertResourceMeta(instance.ResourceMeta)
				newInstance.GenesisBlock = &types.Block{}
				newInstance.GenesisBlock.SignedHeader.Header = convertBlockHeader(instance.GenesisBlock.SignedHeader.Header)
			} else {
				log.Info("detected latest version, need re-signature")

				if err := utils.DecodeMsgPack(rawInstance, &newInstance); err != nil {
					log.WithError(err).Fatal("decode msgpack failed")
					return
				}
			}

			// set genesis block to now
			newInstance.GenesisBlock.SignedHeader.Timestamp = time.Now().UTC()

			// sign peers again
			if err := newInstance.Peers.Sign(privateKey); err != nil {
				log.WithError(err).Fatal("sign peers failed")
				return
			}

			if err := newInstance.GenesisBlock.PackAndSignBlock(privateKey); err != nil {
				log.WithError(err).Fatal("sign genesis block failed")
				return
			}
		}

		d, _ := json.Marshal(newInstance)
		log.Infof("database is: %#v -> %s", id, string(d))

		// encode and put back to database
		rawInstanceBuffer, err := utils.EncodeMsgPack(newInstance)
		if err != nil {
			log.WithError(err).Fatal("encode msgpack failed")
			return
		}

		rawInstance = rawInstanceBuffer.Bytes()

		if _, err := st.Exec(context.Background(), []storage.Query{
			{
				Pattern: "UPDATE `databases` SET `meta` = ? WHERE `id` = ?",
				Args: []sql.NamedArg{
					{
						Value: rawInstance,
					},
					{
						Value: id,
					},
				},
			},
		}); err != nil {
			log.WithError(err).Fatal("update meta failed")
			return
		}
	}

	log.Info("hotfix complete")
}
