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

package debug

import (
	"encoding/json"
	"net/http"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

func init() {
	http.HandleFunc("/debug/covenantsql/loglevel",
		func(w http.ResponseWriter, req *http.Request) {
			data := map[string]interface{}{}
			switch req.Method {
			case http.MethodPost:
				level := req.FormValue("level")
				data["orig"] = log.GetLevel().String()
				if level != "" {
					data["want"] = level
					lvl, err := log.ParseLevel(level)
					if err != nil {
						data["err"] = err.Error()
					} else {
						// set level
						log.SetLevel(lvl)
					}
				}
				fallthrough
			case http.MethodGet:
				data["level"] = log.GetLevel().String()
				_ = json.NewEncoder(w).Encode(data)
			default:
				w.WriteHeader(http.StatusBadRequest)
			}
		},
	)
}
