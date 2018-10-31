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
	"flag"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/kayak"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/sqlchain/storage"
	ct "github.com/CovenantSQL/CovenantSQL/sqlchain/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	wt "github.com/CovenantSQL/CovenantSQL/worker/types"
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

// ServiceInstance defines the old service instance type before marshaller updates.
type ServiceInstance struct {
	DatabaseID   proto.DatabaseID
	Peers        *kayak.Peers
	ResourceMeta wt.ResourceMeta
	GenesisBlock *OldBlock
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
		var newInstance wt.ServiceInstance

		if err := utils.DecodeMsgPackPlain(rawInstance, &testDecode); err != nil {
			log.WithError(err).Fatal("test decode failed")
		} else {
			// detect if the genesis block is in old version
			if strings.Contains(fmt.Sprintf("%#v", testDecode), "\"GenesisBlock\":[]uint8") {
				log.Info("detected old version")
				var instance ServiceInstance

				if err := utils.DecodeMsgPackPlain(rawInstance, &instance); err != nil {
					log.WithError(err).Fatal("decode msgpack failed")
					return
				}

				newInstance.DatabaseID = instance.DatabaseID
				newInstance.Peers = instance.Peers
				newInstance.ResourceMeta = instance.ResourceMeta
				newInstance.GenesisBlock = &ct.Block{
					SignedHeader: instance.GenesisBlock.SignedHeader,
					Queries:      instance.GenesisBlock.Queries,
				}
			} else {
				log.Info("detected new version, need re-signature")

				if err := utils.DecodeMsgPack(rawInstance, &newInstance); err != nil {
					log.WithError(err).Fatal("decode msgpack failed")
					return
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
		}

		log.Infof("database is: %#v -> %#v", id, newInstance)

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
