/*
 * Copyright 2019 The CovenantSQL Authors.
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

package mirror

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	rpc "github.com/CovenantSQL/CovenantSQL/rpc/mux"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/CovenantSQL/CovenantSQL/worker"
	x "github.com/CovenantSQL/CovenantSQL/xenomint"
	xs "github.com/CovenantSQL/CovenantSQL/xenomint/sqlite"
)

const (
	progressFileSuffix = ".progress"
	dbFileSuffix       = ".db3"
)

var (
	// ErrNotReadQuery represents invalid query type for mirror service to respond.
	ErrNotReadQuery = errors.New("only read query is supported")
)

// Service defines a database mirror service handler.
type Service struct {
	server   *rpc.Server
	dbID     proto.DatabaseID
	upstream proto.NodeID
	progress int32
	strg     *xs.SQLite3
	st       *x.State
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// NewService returns new mirror service handler.
func NewService(database string, server *rpc.Server) (s *Service, err error) {
	var (
		dbProgressPath = filepath.Join(conf.GConf.WorkingRoot, database+progressFileSuffix)
		dbPath         = filepath.Join(conf.GConf.WorkingRoot, database+dbFileSuffix)
		progress       int32
	)

	// load current progress
	if rawProgress, err := ioutil.ReadFile(dbProgressPath); err != nil {
		// not exists
		progress = 0
		log.WithError(err).Warning("read progress file failed")
	} else if rawIntProgress, err := strconv.ParseUint(string(rawProgress), 10, 32); err != nil {
		// progress not valid
		progress = 0
		log.WithError(err).Warning("parse progress file failed")
	} else {
		progress = int32(rawIntProgress)
	}

	s = &Service{
		server:   server,
		dbID:     proto.DatabaseID(database),
		progress: progress,
		stopCh:   make(chan struct{}),
	}

	if s.strg, err = xs.NewSqlite(dbPath); err != nil {
		err = errors.Wrap(err, "open database file failed")
		return
	}

	s.st = x.NewState(sql.LevelDefault, proto.NodeID(""), s.strg)

	// register myself
	if err = server.RegisterService(route.DBRPCName, s); err != nil {
		err = errors.Wrap(err, "register rpc failed")
		return
	}

	return
}

func (s *Service) start() (err error) {
	// query sqlchain profile for peers info
	var (
		req  = new(types.QuerySQLChainProfileReq)
		resp = new(types.QuerySQLChainProfileResp)
	)

	req.DBID = s.dbID

	if err = rpc.RequestBP(route.MCCQuerySQLChainProfile.String(), req, resp); err != nil {
		err = errors.Wrap(err, "get peers for database failed")
		return
	} else if len(resp.Profile.Miners) == 0 {
		err = errors.Wrap(err, "get empty peers for database")
		return
	}

	s.upstream = resp.Profile.Miners[0].NodeID

	// start subscriptions
	s.wg.Add(2)
	go s.run()
	go func() {
		defer s.wg.Done()
		s.server.Serve()
	}()

	return
}

func (s *Service) run() {
	defer s.wg.Done()

	var nextTick time.Duration

	for {
		select {
		case <-s.stopCh:
			return
		case <-time.After(nextTick):
			if err := s.pull(s.getProgress()); err != nil {
				nextTick = conf.GConf.SQLChainPeriod
			} else {
				nextTick /= 10
			}
		}
	}
}

func (s *Service) pull(count int32) (err error) {
	var (
		req  = new(worker.ObserverFetchBlockReq)
		resp = new(worker.ObserverFetchBlockResp)
		next int32
	)

	defer func() {
		lf := log.WithFields(log.Fields{
			"req_count": count,
			"count":     resp.Count,
		})

		if err != nil {
			lf.WithError(err).Debug("sync block failed")
		} else {
			if resp.Block != nil {
				lf = lf.WithField("block", resp.Block.BlockHash())
			} else {
				lf = lf.WithField("block", nil)
			}

			lf.WithField("next", next).Debug("sync block success")
		}
	}()

	req.DatabaseID = s.dbID
	req.Count = count

	if err = rpc.NewCaller().CallNode(s.upstream, route.DBSObserverFetchBlock.String(), req, resp); err != nil {
		return
	}

	if resp.Block == nil {
		err = errors.New("nil block, try later")
		return
	}

	if err = s.saveBlock(resp.Block); err != nil {
		err = errors.New("save block failed")
		return
	}

	if count < 0 {
		next = resp.Count + 1
	} else {
		next = count + 1
	}

	if atomic.CompareAndSwapInt32(&s.progress, count, next) {
		s.saveProgress()
	}

	return
}

func (s *Service) saveBlock(b *types.Block) (err error) {
	// save block
	return s.st.ReplayBlock(b)
}

func (s *Service) getProgress() int32 {
	return atomic.LoadInt32(&s.progress)
}

func (s *Service) saveProgress() {
	progressFile := filepath.Join(conf.GConf.WorkingRoot, string(s.dbID)+progressFileSuffix)
	_ = ioutil.WriteFile(progressFile, []byte(fmt.Sprintf("%d", s.getProgress())), 0644)
}

func (s *Service) stop() {
	if s.stopCh != nil {
		select {
		case <-s.stopCh:
		default:
			close(s.stopCh)
		}
	}
	s.server.Stop()
	s.wg.Wait()
}

// Query mocks DBS.Query for mirrored database.
func (s *Service) Query(req *types.Request, res *types.Response) (err error) {
	if req.Header.QueryType != types.ReadQuery {
		// only read query is supported
		err = ErrNotReadQuery
		return
	}

	if req.Header.DatabaseID != s.dbID {
		// database instance not matched
		err = worker.ErrNotExists
		return
	}

	var r *types.Response
	if _, r, err = s.st.Query(req, false); err != nil {
		return
	}
	*res = *r
	return
}

// Ack mocks DBS.Ack for mirrored database.
func (s *Service) Ack(ack *types.Ack, _ *types.AckResponse) (err error) {
	// no-op
	return
}
