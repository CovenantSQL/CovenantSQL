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

package casbin

import (
	"github.com/casbin/casbin"
	"github.com/casbin/casbin/model"
	"github.com/casbin/casbin/persist"
)

const modelConf = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _
g2 = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && g2(r.obj, p.obj) && r.act == p.act`

// NewCasbin returns a casbin enforcer instance with
// fixed rbac_model_with_resource_roles model and custom structured config.
func NewCasbin(cfg *Config) (e *casbin.Enforcer, err error) {
	var (
		m    model.Model
		rule persist.Adapter
	)
	m = casbin.NewModel(modelConf)
	if rule, err = NewAdapter(cfg); err != nil {
		return
	}
	e = casbin.NewEnforcer(m, rule)
	return
}
