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

package client

type execResult struct {
	affectedRows int64
	lastInsertID int64
}

// LastInsertId return last inserted ID.
func (r *execResult) LastInsertId() (int64, error) {
	return r.lastInsertID, nil
}

// RowsAffected return how many rows affected.
func (r *execResult) RowsAffected() (int64, error) {
	return r.affectedRows, nil
}
