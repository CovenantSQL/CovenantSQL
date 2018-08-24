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
	"os/exec"

	"github.com/CovenantSQL/CovenantSQL/kayak"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/sqlchain/storage"
	ct "github.com/CovenantSQL/CovenantSQL/sqlchain/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	wt "github.com/CovenantSQL/CovenantSQL/worker/types"
)

var (
	dhtFile string
)

func init() {
	flag.StringVar(&dhtFile, "dhtFile", "dht.db", "dht database file to fix")
}

type OldBlock ct.Block

func (b *OldBlock) MarshalBinary() ([]byte, error) {
	buffer := bytes.NewBuffer(nil)

	if err := WriteElements(buffer, binary.BigEndian,
		b.SignedHeader.Version,
		b.SignedHeader.Producer,
		&b.SignedHeader.GenesisHash,
		&b.SignedHeader.ParentHash,
		&b.SignedHeader.MerkleRoot,
		b.SignedHeader.Timestamp,
		&b.SignedHeader.BlockHash,
		b.SignedHeader.Signee,
		b.SignedHeader.Signature,
		b.Queries,
	); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil

}

func (b *OldBlock) UnmarshalBinary(data []byte) error {
	reader := bytes.NewReader(data)
	return ReadElements(reader, binary.BigEndian,
		&b.SignedHeader.Version,
		&b.SignedHeader.Producer,
		&b.SignedHeader.GenesisHash,
		&b.SignedHeader.ParentHash,
		&b.SignedHeader.MerkleRoot,
		&b.SignedHeader.Timestamp,
		&b.SignedHeader.BlockHash,
		&b.SignedHeader.Signee,
		&b.SignedHeader.Signature,
		&b.Queries)
}

type ServiceInstance struct {
	DatabaseID   proto.DatabaseID
	Peers        *kayak.Peers
	ResourceMeta wt.ResourceMeta
	GenesisBlock *OldBlock
}

func main() {
	flag.Parse()

	log.Infof("start hotfix")

	// backup dht file
	if err := exec.Command("cp", "-av", dhtFile, dhtFile+".bak").Run(); err != nil {
		log.Fatalf("backup database failed: %v", err)
		return
	}

	st, err := storage.New(dhtFile)
	if err != nil {
		log.Fatalf("open database failed: %v", err)
		return
	}
	defer st.Close()

	_, _, rows, err := st.Query(context.Background(),
		[]storage.Query{{Pattern: "SELECT `id`, `meta` FROM `databases`"}})

	if err != nil {
		log.Fatalf("select databases failed: %v", err)
		return
	}

	log.Infof("rows: %v", rows)

	for _, row := range rows {
		if len(row) <= 0 {
			continue
		}

		var instance ServiceInstance

		id := string(row[0].([]byte))
		rawInstance := row[1].([]byte)

		if err := utils.DecodeMsgPackPlain(rawInstance, &instance); err != nil {
			log.Fatalf("decode msgpack failed: %v", err)
			return
		}

		log.Infof("database is: %v -> %v", id, instance)

		// copy instance to new type
		var newInstance wt.ServiceInstance

		newInstance.DatabaseID = instance.DatabaseID
		newInstance.Peers = instance.Peers
		newInstance.ResourceMeta = instance.ResourceMeta
		newInstance.GenesisBlock = &ct.Block{
			SignedHeader: instance.GenesisBlock.SignedHeader,
			Queries:      instance.GenesisBlock.Queries,
		}

		// encode and put back to database
		rawInstanceBuffer, err := utils.EncodeMsgPack(newInstance)
		if err != nil {
			log.Fatalf("encode msgpack failed: %v", err)
			return
		}

		rawInstance = rawInstanceBuffer.Bytes()

		if _, err := st.Exec(context.Background(), []storage.Query{
			{
				Pattern: "UPDATE `database` SET `meta` = ? WHERE `id` = ?",
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
			log.Fatalf("update meta failed: %v", err)
			return
		}
	}

	log.Infof("hotfix complete")
}
