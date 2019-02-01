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

// Handler defines the main underlying fsm of kayak.
type Handler interface {
	EncodePayload(req interface{}) (data []byte, err error)
	DecodePayload(data []byte) (req interface{}, err error)
	Check(request interface{}) error
	Commit(request interface{}, isLeader bool) (result interface{}, err error)
}
