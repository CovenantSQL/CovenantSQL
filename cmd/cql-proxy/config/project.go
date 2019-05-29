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

package config

const (
	ConfigEnabled       = "enabled"
	ConfigAdminUsers    = "admin"
	ConfigHosts         = "hosts"
	ConfigAuthTraffic   = "auth_traffic"
	ConfigAuthProviders = "auth_providers"
	ConfigAuthAnonymous = "auth_anonymous"
	ConfigAuthSignUp    = "auth_sign_up"
	ConfigACL           = "acl"
)

// RuleItem defines project ACL item.
type RuleItem struct {
	Table string `yaml:"table"`
	Read  string `yaml:"read"`
	Write string `yaml:"write"`
}

// RulesConfig defines project ACL rules config.
type RulesConfig []RuleItem
