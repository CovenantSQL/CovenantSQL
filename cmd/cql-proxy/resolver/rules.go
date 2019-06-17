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

package resolver

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/pkg/errors"
	validator "gopkg.in/go-playground/validator.v9"

	"github.com/CovenantSQL/CovenantSQL/proto"
)

type RuleQueryType uint16

const (
	RuleQueryInsert RuleQueryType = iota
	RuleQueryUpdate
	RuleQueryFind
	RuleQueryRemove
	RuleQueryCount
)

const (
	UserStateAnonymous         = "anonymous"
	UserStateLoggedIn          = "logged_in"
	UserStateWaitSignUpConfirm = "sign_up"
	UserStatePreRegistered     = "pre_register"
	UserStateDisabled          = "disabled"
)

type RulesManager struct {
	rules sync.Map // map[proto.DatabaseID]*Rules
}

func (m *RulesManager) Get(dbID proto.DatabaseID) *Rules {
	if v, ok := m.rules.Load(dbID); ok && v != nil {
		return v.(*Rules)
	}

	return nil
}

func (m *RulesManager) Set(dbID proto.DatabaseID, rules *Rules) {
	m.rules.Store(dbID, rules)
}

// use various helper types
type enforceObject = map[string]interface{}
type queryEnforces = map[string]enforceObject // first dim is group/user/default def, second dim is enforce desc
type updateQueryEnforces struct {
	Filter queryEnforces `json:"filter"`
	Update queryEnforces `json:"update"`
}
type tableEnforces struct {
	Find   queryEnforces       `json:"find"`
	Count  queryEnforces       `json:"count"`
	Remove queryEnforces       `json:"remove"`
	Update updateQueryEnforces `json:"update"`
	Insert queryEnforces       `json:"insert"`
}

type RulesConfig struct {
	Groups map[string][]string      `json:"groups" validate:"omitempty,dive,keys,required,endkeys,required,dive,required"`
	Rules  map[string]tableEnforces `json:"rules" validate:"omitempty,dive,keys,required,endkeys,required"`
}

type Rules struct {
	groups     []string
	userGroups map[string][]string
	rules      map[string]*TableRules
}

type TableRules struct {
	rules       map[RuleQueryType]*QueryRules
	updateRules *QueryRules
}

type QueryRules struct {
	groupRules     map[string]map[string]interface{}
	userRules      map[string]map[string]interface{}
	userStateRules map[string]map[string]interface{}
	defaultRules   map[string]interface{} // worked as deny all, allow all
}

type updateMergeItem struct {
	op       string
	argument interface{}
}

func CompileRawRules(rules json.RawMessage) (r *Rules, err error) {
	var cfg *RulesConfig
	err = json.Unmarshal(rules, &cfg)
	if err != nil || cfg == nil {
		return
	}

	err = validator.New().Struct(*cfg)
	if err != nil {
		return
	}

	// compile rules config to rules
	r = &Rules{
		userGroups: make(map[string][]string),
	}

	for groupName, userNames := range cfg.Groups {
		for _, userName := range userNames {
			r.groups = append(r.groups, groupName)
			r.userGroups[userName] = append(r.userGroups[userName], groupName)
		}
	}

	for tableName, tableEnforces := range cfg.Rules {
		tableRules := &TableRules{
			rules: make(map[RuleQueryType]*QueryRules),
		}

		tableRules.rules[RuleQueryFind], err = compileQueryEnforces(cfg, tableEnforces.Find)
		if err != nil {
			return
		}
		tableRules.rules[RuleQueryCount], err = compileQueryEnforces(cfg, tableEnforces.Count)
		if err != nil {
			return
		}
		tableRules.rules[RuleQueryRemove], err = compileQueryEnforces(cfg, tableEnforces.Remove)
		if err != nil {
			return
		}
		tableRules.rules[RuleQueryInsert], err = compileQueryEnforces(cfg, tableEnforces.Insert)
		if err != nil {
			return
		}
		tableRules.rules[RuleQueryUpdate], err = compileQueryEnforces(cfg, tableEnforces.Update.Filter)
		if err != nil {
			return
		}
		tableRules.updateRules, err = compileQueryEnforces(cfg, tableEnforces.Update.Update)
		if err != nil {
			return
		}

		err = validateUpdateRules(tableRules.updateRules)
		if err != nil {
			return
		}

		r.rules[tableName] = tableRules
	}

	return
}

func CompileRules(rules map[string]interface{}) (r *Rules, err error) {
	rulesCfg, err := json.Marshal(rules)
	if err != nil {
		return
	}

	return CompileRawRules(json.RawMessage(rulesCfg))
}

func compileQueryEnforces(cfg *RulesConfig, enforces queryEnforces) (queryRules *QueryRules, err error) {
	queryRules = &QueryRules{
		groupRules:   make(map[string]map[string]interface{}),
		userRules:    make(map[string]map[string]interface{}),
		defaultRules: make(map[string]interface{}),
	}

	for enforceSubject, enforceObject := range enforces {
		switch {
		case strings.HasPrefix(enforceSubject, "g:"):
			groupName := enforceSubject[2:]

			if groupName == "" {
				err = errors.New("invalid empty group name")
				return
			}

			if _, ok := cfg.Groups[groupName]; !ok {
				// invalid group
				err = errors.Errorf("%s: unknown group", groupName)
				return
			}

			queryRules.groupRules[groupName] = enforceObject
		case strings.HasPrefix(enforceSubject, "u:"):
			userName := enforceSubject[2:]

			if userName == "" {
				err = errors.New("invalid empty user name")
				return
			}

			queryRules.userRules[userName] = enforceObject
		case strings.HasPrefix(enforceSubject, "s:"):
			userState := strings.ToLower(enforceSubject[2:])

			switch userState {
			case UserStateAnonymous:
			case UserStateLoggedIn:
			case UserStateWaitSignUpConfirm:
			case UserStatePreRegistered:
			case UserStateDisabled:
			default:
				err = errors.Errorf("invalid user state %s", userState)
				return
			}

			queryRules.userStateRules[userState] = enforceObject
		case enforceSubject == "default":
			queryRules.defaultRules = enforceObject
		default:
			// invalid enforce type
			err = errors.Errorf("%s: invalid enforce type", enforceSubject)
			return
		}
	}

	return
}

func (r *Rules) EnforceRulesOnFilter(f map[string]interface{}, table string,
	uid string, userState string, vars map[string]interface{}, qt RuleQueryType) (
	filter map[string]interface{}, err error) {
	resultRules, err := r.findRulesToApply(r.findUserRules(table, RuleQueryUpdate), uid, userState)
	if err != nil {
		return
	}

	if resultRules == nil {
		filter = f
		return
	}

	var resultAndSubExpr []map[string]interface{}

	for _, r := range resultRules {
		var singleRule map[string]interface{}

		singleRule, err = InjectMagicVars(r, vars)
		if err != nil {
			return
		}

		resultAndSubExpr = append(resultAndSubExpr, singleRule)
	}

	resultAndSubExpr = append(resultAndSubExpr, f)

	filter = map[string]interface{}{
		"$and": resultAndSubExpr,
	}

	return
}

func (r *Rules) EnforceRulesOnUpdate(d map[string]interface{}, table string,
	uid string, userState string, vars map[string]interface{}) (update map[string]interface{}, err error) {
	var (
		tableRules *TableRules
		ok         bool
	)
	if tableRules, ok = r.rules[table]; !ok || tableRules == nil || tableRules.updateRules == nil {
		update = d
		return
	}

	resultRules, err := r.findRulesToApply(tableRules.updateRules, uid, userState)
	if err != nil {
		return
	}

	update, err = mergeUpdate(resultRules...)
	if err != nil {
		return
	}

	update, err = InjectMagicVars(update, vars)
	if err != nil {
		return
	}

	update, err = mergeUpdate(d, update)

	return
}

func (r *Rules) EnforceRulesOnInsert(d map[string]interface{}, table string,
	uid string, userState string, vars map[string]interface{}) (insert map[string]interface{}, err error) {
	resultRules, err := r.findRulesToApply(r.findUserRules(table, RuleQueryInsert), uid, userState)
	if err != nil {
		return
	}

	if resultRules == nil {
		insert = d
		return
	}

	insert, err = InjectMagicVars(mergeInsert(resultRules...), vars)
	if err != nil {
		return
	}

	// merge inserts vars to original query
	insert = mergeInsert(d, insert)

	return
}

func (r *Rules) findUserRules(table string, qt RuleQueryType) (queryRules *QueryRules) {
	var (
		tableRules *TableRules
		ok         bool
	)

	if tableRules, ok = r.rules[table]; !ok || tableRules == nil {
		// open privilege
		return
	}

	if queryRules, ok = tableRules.rules[qt]; !ok || queryRules == nil {
		// open privilege
		return
	}

	return
}

func (r *Rules) findRulesToApply(queryRules *QueryRules, uid string, userState string) (
	resultRules []map[string]interface{}, err error) {
	var (
		groups = r.userGroups[uid]
		ok     bool
	)

	for _, g := range groups {
		var rule map[string]interface{}

		if rule, ok = queryRules.groupRules[g]; !ok {
			continue
		} else if rule == nil {
			err = errors.Errorf("permission denied of be in group %s", g)
			return
		}

		resultRules = append(resultRules, queryRules.groupRules[g])
	}

	var rule map[string]interface{}
	if rule, ok = queryRules.userRules[uid]; !ok {
		// open privilege
	} else if rule == nil {
		err = errors.New("permission denied of user rule")
		return
	}

	resultRules = append(resultRules, rule)

	if rule, ok = queryRules.userStateRules[userState]; !ok {
		// open privilege
	} else if rule == nil {
		err = errors.Errorf("permission denied of user state %s", userState)
		return
	}

	resultRules = append(resultRules, rule)

	// nothing yet founded, apply to default rules
	if len(resultRules) == 0 {
		if queryRules.defaultRules == nil {
			err = errors.New("permission denied of default rule")
			return
		}

		resultRules = append(resultRules, queryRules.defaultRules)
	}

	return
}

func validateUpdateRules(rules *QueryRules) (err error) {
	if rules == nil {
		return
	}

	for _, stateRules := range rules.userStateRules {
		_, err = mergeUpdate(stateRules)
		if err != nil {
			return
		}
	}

	for _, groupRules := range rules.groupRules {
		_, err = mergeUpdate(groupRules)
		if err != nil {
			return
		}
	}

	for _, userRules := range rules.userRules {
		_, err = mergeUpdate(userRules)
		if err != nil {
			return
		}
	}

	_, err = mergeUpdate(rules.defaultRules)

	return
}

func mergeInsert(q ...map[string]interface{}) (o map[string]interface{}) {
	if len(q) == 0 {
		return
	}
	o = make(map[string]interface{}, len(q)*len(q[0]))

	for _, d := range q {
		for k, v := range d {
			o[k] = v
		}
	}

	return
}

func mergeUpdate(q ...map[string]interface{}) (o map[string]interface{}, err error) {
	if len(q) == 0 {
		return
	}

	result := make(map[string]*updateMergeItem) // field, operator, argument

	for _, d := range q {
		var (
			useDollarOp  bool
			useNormalSet bool
		)
		for k, v := range d {
			if strings.HasPrefix(k, "$") {
				if useNormalSet {
					err = errors.New("could not use both normal field update and $ prefixed ops")
					return
				}
				useDollarOp = true
			} else {
				if useDollarOp {
					err = errors.New("could not use both normal field update and $ prefixed ops")
					return
				}
				useNormalSet = true
			}

			if useDollarOp {
				switch {
				case k == "$currentDate" || k == "$inc" || k == "$max" || k == "$min" || k == "$mul" || k == "$set":
					if ov, ok := v.(map[string]interface{}); !ok {
						err = errors.Errorf("$ operator needs object argument")
						return
					} else {
						for field, argument := range ov {
							result[field] = &updateMergeItem{
								op:       k,
								argument: argument,
							}
						}
					}
				case k == "$comment":
					// ignore
				case strings.HasPrefix(k, "$"):
					err = errors.Errorf("invalid operator %s", k)
					return
				}
			} else if useNormalSet {
				result[k] = &updateMergeItem{
					op:       "$set",
					argument: v,
				}
			}
		}
	}

	// convert result to query object
	o = make(map[string]interface{}, 6) // $currentDate/$inc/$max/$min/$mul/$set

	for field, item := range result {
		var (
			opArgs interface{}
			ok     bool
		)
		if opArgs, ok = o[item.op]; !ok {
			opArgs = make(map[string]interface{})
		}
		ov := opArgs.(map[string]interface{})
		ov[field] = item.argument
	}

	return
}
