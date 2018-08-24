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
	"context"
	"database/sql"
	"flag"
	"os/exec"

	"gitlab.com/thunderdb/ThunderDB/sqlchain/storage"
	"gitlab.com/thunderdb/ThunderDB/utils"
	"gitlab.com/thunderdb/ThunderDB/utils/log"
	wt "gitlab.com/thunderdb/ThunderDB/worker/types"
)

var (
	dhtFile string
)

func init() {
	flag.StringVar(&dhtFile, "dhtFile", "dht.db", "dht database file to fix")
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

		var instance wt.ServiceInstance

		id := string(row[0].([]byte))
		rawInstance := row[1].([]byte)

		if err := utils.DecodeMsgPackPlain(rawInstance, &instance); err != nil {
			log.Fatalf("decode msgpack failed: %v", err)
			return
		}

		log.Infof("database is: %v -> %v", id, instance)

		// encode and put back to database
		rawInstanceBuffer, err := utils.EncodeMsgPack(rawInstance)
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
