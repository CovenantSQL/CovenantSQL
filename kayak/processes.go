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

package kayak

import (
	"context"

	kt "github.com/CovenantSQL/CovenantSQL/kayak/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/CovenantSQL/CovenantSQL/utils/timer"
	"github.com/CovenantSQL/CovenantSQL/utils/trace"
	"github.com/pkg/errors"
)

func (r *Runtime) doLeaderPrepare(ctx context.Context, tm *timer.Timer, req interface{}) (prepareLog *kt.Log, err error) {
	defer trace.StartRegion(ctx, "doLeaderPrepare").End()

	// check prepare in leader
	if err = r.doCheck(ctx, req); err != nil {
		err = errors.Wrap(err, "leader verify log")
		return
	}

	tm.Add("leader_check")

	// encode request
	var encBuf []byte
	if encBuf, err = r.doEncodePayload(ctx, req); err != nil {
		err = errors.Wrap(err, "encode kayak payload failed")
		return
	}

	tm.Add("leader_encode_payload")

	// create prepare request
	if prepareLog, err = r.leaderLogPrepare(ctx, tm, encBuf); err != nil {
		// serve error, leader could not write logs, change leader in block producer
		// TODO(): CHANGE LEADER
		return
	}

	// Leader pending map handling.
	r.markPendingPrepare(ctx, prepareLog.Index)

	tm.Add("leader_prepare")

	// send prepare to all nodes
	prepareTracker := r.applyRPC(prepareLog, r.minPreparedFollowers)
	prepareCtx, prepareCtxCancelFunc := context.WithTimeout(ctx, r.prepareTimeout)
	defer prepareCtxCancelFunc()
	prepareErrors, prepareDone, _ := prepareTracker.get(prepareCtx)
	if !prepareDone {
		// timeout, rollback
		err = kt.ErrPrepareTimeout
		return
	}

	tm.Add("follower_prepare")

	// collect errors
	err = r.errorSummary(prepareErrors)

	return
}

func (r *Runtime) doLeaderCommit(ctx context.Context, tm *timer.Timer, prepareLog *kt.Log, req interface{}) (
	result interface{}, logIndex uint64, err error) {
	defer trace.StartRegion(ctx, "doLeaderCommit").End()
	var commitResult *commitResult
	if commitResult, err = r.leaderCommitResult(ctx, tm, req, prepareLog).Get(ctx); err != nil {
		return
	}

	result = commitResult.result
	logIndex = commitResult.index
	err = commitResult.err

	if commitResult.rpc != nil {
		commitResult.rpc.get(ctx)
	}

	tm.Add("wait_follower_commit")

	return
}

func (r *Runtime) doLeaderRollback(ctx context.Context, tm *timer.Timer, prepareLog *kt.Log) {
	defer trace.StartRegion(ctx, "doLeaderRollback").End()
	// rollback local
	var rollbackLog *kt.Log
	var logErr error
	if rollbackLog, logErr = r.leaderLogRollback(ctx, tm, prepareLog.Index); logErr != nil {
		// serve error, construct rollback log failed, internal error
		// TODO(): CHANGE LEADER
		return
	}

	defer trace.StartRegion(ctx, "followerRollback").End()

	// async send rollback to all nodes
	r.applyRPC(rollbackLog, 0)

	tm.Add("follower_rollback")
}

func (r *Runtime) leaderLogPrepare(ctx context.Context, tm *timer.Timer, data []byte) (*kt.Log, error) {
	defer trace.StartRegion(ctx, "leaderLogPrepare").End()
	defer tm.Add("leader_log_prepare")
	// just write new log
	return r.newLog(ctx, kt.LogPrepare, data)
}

func (r *Runtime) leaderLogRollback(ctx context.Context, tm *timer.Timer, i uint64) (*kt.Log, error) {
	defer trace.StartRegion(ctx, "leaderLogRollback").End()
	defer tm.Add("leader_log_rollback")
	// just write new log
	return r.newLog(ctx, kt.LogRollback, r.uint64ToBytes(i))
}

func (r *Runtime) followerPrepare(ctx context.Context, tm *timer.Timer, l *kt.Log) (err error) {
	defer func() {
		log.WithField("r", l.Index).WithFields(tm.ToLogFields()).Debug("kayak follower prepare stat")
	}()

	// decode
	var req interface{}
	if req, err = r.doDecodePayload(ctx, l.Data); err != nil {
		return
	}
	tm.Add("decode")

	if err = r.doCheck(ctx, req); err != nil {
		return
	}
	tm.Add("check")

	// write log
	if err = r.writeWAL(ctx, l); err != nil {
		return

	}
	tm.Add("write_wal")

	r.markPendingPrepare(ctx, l.Index)
	tm.Add("mark")

	return
}

func (r *Runtime) followerRollback(ctx context.Context, tm *timer.Timer, l *kt.Log) (err error) {
	var prepareLog *kt.Log
	if _, prepareLog, err = r.getPrepareLog(ctx, l); err != nil || prepareLog == nil {
		err = errors.Wrap(err, "get original request in rollback failed")
		return
	}
	tm.Add("get_prepare")

	// check if prepare already processed
	if r.checkIfPrepareFinished(ctx, prepareLog.Index) {
		err = errors.Wrap(kt.ErrInvalidLog, "prepare request already processed")
		return
	}
	tm.Add("check_prepare")

	// write wal
	if err = r.writeWAL(ctx, l); err != nil {
		return
	}
	tm.Add("write_wal")

	r.markPrepareFinished(ctx, l.Index)
	tm.Add("mark")

	return
}

func (r *Runtime) followerCommit(ctx context.Context, tm *timer.Timer, l *kt.Log) (err error) {
	var (
		prepareLog *kt.Log
		lastCommit uint64
		cResult    *commitResult
	)

	defer func() {
		log.WithField("r", l.Index).WithFields(tm.ToLogFields()).Debug("kayak follower commit stat")
	}()

	if lastCommit, prepareLog, err = r.getPrepareLog(ctx, l); err != nil {
		err = errors.Wrap(err, "get original request in commit failed")
		return
	}
	tm.Add("get_prepare")

	// check if prepare already processed
	if r.checkIfPrepareFinished(ctx, prepareLog.Index) {
		err = errors.Wrap(kt.ErrInvalidLog, "prepare request already processed")
		return
	}
	tm.Add("check_prepare")

	cResult, err = r.followerCommitResult(ctx, tm, l, prepareLog, lastCommit).Get(ctx)
	if cResult != nil {
		err = cResult.err
	}

	r.markPrepareFinished(ctx, l.Index)
	tm.Add("mark")

	return
}
