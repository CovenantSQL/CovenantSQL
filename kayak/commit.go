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
	"sync/atomic"

	"github.com/pkg/errors"

	kt "github.com/CovenantSQL/CovenantSQL/kayak/types"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/CovenantSQL/CovenantSQL/utils/timer"
	"github.com/CovenantSQL/CovenantSQL/utils/trace"
)

func (r *Runtime) leaderCommitResult(ctx context.Context, tm *timer.Timer, reqPayload interface{}, prepareLog *kt.Log) (res *commitFuture) {
	defer trace.StartRegion(ctx, "leaderCommitResult").End()

	// decode log and send to commit channel to process
	res = newCommitFuture()

	if prepareLog == nil {
		res.Set(&commitResult{err: errors.Wrap(kt.ErrInvalidLog, "nil prepare log in commit")})
		return
	}

	// decode prepare log
	req := &commitReq{
		ctx:    ctx,
		data:   reqPayload,
		index:  prepareLog.Index,
		result: res,
		tm:     tm,
	}

	select {
	case <-ctx.Done():
		res = nil
	case r.commitCh <- req:
	}

	return
}

func (r *Runtime) followerCommitResult(ctx context.Context, tm *timer.Timer, commitLog *kt.Log, prepareLog *kt.Log, lastCommit uint64) (res *commitFuture) {
	defer trace.StartRegion(ctx, "followerCommitResult").End()

	// decode log and send to commit channel to process
	res = newCommitFuture()

	if prepareLog == nil {
		res.Set(&commitResult{err: errors.Wrap(kt.ErrInvalidLog, "nil prepare log in commit")})
		return
	}

	myLastCommit := atomic.LoadUint64(&r.lastCommit)

	// check committed index
	if lastCommit < myLastCommit {
		// leader pushed a early index before commit
		log.WithFields(log.Fields{
			"head":     myLastCommit,
			"supplied": lastCommit,
		}).Warning("invalid last commit log")
		res.Set(&commitResult{err: errors.Wrap(kt.ErrInvalidLog, "invalid last commit log index")})
		return
	}

	// decode prepare log
	var logReq interface{}
	var err error
	if logReq, err = r.doDecodePayload(ctx, prepareLog.Data); err != nil {
		res.Set(&commitResult{err: errors.Wrap(err, "decode log payload failed")})
		return
	}

	tm.Add("decode_payload")

	req := &commitReq{
		ctx:        ctx,
		data:       logReq,
		index:      prepareLog.Index,
		lastCommit: lastCommit,
		result:     res,
		log:        commitLog,
		tm:         tm,
	}

	select {
	case <-ctx.Done():
	case r.commitCh <- req:
	}

	return
}

func (r *Runtime) commitCycle() {
	for {
		var cReq *commitReq

		select {
		case <-r.stopCh:
			return
		case cReq = <-r.commitCh:
		}

		if cReq != nil {
			r.doCommitCycle(cReq)
		}
	}
}

func (r *Runtime) leaderDoCommit(req *commitReq) {
	if req.log != nil {
		// mis-use follower commit for leader
		log.Fatal("INVALID EXISTING LOG FOR LEADER COMMIT")
		return
	}

	req.tm.Add("queue")

	// create leader log
	var (
		l       *kt.Log
		logData []byte
		cr      = &commitResult{}
		err     error
	)

	logData = append(logData, r.uint64ToBytes(req.index)...)
	logData = append(logData, r.uint64ToBytes(atomic.LoadUint64(&r.lastCommit))...)

	if l, err = r.newLog(req.ctx, kt.LogCommit, logData); err != nil {
		// serve error, leader could not write log
		return
	}

	req.tm.Add("write_wal")

	// not wrapping underlying handler commit error
	cr.result, err = r.doCommit(req.ctx, req.data, true)

	req.tm.Add("db_write")

	// mark last commit
	atomic.StoreUint64(&r.lastCommit, l.Index)

	// send commit
	cr.rpc = r.applyRPC(l, r.minCommitFollowers)
	cr.index = l.Index
	cr.err = err

	// TODO(): text log for rpc errors

	// TODO(): mark uncommitted nodes and remove from peers

	req.result.Set(cr)

	req.tm.Add("send_follower_commit")

	return
}

func (r *Runtime) followerDoCommit(req *commitReq) {
	if req.log == nil {
		log.Fatal("NO LOG FOR FOLLOWER COMMIT")
		return
	}

	waitCommitTask := trace.StartRegion(req.ctx, "waitForLastCommit")

	// check for last commit availability
	myLastCommit := atomic.LoadUint64(&r.lastCommit)
	if req.lastCommit != myLastCommit {
		// TODO(): need counter for retries, infinite commit re-order would cause troubles
		go func(req *commitReq) {
			_, _ = r.waitForLog(req.ctx, req.lastCommit)
			r.commitCh <- req
		}(req)
		waitCommitTask.End()
		return
	}

	waitCommitTask.End()
	req.tm.Add("queue")

	defer trace.StartRegion(req.ctx, "commitCycle").End()

	var err error

	// write log first
	if err = r.writeWAL(req.ctx, req.log); err != nil {
		return
	}

	req.tm.Add("write_wal")

	// do commit, not wrapping underlying handler commit error
	_, err = r.doCommit(req.ctx, req.data, false)

	req.tm.Add("db_write")

	// mark last commit
	atomic.StoreUint64(&r.lastCommit, req.log.Index)

	req.result.Set(&commitResult{
		err: err,
	})

	return
}

func (r *Runtime) getPrepareLog(ctx context.Context, l *kt.Log) (lastCommitIndex uint64, pl *kt.Log, err error) {
	defer trace.StartRegion(ctx, "getPrepareLog").End()

	var prepareIndex uint64

	// decode prepare index
	if prepareIndex, err = r.bytesToUint64(l.Data); err != nil {
		err = errors.Wrap(err, "log does not contain valid prepare index")
		return
	}

	if pl, err = r.waitForLog(ctx, prepareIndex); err != nil {
		err = errors.Wrap(err, "wait for prepare log failed")
		return
	}

	// decode commit index
	if len(l.Data) >= 16 {
		lastCommitIndex, _ = r.bytesToUint64(l.Data[8:])

		if _, err = r.waitForLog(ctx, lastCommitIndex); err != nil {
			err = errors.Wrap(err, "wait for last commit log failed")
			return
		}
	}

	return
}

func (r *Runtime) doCommitCycle(req *commitReq) {
	r.peersLock.RLock()
	defer r.peersLock.RUnlock()

	if r.role == proto.Leader {
		defer trace.StartRegion(req.ctx, "commitCycle").End()
		r.leaderDoCommit(req)
	} else {
		r.followerDoCommit(req)
	}
}
