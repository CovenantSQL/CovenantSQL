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

// TwoPCRunner is a Runner implementation organizing two phase commit mutation
type TwoPCRunner struct {
}

// Init implements Runner.Init.
func (tpr *TwoPCRunner) Init(config *Config, peers *Peers, log LogStore, stable StableStore, dialer Dialer) error {
	// TODO
	return nil
}

// UpdatePeers implements Runner.UpdatePeers.
func (tpr *TwoPCRunner) UpdatePeers(peers *Peers) error {
	// TODO
	return nil
}

// Process implements Runner.Process.
func (tpr *TwoPCRunner) Process(log *Log) error {
	// TODO
	return nil
}

// Shutdown implements Runner.Shutdown.
func (tpr *TwoPCRunner) Shutdown() error {
	// TODO
	return nil
}

var (
	_ Runner = &TwoPCRunner{}
)
