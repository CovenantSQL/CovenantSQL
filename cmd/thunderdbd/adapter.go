/*
 * Copyright 2018 The ThunderDB Authors.
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

package main

import (
	"context"
	"encoding/json"

	"gitlab.com/thunderdb/ThunderDB/kayak"
	"gitlab.com/thunderdb/ThunderDB/sqlchain/storage"
	"gitlab.com/thunderdb/ThunderDB/twopc"
)

// PayloadCodec defines simple codec.
type PayloadCodec struct{}

func (m *PayloadCodec) Encode(execLog *storage.ExecLog) ([]byte, error) {
	return json.Marshal(execLog)
}

func (m *PayloadCodec) Decode(data []byte, execLog *storage.ExecLog) (err error) {
	return json.Unmarshal(data, execLog)
}

type Storage struct {
	*storage.Storage
}

func initStorage(dbFile string) (stor *Storage, err error) {
	var st *storage.Storage
	if st, err = storage.New(dbFile); err != nil {
		return
	}

	stor = &Storage{
		Storage: st,
	}
	return
}

func (s *Storage) Prepare(ctx context.Context, wb twopc.WriteBatch) (err error) {
	var execLog *storage.ExecLog
	if execLog, err = s.decodeLog(wb); err != nil {
		return
	}

	return s.Storage.Prepare(ctx, execLog)
}

func (s *Storage) Commit(ctx context.Context, wb twopc.WriteBatch) (err error) {
	var execLog *storage.ExecLog
	if execLog, err = s.decodeLog(wb); err != nil {
		return
	}

	return s.Storage.Commit(ctx, execLog)
}

func (s *Storage) Rollback(ctx context.Context, wb twopc.WriteBatch) (err error) {
	var execLog *storage.ExecLog
	if execLog, err = s.decodeLog(wb); err != nil {
		return
	}

	return s.Storage.Rollback(ctx, execLog)
}

func (s *Storage) decodeLog(wb twopc.WriteBatch) (log *storage.ExecLog, err error) {
	var bytesPayload []byte
	var execLog storage.ExecLog
	var ok bool

	if bytesPayload, ok = wb.([]byte); !ok {
		err = kayak.ErrInvalidLog
		return
	}
	if err = payloadCodec.Decode(bytesPayload, &execLog); err != nil {
		return
	}

	log = &execLog
	return
}

type StubServer struct {
	Runtime *kayak.Runtime
	Storage *Storage
}

type ResponseRows []map[string]interface{}

func (s *StubServer) Write(sql string, _ *ResponseRows) (err error) {
	var writeData []byte

	l := &storage.ExecLog{
		Queries: []string{sql},
	}

	if writeData, err = payloadCodec.Encode(l); err != nil {
		return err
	}

	err = s.Runtime.Apply(writeData)

	return
}

func (s *StubServer) Read(sql string, rows *ResponseRows) (err error) {
	var columns []string
	var result [][]interface{}
	columns, _, result, err = s.Storage.Query(context.Background(), []string{sql})

	// rebuild map
	*rows = make([]map[string]interface{}, 0, len(result))

	for _, r := range result {
		// build row map
		row := make(map[string]interface{})

		for i, c := range r {
			row[columns[i]] = c
		}

		*rows = append(*rows, row)
	}

	return
}
