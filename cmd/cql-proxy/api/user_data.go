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

package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	gorp "gopkg.in/gorp.v2"

	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/model"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/resolver"
)

func userDataFind(c *gin.Context) {
	r := struct {
		Table      string                 `json:"table" form:"table" uri:"table" binding:"required,max=128"`
		Filter     map[string]interface{} `json:"filter" form:"filter"`
		Projection map[string]interface{} `json:"projection" form:"projection"`
		OrderBy    map[string]interface{} `json:"order" form:"order"`
		Skip       *int64                 `json:"skip" form:"skip" binding:"omitempty,gte=0"`
		Limit      *int64                 `json:"limit" form:"limit" binding:"omitempty,gte=0"`
	}{}

	_ = c.ShouldBindUri(&r)

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	db, uid, userState, vars, rules, fieldMap, adminMode, err := buildExecuteContext(c, r.Table)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	var filter map[string]interface{}

	if !adminMode {
		filter, err = rules.EnforceRulesOnFilter(r.Filter, r.Table, uid, userState, vars, resolver.RuleQueryFind)
		if err != nil {
			abortWithError(c, http.StatusForbidden, err)
			return
		}
	} else {
		filter = r.Filter
	}

	stmt, args, _, err := resolver.Find(r.Table, fieldMap, filter, r.Projection, r.OrderBy, r.Skip, r.Limit)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	var rows *sql.Rows
	rows, err = db.Query(stmt, args...)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	var result []gin.H
	result, err = scanRows(rows)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	responseWithData(c, http.StatusOK, result)
}

func userDataInsert(c *gin.Context) {
	r := struct {
		Table string                 `json:"table" form:"table" uri:"table" binding:"required,max=128"`
		Data  map[string]interface{} `json:"data" form:"data"`
	}{}

	_ = c.ShouldBindUri(&r)

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	db, uid, userState, vars, rules, fieldMap, adminMode, err := buildExecuteContext(c, r.Table)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	var insertData map[string]interface{}

	if !adminMode {
		insertData, err = rules.EnforceRulesOnInsert(r.Data, r.Table, uid, userState, vars)
		if err != nil {
			abortWithError(c, http.StatusForbidden, err)
			return
		}
	} else {
		insertData = r.Data
	}

	stmt, args, _, err := resolver.Insert(r.Table, fieldMap, insertData)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	result, err := db.Exec(stmt, args...)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	responseWithData(c, http.StatusOK, gin.H{
		"last_insert_id": mustGetInt64Var(result.LastInsertId()),
		"affected_rows":  mustGetInt64Var(result.RowsAffected()),
	})
}

func userDataUpdate(c *gin.Context) {
	r := struct {
		Table   string                 `json:"table" form:"table" uri:"table" binding:"required,max=128"`
		Filter  map[string]interface{} `json:"filter" form:"filter"`
		Update  map[string]interface{} `json:"update" form:"update"`
		JustOne bool                   `json:"one" form:"one"`
	}{}

	_ = c.ShouldBindUri(&r)

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	db, uid, userState, vars, rules, fieldMap, adminMode, err := buildExecuteContext(c, r.Table)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	var (
		filter map[string]interface{}
		update map[string]interface{}
	)

	if !adminMode {
		filter, err = rules.EnforceRulesOnFilter(r.Filter, r.Table, uid, userState, vars, resolver.RuleQueryUpdate)
		if err != nil {
			abortWithError(c, http.StatusForbidden, err)
			return
		}

		update, err = rules.EnforceRulesOnUpdate(r.Update, r.Table, uid, userState, vars)
		if err != nil {
			abortWithError(c, http.StatusForbidden, err)
			return
		}
	} else {
		filter = r.Filter
		update = r.Update
	}

	stmt, args, _, err := resolver.Update(r.Table, fieldMap, filter, update, r.JustOne)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	result, err := db.Exec(stmt, args...)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	responseWithData(c, http.StatusOK, gin.H{
		"affected_rows": mustGetInt64Var(result.RowsAffected()),
	})
}

func userDataRemove(c *gin.Context) {
	r := struct {
		Table   string                 `json:"table" form:"table" uri:"table" binding:"required,max=128"`
		Filter  map[string]interface{} `json:"filter" form:"filter"`
		JustOne bool                   `json:"one" form:"one"`
	}{}

	_ = c.ShouldBindUri(&r)

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	db, uid, userState, vars, rules, fieldMap, adminMode, err := buildExecuteContext(c, r.Table)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	var filter map[string]interface{}

	if !adminMode {
		filter, err = rules.EnforceRulesOnFilter(r.Filter, r.Table, uid, userState, vars, resolver.RuleQueryRemove)
		if err != nil {
			abortWithError(c, http.StatusForbidden, err)
			return
		}
	} else {
		filter = r.Filter
	}

	stmt, args, _, err := resolver.Remove(r.Table, fieldMap, filter, r.JustOne)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	result, err := db.Exec(stmt, args...)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	responseWithData(c, http.StatusOK, gin.H{
		"affected_rows": mustGetInt64Var(result.RowsAffected()),
	})
}

func userDataCount(c *gin.Context) {
	r := struct {
		Table  string                 `json:"table" form:"table" uri:"table" binding:"required,max=128"`
		Filter map[string]interface{} `json:"filter" form:"filter"`
	}{}

	_ = c.ShouldBindUri(&r)

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	db, uid, userState, vars, rules, fieldMap, adminMode, err := buildExecuteContext(c, r.Table)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	var filter map[string]interface{}

	if !adminMode {
		filter, err = rules.EnforceRulesOnFilter(r.Filter, r.Table, uid, userState, vars, resolver.RuleQueryCount)
		if err != nil {
			abortWithError(c, http.StatusForbidden, err)
			return
		}
	} else {
		filter = r.Filter
	}

	stmt, args, _, err := resolver.Count(r.Table, fieldMap, filter)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	count, err := db.SelectInt(stmt, args...)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	responseWithData(c, http.StatusOK, gin.H{
		"count": count,
	})
}

func buildExecuteContext(c *gin.Context, tableName string) (projectDB *gorp.DbMap, uid string, userState string,
	vars map[string]interface{}, r *resolver.Rules, fields resolver.FieldMap, adminMode bool, err error) {
	project := getCurrentProject(c)
	projectDB, err = getCurrentProjectDB(c)
	if err != nil {
		return
	}

	var (
		userID      = getUserID(c)
		developerID = getDeveloperID(c)
		userInfo    *model.ProjectUser
	)

	if userID != 0 {
		userInfo, err = model.GetProjectUser(projectDB, userID)
		if err != nil {
			return
		}

		uid = fmt.Sprint(userID)
	}
	if developerID != 0 && project.Developer == developerID {
		// table accessing from admin mode, check admin project belonging
		adminMode = true
	}

	vars = map[string]interface{}{}

	if userInfo == nil {
		vars["user_id"] = 0
		vars["user_name"] = nil
		vars["user_email"] = nil
		vars["user_provider"] = nil
		vars["user_created"] = nil
		vars["user_last_login"] = nil

		userState = resolver.UserStateAnonymous
	} else {
		vars["user_id"] = userInfo.ID
		vars["user_name"] = userInfo.Name
		vars["user_email"] = userInfo.Email
		vars["user_provider"] = userInfo.Provider
		vars["user_created"] = userInfo.Created
		vars["user_last_login"] = userInfo.LastLogin

		switch userInfo.State {
		case model.ProjectUserStateEnabled:
			userState = resolver.UserStateLoggedIn
		case model.ProjectUserStateDisabled:
			userState = resolver.UserStateDisabled
		case model.ProjectUserStatePreRegistered:
			userState = resolver.UserStatePreRegistered
		case model.ProjectUserStateWaitSignedConfirm:
			userState = resolver.UserStateWaitSignUpConfirm
		}
	}

	r, err = loadRules(c, project.DB, projectDB)
	if err != nil {
		return
	}

	// load table fields
	_, ptc, err := model.GetProjectTableConfig(projectDB, tableName)
	if err != nil {
		return
	}

	if ptc.IsDeleted {
		err = errors.New("table does not exists")
		return
	}

	fields = resolver.FieldMap{}

	for _, c := range ptc.Columns {
		fields[c] = true
	}

	return
}

func mustGetInt64Var(i int64, err error) int64 {
	_ = err
	return i
}

func scanRows(rows *sql.Rows) (result []gin.H, err error) {
	defer func() {
		_ = rows.Close()
	}()

	var columns []string
	columns, err = rows.Columns()
	if err != nil {
		return
	}

	for rows.Next() {
		var (
			row  = make([]interface{}, len(columns))
			dest = make([]interface{}, len(columns))
		)

		for i := range row {
			dest[i] = &row[i]
		}
		if err = rows.Scan(dest...); err != nil {
			return
		}

		target := gin.H{}

		for i, col := range columns {
			switch v := row[i].(type) {
			case []byte:
				target[col] = string(v)
			default:
				target[col] = v
			}
		}

		result = append(result, target)
	}

	return
}
