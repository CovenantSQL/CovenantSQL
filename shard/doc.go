/*
 * Copyright 2018 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package shard

/****** TODO List with order ******/

// - Driver
// 	- implement ShardingRows.Columns
// 	- implement ShardingRows.Close
// 	- implement ShardingRows.Next
// - process auto increase column
//  - "In other words, the purpose of AUTOINCREMENT is to prevent the reuse of ROWIDs from previously deleted rows."
//  - we should change the primary key of first inserted row in shard table
//  - SHARDCONF use json
// - handle DDL query
// - Optimize temp table create, current implementation copy too much to temp table
