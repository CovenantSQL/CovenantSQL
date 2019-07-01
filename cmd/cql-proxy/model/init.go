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

package model

import gorp "gopkg.in/gorp.v2"

// AddTables register tables to gorp database map.
func AddTables(dbMap *gorp.DbMap) {
	dbMap.AddTableWithName(Developer{}, "developer").
		SetKeys(true, "ID").
		ColMap("GithubID").SetUnique(true)
	dbMap.AddTableWithName(Session{}, "session").
		SetKeys(false, "ID")
	dbMap.AddTableWithName(TokenApply{}, "token_apply").
		SetKeys(false, "ID")
	dbMap.AddTableWithName(DeveloperPrivateKey{}, "private_keys").
		SetKeys(true, "ID")
	dbMap.AddTableWithName(Task{}, "task").
		SetKeys(true, "ID")
	tblProject := dbMap.AddTableWithName(Project{}, "project").
		SetKeys(true, "ID")
	tblProject.ColMap("Alias").SetUnique(true)
	tblProject.ColMap("DB").SetUnique(true)

	return
}
