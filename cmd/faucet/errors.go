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

	// ErrInvalidURL represents the invalid media url error.
	ErrInvalidURL = errors.New("INVALID_URL")
	// ErrInvalidAddress represents address is not a valid test net address.
	ErrInvalidAddress = errors.New("INVALID_ADDRESS")
	// ErrInvalidApplicationID represents the application id provided is invalid.
	ErrInvalidApplicationID = errors.New("INVALID_APPLICATION_ID")
	// ErrAccountQuotaExceeded represents the applicant has exceeded the account daily application quota.
	ErrAccountQuotaExceeded = errors.New("ACCOUNT_QUOTA_EXCEEDED")
	// ErrAddressQuotaExceeded represents the applicant has exceeded the address daily application quota.
	ErrAddressQuotaExceeded = errors.New("ADDRESS_QUOTA_EXCEEDED")
	// ErrEnqueueApplication represents failing to enqueue the application request.
	ErrEnqueueApplication = errors.New("ENQUEUE_FAILED")
	// ErrRequiredContentNotExists represents invalid application which contains no advertising content.
	ErrRequiredContentNotExists = errors.New("NO_REQUIRED_CONTENT")
	// ErrRequiredURLNotExists represents invalid application which contains no advertising url.
	ErrRequiredURLNotExists = errors.New("NO_REQUIRED_LINK")

	// system errors

	// ErrInvalidFaucetConfig represents invalid faucet config without enough configurations.
	ErrInvalidFaucetConfig = errors.New("invalid faucet config")
)
