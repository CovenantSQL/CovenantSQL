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

import "C"
import (
	"encoding/json"
	"time"

	gorp "gopkg.in/gorp.v2"
)

type ProjectConfigType int16

const (
	ProjectConfigMisc ProjectConfigType = iota
	ProjectConfigOAuth
	ProjectConfigTable
)

func (c ProjectConfigType) String() string {
	switch c {
	case ProjectConfigMisc:
		return "Misc"
	case ProjectConfigOAuth:
		return "OAuth"
	case ProjectConfigTable:
		return "Table"
	default:
		return "Unknown"
	}
}

type ProjectConfig struct {
	ID          int64             `db:"id"`
	Type        ProjectConfigType `db:"type"`
	Key         string            `db:"key"`
	RawValue    []byte            `db:"value"`
	Created     int64             `db:"created"`
	LastUpdated int64             `db:"last_updated"`
	Value       interface{}       `db:"-"`
}

type ProjectMiscConfig struct {
	Alias                    string        `json:"alias,omitempty" form:"alias" binding:"omitempty,alphanum,min=1,max=16"`
	Enabled                  *bool         `json:"enabled,omitempty" form:"enabled"`
	EnableSignUp             *bool         `json:"enable_sign_up,omitempty" form:"enable_sign_up"`
	EnableSignUpVerification *bool         `json:"sign_up_verify,omitempty" form:"sign_up_verify"`
	SessionAge               time.Duration `json:"session_age" form:"session_age"`
}

func (c *ProjectMiscConfig) SupportSignUp() bool {
	return c != nil && c.EnableSignUp != nil && *c.EnableSignUp
}

func (c *ProjectMiscConfig) ShouldVerifyAfterSignUp() bool {
	return c != nil && c.EnableSignUpVerification != nil && *c.EnableSignUpVerification
}

type ProjectOAuthConfig struct {
	ClientID     string `json:"client_id" form:"client_id"`
	ClientSecret string `json:"client_secret" form:"client_secret"`
	Enabled      *bool  `json:"enabled,omitempty" form:"enabled"`
}

func (c *ProjectOAuthConfig) IsEnabled() bool {
	return c != nil && c.Enabled != nil && *c.Enabled
}

type ProjectTableConfig struct {
	Columns   []string          `json:"columns"`
	Types     []string          `json:"types"`
	Keys      map[string]string `json:"keys"`
	Rules     string            `json:"rules"`
	IsDeleted bool              `json:"is_deleted"`
}

func GetAllProjectConfig(db *gorp.DbMap) (p []*ProjectConfig, err error) {
	_, err = db.Select(&p, `SELECT * FROM "____config"`)
	if err != nil {
		return
	}

	for _, pc := range p {
		_ = json.Unmarshal(pc.RawValue, &pc.Value)
	}

	return
}

func GetProjectOAuthConfig(db *gorp.DbMap, provider string) (p *ProjectConfig, pc *ProjectOAuthConfig, err error) {
	err = db.SelectOne(&p, `SELECT * FROM "____config" WHERE "type" = ? AND "key" = ? LIMIT 1`,
		ProjectConfigOAuth, provider)
	if err != nil {
		return
	}

	err = json.Unmarshal(p.RawValue, &pc)
	if err == nil {
		p.Value = pc
	}

	return
}

func GetProjectTableConfig(db *gorp.DbMap, tableName string) (p *ProjectConfig, pc *ProjectTableConfig, err error) {
	err = db.SelectOne(&p, `SELECT * FROM "____config" WHERE "type" = ? AND "key" = ? LIMIT 1`,
		ProjectConfigTable, tableName)
	if err != nil {
		return
	}

	err = json.Unmarshal(p.RawValue, &pc)
	if err == nil {
		p.Value = pc
	}

	return
}

func GetProjectTablesConfig(db *gorp.DbMap) (tables []string, err error) {
	var projects []*ProjectConfig

	_, err = db.Select(&projects, `SELECT * FROM "____config" WHERE "type" = ?`, ProjectConfigTable)
	if err != nil {
		return
	}

	for _, p := range projects {
		var ptc *ProjectTableConfig

		_ = json.Unmarshal(p.RawValue, &ptc)

		if ptc != nil && !ptc.IsDeleted {
			tables = append(tables, p.Key)
		}
	}

	return
}

func GetProjectMiscConfig(db *gorp.DbMap) (p *ProjectConfig, pc *ProjectMiscConfig, err error) {
	err = db.SelectOne(&p, `SELECT * FROM "____config" WHERE "type" = ? LIMIT 1`,
		ProjectConfigMisc)
	if err != nil {
		return
	}

	err = json.Unmarshal(p.RawValue, &pc)
	if err == nil {
		p.Value = pc
	}

	return
}

func AddProjectConfig(db *gorp.DbMap, configType ProjectConfigType, configKey string, value interface{}) (p *ProjectConfig, err error) {
	p = &ProjectConfig{
		Type:    configType,
		Key:     configKey,
		Value:   value,
		Created: time.Now().Unix(),
	}

	p.RawValue, err = json.Marshal(p.Value)
	if err != nil {
		return
	}

	err = db.Insert(p)

	return
}

func UpdateProjectConfig(db *gorp.DbMap, p *ProjectConfig) (err error) {
	p.RawValue, err = json.Marshal(p.Value)
	if err != nil {
		return
	}

	p.LastUpdated = time.Now().Unix()

	_, err = db.Update(p)

	return
}
