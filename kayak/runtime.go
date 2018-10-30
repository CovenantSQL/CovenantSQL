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

package kayak

import (
	"fmt"
	"path/filepath"

	"github.com/CovenantSQL/CovenantSQL/proto"
)

const (
	// FileStorePath is the default log store filename
	FileStorePath = "kayak.db"
)

// Runtime defines common init/shutdown logic for different consensus protocol runner.
type Runtime struct {
	config       *RuntimeConfig
	runnerConfig Config
	peers        *Peers
	isLeader     bool
	logStore     *BoltStore
}

// NewRuntime creates new runtime.
func NewRuntime(config Config, peers *Peers) (*Runtime, error) {
	if config == nil || peers == nil {
		return nil, ErrInvalidConfig
	}

	// config authentication check
	if !peers.Verify() {
		return nil, ErrInvalidConfig
	}

	// peers config verification
	serverInPeers := false
	runtime := &Runtime{
		config:       config.GetRuntimeConfig(),
		peers:        peers,
		runnerConfig: config,
	}

	for _, s := range peers.Servers {
		if s.ID == runtime.config.LocalID {
			serverInPeers = true

			if s.Role == proto.Leader {
				runtime.isLeader = true
			}
		}
	}

	if !serverInPeers {
		return nil, ErrInvalidConfig
	}

	return runtime, nil
}

// Init defines the common init logic.
func (r *Runtime) Init() (err error) {
	// init log store
	var logStore *BoltStore

	if logStore, err = NewBoltStore(filepath.Join(r.config.RootDir, FileStorePath)); err != nil {
		return fmt.Errorf("new bolt store: %s", err.Error())
	}

	// call transport init
	if err = r.config.Transport.Init(); err != nil {
		return
	}

	// call runner init
	if err = r.config.Runner.Init(r.runnerConfig, r.peers, logStore, logStore, r.config.Transport); err != nil {
		logStore.Close()
		return fmt.Errorf("%s runner init: %s", r.config.LocalID, err.Error())
	}
	r.logStore = logStore

	return nil
}

// Shutdown defines common shutdown logic.
func (r *Runtime) Shutdown() (err error) {
	if err = r.config.Runner.Shutdown(true); err != nil {
		return fmt.Errorf("%s runner shutdown: %s", r.config.LocalID, err.Error())
	}

	if err = r.config.Transport.Shutdown(); err != nil {
		return
	}

	if r.logStore != nil {
		if err = r.logStore.Close(); err != nil {
			return fmt.Errorf("shutdown bolt store: %s", err.Error())
		}

		r.logStore = nil
	}

	return nil
}

// Apply defines common process logic.
func (r *Runtime) Apply(data []byte) (result interface{}, offset uint64, err error) {
	// validate if myself is leader
	if !r.isLeader {
		return nil, 0, ErrNotLeader
	}

	result, offset, err = r.config.Runner.Apply(data)
	if err != nil {
		return nil, 0, err
	}

	return
}

// GetLog fetches runtime log produced by runner.
func (r *Runtime) GetLog(offset uint64) (data []byte, err error) {
	var l Log
	if err = r.logStore.GetLog(offset, &l); err != nil {
		return
	}

	data = l.Data

	return
}

// UpdatePeers defines common peers update logic.
func (r *Runtime) UpdatePeers(peers *Peers) error {
	// Verify peers
	if !peers.Verify() {
		return ErrInvalidConfig
	}

	// Check if myself is still in peers
	inPeers := false
	isLeader := false

	for _, s := range peers.Servers {
		if s.ID == r.config.LocalID {
			inPeers = true
			isLeader = s.Role == proto.Leader
			break
		}
	}

	if !inPeers {
		// shutdown
		return r.Shutdown()
	}

	if err := r.config.Runner.UpdatePeers(peers); err != nil {
		return fmt.Errorf("update peers to %s: %s", peers, err.Error())
	}

	r.isLeader = isLeader

	return nil
}
