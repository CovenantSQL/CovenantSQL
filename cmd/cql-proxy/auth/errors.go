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

package auth

import "errors"

var (
	// ErrIncorrectPassword define password incorrect error on non-oauth based admin authorization.
	ErrIncorrectPassword = errors.New("incorrect password")
	// ErrOAuthGetUserFailed defines error on failure to fetch user info for oauth process.
	ErrOAuthGetUserFailed = errors.New("get user failed")
	// ErrUnsupportedUserAuthProvider defines error on currently unsupported oauth user provider.
	ErrUnsupportedUserAuthProvider = errors.New("unsupported user auth provider")
)
