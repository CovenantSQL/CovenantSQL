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

package types

import "github.com/pkg/errors"

var (
	// ErrNotLeader represents current node is not a peer leader.
	ErrNotLeader = errors.New("not leader")
	// ErrNotFollower represents current node is not a peer follower.
	ErrNotFollower = errors.New("not follower")
	// ErrPrepareTimeout represents timeout failure for prepare operation.
	ErrPrepareTimeout = errors.New("prepare timeout")
	// ErrPrepareFailed represents failure for prepare operation.
	ErrPrepareFailed = errors.New("prepare failed")
	// ErrInvalidLog represents log is invalid.
	ErrInvalidLog = errors.New("invalid log")
	// ErrNotInPeer represents current node does not exists in peer list.
	ErrNotInPeer = errors.New("node not in peer")
	// ErrNeedRecovery represents current follower node needs recovery, back-off is required by leader.
	ErrNeedRecovery = errors.New("need recovery")
	// ErrInvalidConfig represents invalid kayak runtime config.
	ErrInvalidConfig = errors.New("invalid runtime config")
	// ErrNotStarted defines error which the runtime is not started or already stopped.
	ErrNotStarted = errors.New("runtime not started")
)
