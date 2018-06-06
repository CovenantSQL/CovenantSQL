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

import "context"

// ETLSRequest defines accepted request payload.
type ETLSRequest struct {
}

// GetServerID implements Request.GetServerID.
func (er *ETLSRequest) GetServerID() ServerID {
	return ""
}

// GetMethod implements Request.GetMethod.
func (er *ETLSRequest) GetMethod() string {
	return ""
}

// GetRequest implements Request.GetRequest.
func (er *ETLSRequest) GetRequest() interface{} {
	return nil
}

// SendResponse implements Request.SendResponse.
func (er *ETLSRequest) SendResponse(resp interface{}) error {
	return nil
}

// ETLSTransport defines connection resolution feature of kayak.
type ETLSTransport struct {
}

// Request implements Transport.Request.
func (et *ETLSTransport) Request(ctx context.Context, serverID ServerID, method string,
	args interface{}, response interface{}) error {
	return nil
}

// Process implements Transport.Process.
func (et *ETLSTransport) Process() <-chan Request {
	return nil
}

var (
	_ Request   = &ETLSRequest{}
	_ Transport = &ETLSTransport{}
)
