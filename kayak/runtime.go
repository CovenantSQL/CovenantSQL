/*
 * Copyright 2018 The ThunderDB Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the “License”);
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an “AS IS” BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package kayak

import (
	"errors"
	"fmt"
	"path/filepath"
)

const (
	// FileStorePath is the default log store filename
	FileStorePath = "kayak.db"
)

var (
	// ErrInvalidConfig defines invalid config error
	ErrInvalidConfig = errors.New("invalid configuration")
	// ErrInvalidLog defines invalid log error
	ErrInvalidLog = errors.New("invalid log")
	// ErrNotLeader defines not leader on log processing
	ErrNotLeader = errors.New("not leader")
)

// Runtime defines common init/shutdown logic for different consensus protocol runner
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

			if s.Role == Leader {
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
func (r *Runtime) Init() error {
	// init log store
	logStore, err := NewBoltStore(filepath.Join(r.config.RootDir, FileStorePath))
	if err != nil {
		return fmt.Errorf("new bolt store: %s", err.Error())
	}

	// call runner init
	err = r.config.Runner.Init(r.runnerConfig, r.peers, logStore, logStore, r.config.Transport)
	if err != nil {
		logStore.Close()
		return fmt.Errorf("%s runner init: %s", r.config.LocalID, err.Error())
	}
	r.logStore = logStore

	return nil
}

// Shutdown defines common shutdown logic.
func (r *Runtime) Shutdown() error {
	err := r.config.Runner.Shutdown(true)
	if err != nil {
		return fmt.Errorf("%s runner shutdown: %s", r.config.LocalID, err.Error())
	}

	if r.logStore != nil {
		err = r.logStore.Close()
		if err != nil {
			return fmt.Errorf("shutdown bolt store: %s", err.Error())
		}

		r.logStore = nil
	}

	return nil
}

// Process defines common process logic.
func (r *Runtime) Process(data []byte) error {
	// validate if myself is leader
	if !r.isLeader {
		return ErrNotLeader
	}

	err := r.config.Runner.Process(data)
	if err != nil {
		return fmt.Errorf("process log: %s", err.Error())
	}

	return nil
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
			isLeader = s.Role == Leader
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
