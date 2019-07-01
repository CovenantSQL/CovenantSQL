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

package api

import "github.com/pkg/errors"

var (
	ErrTokenApplyDisabled             = errors.New("ERR_TOKEN_APPLY_DISABLED")
	ErrNoMainAccount                  = errors.New("ERR_NO_MAIN_ACCOUNT")
	ErrTokenApplyLimitExceeded        = errors.New("ERR_TOKEN_APPLY_LIMIT_EXCEEDED")
	ErrCreateTaskFailed               = errors.New("ERR_CREATE_TASK_FAILED")
	ErrInvalidDeveloper               = errors.New("ERR_INVALID_DEVELOPER")
	ErrGetAccountFailed               = errors.New("ERR_GET_ACCOUNT_INFO_FAILED")
	ErrGetDeveloperFailed             = errors.New("ERR_GET_DEVELOPER_INFO_FAILED")
	ErrParseAccountFailed             = errors.New("ERR_PARSE_ACCOUNT_FAILED")
	ErrSendETLSRPCFailed              = errors.New("ERR_SEND_ETLS_RPC_FAILED")
	ErrSetMainAccountFailed           = errors.New("ERR_SET_MAIN_ACCOUNT_FAILED")
	ErrUpdateDeveloperAccount         = errors.New("ERR_UPDATE_DEVELOPER_ACCOUNT_FAILED")
	ErrCreateSessionFailed            = errors.New("ERR_CREATE_SESSION_FAILED")
	ErrNotAuthorizedAdmin             = errors.New("ERR_NOT_AUTHORIZED_ADMIN")
	ErrInvalidTxHash                  = errors.New("ERR_INVALID_TX_HASH")
	ErrWaitTxConfirmationTimeout      = errors.New("ERR_WAIT_TX_CONFIRMATION_TIMEOUT")
	ErrGenerateKeypairFailed          = errors.New("ERR_GENERATE_KEYPAIR_FAILED")
	ErrEncodePrivateKeyFailed         = errors.New("ERR_ENCODE_PRIVATE_KEY")
	ErrInvalidPrivateKeyUploaded      = errors.New("ERR_INVALID_PRIVATE_KEY_UPLOADED")
	ErrSavePrivateKeyFailed           = errors.New("ERR_SAVE_PRIVATE_KEY_FAILED")
	ErrDeletePrivateKeyFailed         = errors.New("ERR_DELETE_PRIVATE_KEY_FAILED")
	ErrUnbindMainAccountFailed        = errors.New("ERR_UNBIND_MAIN_ACCOUNT_FAILED")
	ErrGetProjectsFailed              = errors.New("ERR_GET_PROJECTS_FAILED")
	ErrLoadProjectDatabaseFailed      = errors.New("ERR_LOAD_PROJECT_DATABASE_FAILED")
	ErrGetUserListFailed              = errors.New("ERR_GET_USER_LIST_FAILED")
	ErrPreRegisterUserFailed          = errors.New("ERR_PRE_REGISTER_USER_FAILED")
	ErrPreRegisterUserAlreadyExists   = errors.New("ERR_PRE_REGISTER_USER_EXISTS")
	ErrGetProjectUserFailed           = errors.New("ERR_GET_PROJECT_USER_FAILED")
	ErrGetProjectUserStateFailed      = errors.New("ERR_GET_PROJECT_USER_STATE_FAILED")
	ErrUpdateUserFailed               = errors.New("ERR_UPDATE_USER_FAILED")
	ErrUpdateUserConflictWithExisting = errors.New("ERR_UPDATE_USER_CONFLICT_WITH_EXISTING")
	ErrGetProjectFailed               = errors.New("ERR_GET_PROJECT_FAILED")
	ErrNoPublicServiceHosts           = errors.New("ERR_NO_PUBLIC_SERVICE_HOSTS")
	ErrIncompleteOAuthConfig          = errors.New("ERR_INCOMPLETE_OAUTH_CONFIG")
	ErrAddProjectOAuthConfigFailed    = errors.New("ERR_ADD_PROJECT_OAUTH_CONFIG_FAILED")
	ErrUpdateProjectConfigFailed      = errors.New("ERR_UPDATE_PROJECT_CONFIG_FAILED")
	ErrGetProjectRulesFailed          = errors.New("ERR_GET_PROJECT_RULES_FAILED")
	ErrPopulateProjectRulesFailed     = errors.New("ERR_POPULATE_PROJECT_RULES_FAILED")
	ErrSetProjectAliasFailed          = errors.New("ERR_SET_PROJECT_ALIAS_FAILED")
	ErrAddProjectMiscConfigFailed     = errors.New("ERR_ADD_PROJECT_MISC_CONFIG_FAILED")
	ErrGetProjectConfigFailed         = errors.New("ERR_GET_PROJECT_CONFIG_FAILED")
	ErrGetProjectTableNamesFailed     = errors.New("ERR_GET_PROJECT_TABLE_NAMES_FAILED")
	ErrMismatchedColumnNameAndTypes   = errors.New("ERR_MISMATCHED_COLUMN_NAMES_AND_TYPES")
	ErrReservedTableName              = errors.New("ERR_RESERVED_TABLE_NAME")
	ErrInvalidAutoIncrementColumnType = errors.New("ERR_INVALID_AUTO_INCREMENT_COLUMN_TYPE")
	ErrUnknownPrimaryKeyColumn        = errors.New("ERR_UNKNOWN_PRIVATE_KEY_COLUMN")
	ErrDDLExecuteFailed               = errors.New("ERR_DDL_EXECUTE_FAILED")
	ErrAddProjectTableConfigFailed    = errors.New("ERR_ADD_PROJECT_TABLE_CONFIG_FAILED")
	ErrGetProjectTableConfigFailed    = errors.New("ERR_GET_PROJECT_TABLE_CONFIG_FAILED")
	ErrGetProjectTableDDLFailed       = errors.New("ERR_GET_PROJECT_TABLE_DDL_FAILED")
	ErrColumnAlreadyExists            = errors.New("ERR_COLUMN_ALREADY_EXISTS")
	ErrTableNotExists                 = errors.New("ERR_TABLE_NOT_EXISTS")
	ErrGetTaskListFailed              = errors.New("ERR_GET_TASK_LIST_FAILED")
	ErrGetTaskDetailFailed            = errors.New("ERR_GET_TASK_DETAIL_FAILED")
	ErrPrepareExecutionContextFailed  = errors.New("ERR_PREPARE_EXECUTION_CONTEXT_FAILED")
	ErrEnforceRuleOnQueryFailed       = errors.New("ERR_ENFORCE_RULES_ON_QUERY_FAILED")
	ErrExecuteQueryFailed             = errors.New("ERR_EXECUTE_QUERY_FAILED")
	ErrScanRowsFailed                 = errors.New("ERR_SCAN_ROWS_FAILED")
	ErrNotAuthorizedUser              = errors.New("ERR_NOT_AUTHORIZED_USER")
)
