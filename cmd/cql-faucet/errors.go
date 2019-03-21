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

package main

import "errors"

var (
	// user errors

	// ErrInvalidAccount represents account is not a valid account.
	ErrInvalidAccount = errors.New("INVALID_ADDRESS")
	// ErrInvalidDatabase represents database id is not valid.
	ErrInvalidDatabase = errors.New("INVALID_DATABASE")
	// ErrAccountQuotaExceeded represents the applicant has exceeded the account daily applyToken quota.
	ErrAccountQuotaExceeded = errors.New("ACCOUNT_QUOTA_EXCEEDED")
	// ErrEmailQuotaExceeded represents the applicant has exceeded the account daily applyToken quota.
	ErrEmailQuotaExceeded = errors.New("EMAIL_QUOTA_EXCEEDED")
	// ErrEnqueueApplication represents failing to enqueue the applyToken request.
	ErrEnqueueApplication = errors.New("ADD_RECORD_FAILED")

	// system errors

	// ErrInvalidFaucetConfig represents invalid faucet config without enough configurations.
	ErrInvalidFaucetConfig = errors.New("invalid faucet config")
)
