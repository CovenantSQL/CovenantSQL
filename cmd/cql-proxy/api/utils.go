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
	"github.com/gin-gonic/gin"
)

func abortWithError(c *gin.Context, code int, err error) {
	if err != nil {
		c.AbortWithStatusJSON(code, map[string]interface{}{
			"success": false,
			"msg":     err.Error(),
		})
		_ = c.Error(err)
	}
}

func responseWithData(c *gin.Context, code int, data interface{}) {
	c.JSON(code, map[string]interface{}{
		"success": true,
		"msg":     "",
		"data":    data,
	})
}
