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
	"strings"
)

func tryGlobMatch(s1, s2 string) bool {
	// only support * globing
	s1 = strings.ToLower(s1)
	s2 = strings.ToLower(s2)
	s1StarIndex := strings.Index(s1, "*")
	s2StarIndex := strings.Index(s2, "*")
	if s1StarIndex == -1 && s2StarIndex == -1 {
		// normal string
		return s1 == s2
	}

	if s1StarIndex != -1 && s2StarIndex != -1 {
		// both contains stars, overlapping could not be checked
		return s1 == s2
	}

	if s1StarIndex == -1 {
		// s2 contains a star character
		return globMatch(s2, s1)
	} else {
		return globMatch(s1, s2)
	}
}

func globMatch(pattern, s string) bool {
	for len(pattern) > 0 {
		var (
			star  bool
			chunk string
		)
		star, chunk, pattern = scanChunk(pattern)
		if star && chunk == "" {
			return true
		}
		if strings.HasPrefix(s, chunk) && (s == chunk || len(pattern) > 0) {
			s = s[len(chunk):]
			continue
		}
		if star {
			if pos := strings.Index(s, chunk); pos != -1 {
				t := s[pos+len(chunk):]
				if len(pattern) == 0 && len(t) > 0 {
					continue
				}
				s = t
				continue
			}
		}
		return false
	}
	return len(s) == 0
}

func scanChunk(pattern string) (star bool, chunk, rest string) {
	for len(pattern) > 0 && pattern[0] == '*' {
		pattern = pattern[1:]
		star = true
	}
	var i int
	for i = 0; i < len(pattern); i++ {
		if pattern[i] == '*' {
			break
		}
	}
	return star, pattern[0:i], pattern[i:]
}
