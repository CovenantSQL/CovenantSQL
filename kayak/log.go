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
	"io"
	"log"

	kt "github.com/CovenantSQL/CovenantSQL/kayak/types"
	"github.com/CovenantSQL/CovenantSQL/utils/trace"
	"github.com/pkg/errors"
)

func (r *Runtime) newLog(ctx context.Context, logType kt.LogType, data []byte) (l *kt.Log, err error) {
	defer trace.StartRegion(ctx, "newWAL").End()

	// allocate index
	r.nextIndexLock.Lock()
	i := r.nextIndex
	r.nextIndex++
	r.nextIndexLock.Unlock()
	l = &kt.Log{
		LogHeader: kt.LogHeader{
			Index:    i,
			Type:     logType,
			Producer: r.nodeID,
		},
		Data: data,
	}

	// error write will be a fatal error, cause to node to fail fast
	if err = r.wal.Write(l); err != nil {
		log.Fatalf("WRITE LOG FAILED: %v", err)
	}

	return
}

func (r *Runtime) writeWAL(ctx context.Context, l *kt.Log) (err error) {
	defer trace.StartRegion(ctx, "writeWal").End()

	if err = r.wal.Write(l); err != nil {
		err = errors.Wrap(err, "write follower log failed")
	}

	return
}

func (r *Runtime) readLogs() (err error) {
	// load logs, only called during init
	var l *kt.Log

	for {
		if l, err = r.wal.Read(); err != nil && err != io.EOF {
			err = errors.Wrap(err, "load previous logs in wal failed")
			return
		} else if err == io.EOF {
			err = nil
			break
		}

		switch l.Type {
		case kt.LogPrepare:
			// record in pending prepares
			r.pendingPrepares[l.Index] = true
		case kt.LogCommit:
			// record last commit
			var lastCommit uint64
			var prepareLog *kt.Log
			if lastCommit, prepareLog, err = r.getPrepareLog(context.Background(), l); err != nil {
				err = errors.Wrap(err, "previous prepare does not exists, node need full recovery")
				return
			}
			if lastCommit != r.lastCommit {
				err = errors.Wrapf(err,
					"last commit record in wal mismatched (expected: %v, actual: %v)", r.lastCommit, lastCommit)
				return
			}
			if !r.pendingPrepares[prepareLog.Index] {
				err = errors.Wrap(kt.ErrInvalidLog, "previous prepare already committed/rollback")
				return
			}
			r.lastCommit = l.Index
			// resolve previous prepared
			delete(r.pendingPrepares, prepareLog.Index)
		case kt.LogRollback:
			var prepareLog *kt.Log
			if _, prepareLog, err = r.getPrepareLog(context.Background(), l); err != nil {
				err = errors.Wrap(err, "previous prepare does not exists, node need full recovery")
				return
			}
			if !r.pendingPrepares[prepareLog.Index] {
				err = errors.Wrap(kt.ErrInvalidLog, "previous prepare already committed/rollback")
				return
			}
			// resolve previous prepared
			delete(r.pendingPrepares, prepareLog.Index)
		default:
			err = errors.Wrapf(kt.ErrInvalidLog, "invalid log type: %v", l.Type)
			return
		}

		// record nextIndex
		r.updateNextIndex(context.Background(), l)
	}

	return
}
