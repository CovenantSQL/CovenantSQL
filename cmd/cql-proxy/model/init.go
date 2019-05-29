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

import (
	"gopkg.in/gorp.v1"
	"sync"
)

var tableMapping sync.Map

type regTable struct {
	name       string
	object     interface{}
	pk         string
	isAutoIncr bool
}

func RegisterModel(tableName string, object interface{}, pk string, isAutoIncr bool) {
	tableMapping.Store(tableName, &regTable{
		name:       tableName,
		object:     object,
		pk:         pk,
		isAutoIncr: isAutoIncr,
	})
}

func AddTables(dbMap *gorp.DbMap) {
	tableMapping.Range(func(_, value interface{}) bool {
		table := value.(*regTable)
		tableMap := dbMap.AddTableWithName(table.object, table.name)
		if table.pk != "" {
			tableMap.SetKeys(table.isAutoIncr, table.pk)
		}

		return true
	})

	return
}
