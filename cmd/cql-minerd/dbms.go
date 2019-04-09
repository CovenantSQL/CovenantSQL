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
	"github.com/pkg/errors"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/rpc/mux"
	"github.com/CovenantSQL/CovenantSQL/worker"
)

var rootHash = hash.Hash{}

func startDBMS(server *mux.Server, direct *rpc.Server, onCreateDB func()) (dbms *worker.DBMS, err error) {
	if conf.GConf.Miner == nil {
		err = errors.New("invalid database config")
		return
	}

	cfg := &worker.DBMSConfig{
		RootDir:          conf.GConf.Miner.RootDir,
		Server:           server,
		DirectServer:     direct,
		MaxReqTimeGap:    conf.GConf.Miner.MaxReqTimeGap,
		OnCreateDatabase: onCreateDB,
	}

	if dbms, err = worker.NewDBMS(cfg); err != nil {
		err = errors.Wrap(err, "create new DBMS failed")
		return
	}

	if err = dbms.Init(); err != nil {
		err = errors.Wrap(err, "init DBMS failed")
		return
	}

	return
}
