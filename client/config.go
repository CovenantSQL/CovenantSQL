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

package client

import (
	"net/url"
	"strconv"
	"strings"
)

const (
	paramUseLeader    = "use_leader"
	paramUseFollower  = "use_follower"
	paramUseDirectRPC = "use_direct_rpc"
	paramMirror       = "mirror"
)

// Config is a configuration parsed from a DSN string.
type Config struct {
	DatabaseID string

	// additional configs should be filled
	// such as read/write/exec timeout
	// currently no timeout is supported.

	// UseLeader use leader nodes to do queries
	UseLeader bool

	// UseFollower use follower nodes to do queries
	UseFollower bool

	// UseDirectRPC use direct RPC to access the miner
	UseDirectRPC bool

	// Mirror option forces client to query from mirror server
	Mirror string
}

// NewConfig creates a new config with default value.
func NewConfig() *Config {
	return &Config{UseLeader: true}
}

// FormatDSN formats the given Config into a DSN string which can be passed to the driver.
func (cfg *Config) FormatDSN() string {
	// build query from arguments

	u := &url.URL{
		Scheme: DBScheme,
		Host:   cfg.DatabaseID,
	}

	newQuery := u.Query()
	if cfg.UseFollower {
		newQuery.Add(paramUseFollower, strconv.FormatBool(cfg.UseFollower))

		if cfg.UseLeader {
			newQuery.Add(paramUseLeader, strconv.FormatBool(cfg.UseLeader))
		}
	}
	if cfg.Mirror != "" {
		newQuery.Add(paramMirror, cfg.Mirror)
	}
	if cfg.UseDirectRPC {
		newQuery.Add(paramUseDirectRPC, strconv.FormatBool(cfg.UseDirectRPC))
	}
	u.RawQuery = newQuery.Encode()

	return u.String()
}

// ParseDSN parse the DSN string to a Config.
func ParseDSN(dsn string) (cfg *Config, err error) {
	if !strings.HasPrefix(dsn, DBScheme) && !strings.HasPrefix(dsn, DBSchemeAlias) {
		dsn = DBScheme + "://" + dsn
	}

	var u *url.URL
	if u, err = url.Parse(dsn); err != nil {
		return nil, err
	}

	cfg = NewConfig()
	cfg.DatabaseID = u.Host

	q := u.Query()
	// option: use_leader, use_follower
	cfg.UseLeader, _ = strconv.ParseBool(q.Get(paramUseLeader))
	cfg.UseFollower, _ = strconv.ParseBool(q.Get(paramUseFollower))
	if !cfg.UseLeader && !cfg.UseFollower {
		cfg.UseLeader = true
	}
	cfg.Mirror = q.Get(paramMirror)
	cfg.UseDirectRPC, _ = strconv.ParseBool(q.Get(paramUseDirectRPC))

	return cfg, nil
}
