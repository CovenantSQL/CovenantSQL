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
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/model"
)

func listTasks(c *gin.Context) {
	r := struct {
		All    bool  `json:"all" form:"all"`
		Offset int64 `json:"offset" form:"offset" binding:"gte=0"`
		Limit  int64 `json:"limit" form:"limit" binding:"gte=0"`
	}{}

	if r.Limit == 0 {
		r.Limit = 20
	}

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	developer := getDeveloperID(c)

	p, err := model.GetMainAccount(model.GetDB(c), developer)
	if err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	tasks, total, err := model.ListTask(model.GetDB(c), developer, p.ID, r.All, r.Offset, r.Limit)
	if err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusInternalServerError, ErrGetTaskListFailed)
		return
	}

	var resp []gin.H

	for _, t := range tasks {
		resp = append(resp, gin.H{
			"id":       t.ID,
			"type":     t.Type.String(),
			"state":    t.State.String(),
			"created":  formatUnixTime(t.Created),
			"updated":  formatUnixTime(t.Updated),
			"finished": formatUnixTime(t.Finished),
		})
	}

	responseWithData(c, http.StatusOK, gin.H{
		"tasks": resp,
		"total": total,
	})
}

func getTask(c *gin.Context) {
	r := struct {
		ID int64 `json:"id" form:"id" uri:"id" binding:"required"`
	}{}

	_ = c.ShouldBindUri(&r)

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	developer := getDeveloperID(c)
	task, err := model.GetTask(model.GetDB(c), developer, r.ID)
	if err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusForbidden, ErrGetTaskDetailFailed)
		return
	}

	responseWithData(c, http.StatusOK, gin.H{
		"task": func(t *model.Task) gin.H {
			return gin.H{
				"id":       t.ID,
				"type":     t.Type.String(),
				"state":    t.State.String(),
				"args":     t.Args,
				"result":   t.Result,
				"created":  formatUnixTime(t.Created),
				"updated":  formatUnixTime(t.Updated),
				"finished": formatUnixTime(t.Finished),
			}
		}(task),
	})
}

func cancelTask(c *gin.Context) {
	r := struct {
		ID int64 `json:"id" form:"id" uri:"id" binding:"required"`
	}{}

	_ = c.ShouldBindUri(&r)

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	developer := getDeveloperID(c)
	task, err := model.GetTask(model.GetDB(c), developer, r.ID)
	if err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusForbidden, ErrGetTaskDetailFailed)
		return
	}

	// call task executor to abort
	getTaskManager(c).Kill(task.ID)
}
