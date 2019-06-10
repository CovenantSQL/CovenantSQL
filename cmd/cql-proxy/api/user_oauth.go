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

import "github.com/gin-gonic/gin"

func userOAuthAuthorize(c *gin.Context) {
	// load providers from project config

}

func userOAuthCallback(c *gin.Context) {

}

func userCheck(c *gin.Context) {

}

func projectIDInject(c *gin.Context) {

	c.Next()
}

func getUserInfo(c *gin.Context) {

}
