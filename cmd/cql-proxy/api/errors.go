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
	// ErrTokenApplyDisabled defines token application disabled error.
	ErrTokenApplyDisabled = errors.New("ERR_TOKEN_APPLY_DISABLED")
	// ErrNoMainAccount defines no main account is set by developer error.
	ErrNoMainAccount = errors.New("ERR_NO_MAIN_ACCOUNT")
	// ErrTokenApplyLimitExceeded defines daily limits for token application is exceeded by developer.
	ErrTokenApplyLimitExceeded = errors.New("ERR_TOKEN_APPLY_LIMIT_EXCEEDED")
	// ErrCreateTaskFailed defines failure to create new task for async process.
	ErrCreateTaskFailed = errors.New("ERR_CREATE_TASK_FAILED")
	// ErrInvalidDeveloper defines invalid developer for next process.
	ErrInvalidDeveloper = errors.New("ERR_INVALID_DEVELOPER")
	// ErrGetAccountFailed defines get account/keypair info object failure.
	ErrGetAccountFailed = errors.New("ERR_GET_ACCOUNT_INFO_FAILED")
	// ErrGetDeveloperFailed defines get developer info object failure.
	ErrGetDeveloperFailed = errors.New("ERR_GET_DEVELOPER_INFO_FAILED")
	// ErrParseAccountFailed defines error for account parsing.
	ErrParseAccountFailed = errors.New("ERR_PARSE_ACCOUNT_FAILED")
	// ErrSendETLSRPCFailed defines error for sending etls rpc.
	ErrSendETLSRPCFailed = errors.New("ERR_SEND_ETLS_RPC_FAILED")
	// ErrSetMainAccountFailed defines error for setting keypair as new developer main account.
	ErrSetMainAccountFailed = errors.New("ERR_SET_MAIN_ACCOUNT_FAILED")
	// ErrUpdateDeveloperAccount defines error on update keypair info.
	ErrUpdateDeveloperAccount = errors.New("ERR_UPDATE_DEVELOPER_ACCOUNT_FAILED")
	// ErrCreateSessionFailed defines error on user/admin login session creation.
	ErrCreateSessionFailed = errors.New("ERR_CREATE_SESSION_FAILED")
	// ErrNotAuthorizedAdmin defines error on unauthorized admin-only api access.
	ErrNotAuthorizedAdmin = errors.New("ERR_NOT_AUTHORIZED_ADMIN")
	// ErrInvalidTxHash defines malformed hash string.
	ErrInvalidTxHash = errors.New("ERR_INVALID_TX_HASH")
	// ErrWaitTxConfirmationTimeout defines timeout error for tx completion.
	ErrWaitTxConfirmationTimeout = errors.New("ERR_WAIT_TX_CONFIRMATION_TIMEOUT")
	// ErrGenerateKeyPairFailed defines error for generating new keypair.
	ErrGenerateKeyPairFailed = errors.New("ERR_GENERATE_KEYPAIR_FAILED")
	// ErrEncodePrivateKeyFailed defines error for encode private key to bytes.
	ErrEncodePrivateKeyFailed = errors.New("ERR_ENCODE_PRIVATE_KEY")
	// ErrInvalidPrivateKeyUploaded defines error for invalid private key being uploaded.
	ErrInvalidPrivateKeyUploaded = errors.New("ERR_INVALID_PRIVATE_KEY_UPLOADED")
	// ErrSavePrivateKeyFailed defines error on saving private key to proxy database.
	ErrSavePrivateKeyFailed = errors.New("ERR_SAVE_PRIVATE_KEY_FAILED")
	// ErrDeletePrivateKeyFailed defines error on deleting private key record.
	ErrDeletePrivateKeyFailed = errors.New("ERR_DELETE_PRIVATE_KEY_FAILED")
	// ErrUnbindMainAccountFailed defines error on unbinding keypair as main account of developer.
	ErrUnbindMainAccountFailed = errors.New("ERR_UNBIND_MAIN_ACCOUNT_FAILED")
	// ErrGetProjectsFailed defines error on fetching all available projects.
	ErrGetProjectsFailed = errors.New("ERR_GET_PROJECTS_FAILED")
	// ErrLoadProjectDatabaseFailed defines error on build project database access object.
	ErrLoadProjectDatabaseFailed = errors.New("ERR_LOAD_PROJECT_DATABASE_FAILED")
	// ErrGetUserListFailed defines error on fetching user list of project.
	ErrGetUserListFailed = errors.New("ERR_GET_USER_LIST_FAILED")
	// ErrPreRegisterUserFailed defines error on pre-registering user.
	ErrPreRegisterUserFailed = errors.New("ERR_PRE_REGISTER_USER_FAILED")
	// ErrPreRegisterUserAlreadyExists defines error on duplicate registering same user.
	ErrPreRegisterUserAlreadyExists = errors.New("ERR_PRE_REGISTER_USER_EXISTS")
	// ErrGetProjectUserFailed defines error on get project user info object.
	ErrGetProjectUserFailed = errors.New("ERR_GET_PROJECT_USER_FAILED")
	// ErrGetProjectUserStateFailed defines error on parsing project user state string.
	ErrGetProjectUserStateFailed = errors.New("ERR_GET_PROJECT_USER_STATE_FAILED")
	// ErrUpdateUserFailed defines error on update project user info object.
	ErrUpdateUserFailed = errors.New("ERR_UPDATE_USER_FAILED")
	// ErrUpdateUserConflictWithExisting defines error on updating user triggers conflict with existing user records (maybe email conflict).
	ErrUpdateUserConflictWithExisting = errors.New("ERR_UPDATE_USER_CONFLICT_WITH_EXISTING")
	// ErrGetProjectFailed defines error on get project object.
	ErrGetProjectFailed = errors.New("ERR_GET_PROJECT_FAILED")
	// ErrNoPublicServiceHosts defines no public service host being set in proxy.
	ErrNoPublicServiceHosts = errors.New("ERR_NO_PUBLIC_SERVICE_HOSTS")
	// ErrIncompleteOAuthConfig defines incomplete oauth config being set.
	ErrIncompleteOAuthConfig = errors.New("ERR_INCOMPLETE_OAUTH_CONFIG")
	// ErrAddProjectOAuthConfigFailed defines error on add new project oauth config.
	ErrAddProjectOAuthConfigFailed = errors.New("ERR_ADD_PROJECT_OAUTH_CONFIG_FAILED")
	// ErrUpdateProjectConfigFailed defines error on update project config.
	ErrUpdateProjectConfigFailed = errors.New("ERR_UPDATE_PROJECT_CONFIG_FAILED")
	// ErrGetProjectRulesFailed defines error on get project query enforce rules.
	ErrGetProjectRulesFailed = errors.New("ERR_GET_PROJECT_RULES_FAILED")
	// ErrPopulateProjectRulesFailed defines error on update project query enforce rules in database and take effect.
	ErrPopulateProjectRulesFailed = errors.New("ERR_POPULATE_PROJECT_RULES_FAILED")
	// ErrSetProjectAliasFailed defines error on setting project alias.
	ErrSetProjectAliasFailed = errors.New("ERR_SET_PROJECT_ALIAS_FAILED")
	// ErrAddProjectMiscConfigFailed defines failure on adding project misc config.
	ErrAddProjectMiscConfigFailed = errors.New("ERR_ADD_PROJECT_MISC_CONFIG_FAILED")
	// ErrGetProjectConfigFailed defines error on get project config.
	ErrGetProjectConfigFailed = errors.New("ERR_GET_PROJECT_CONFIG_FAILED")
	// ErrGetProjectTableNamesFailed defines error on getting project table names.
	ErrGetProjectTableNamesFailed = errors.New("ERR_GET_PROJECT_TABLE_NAMES_FAILED")
	// ErrMismatchedColumnNamesAndTypes defines error on getting column name and types of project tables.
	ErrMismatchedColumnNamesAndTypes = errors.New("ERR_MISMATCHED_COLUMN_NAMES_AND_TYPES")
	// ErrReservedTableName defines error on create table with reserved table name.
	ErrReservedTableName = errors.New("ERR_RESERVED_TABLE_NAME")
	// ErrInvalidAutoIncrementColumnType defines error on using invalid column type for auto increment column (only integer type is supported)
	ErrInvalidAutoIncrementColumnType = errors.New("ERR_INVALID_AUTO_INCREMENT_COLUMN_TYPE")
	// ErrUnknownPrimaryKeyColumn defines error on unknown column being set as private key.
	ErrUnknownPrimaryKeyColumn = errors.New("ERR_UNKNOWN_PRIVATE_KEY_COLUMN")
	// ErrDDLExecuteFailed defines error on failure executing ddl statement for create table/alter table.
	ErrDDLExecuteFailed = errors.New("ERR_DDL_EXECUTE_FAILED")
	// ErrAddProjectTableConfigFailed defines error on add project table config object.
	ErrAddProjectTableConfigFailed = errors.New("ERR_ADD_PROJECT_TABLE_CONFIG_FAILED")
	// ErrGetProjectTableConfigFailed defines error on get project table config object.
	ErrGetProjectTableConfigFailed = errors.New("ERR_GET_PROJECT_TABLE_CONFIG_FAILED")
	// ErrGetProjectTableDDLFailed defines error on get project table ddl statement.
	ErrGetProjectTableDDLFailed = errors.New("ERR_GET_PROJECT_TABLE_DDL_FAILED")
	// ErrColumnAlreadyExists defines error on add duplicate columns names to same table.
	ErrColumnAlreadyExists = errors.New("ERR_COLUMN_ALREADY_EXISTS")
	// ErrTableNotExists defines unknown table being requested.
	ErrTableNotExists = errors.New("ERR_TABLE_NOT_EXISTS")
	// ErrGetTaskListFailed define error on fetching task list.
	ErrGetTaskListFailed = errors.New("ERR_GET_TASK_LIST_FAILED")
	// ErrGetTaskDetailFailed defines error on fetching task detail.
	ErrGetTaskDetailFailed = errors.New("ERR_GET_TASK_DETAIL_FAILED")
	// ErrPrepareExecutionContextFailed defines error on build user query execution context (prepare project database access object, rules object, etc.).
	ErrPrepareExecutionContextFailed = errors.New("ERR_PREPARE_EXECUTION_CONTEXT_FAILED")
	// ErrEnforceRuleOnQueryFailed defines error on resolving enforce rules.
	ErrEnforceRuleOnQueryFailed = errors.New("ERR_ENFORCE_RULES_ON_QUERY_FAILED")
	// ErrExecuteQueryFailed defines error on executing query.
	ErrExecuteQueryFailed = errors.New("ERR_EXECUTE_QUERY_FAILED")
	// ErrScanRowsFailed defines error on scanning rows for find query.
	ErrScanRowsFailed = errors.New("ERR_SCAN_ROWS_FAILED")
	// ErrNotAuthorizedUser defines error on unauthorized user api access.
	ErrNotAuthorizedUser = errors.New("ERR_NOT_AUTHORIZED_USER")
	// ErrKeyPairHasRelatedProjects defines error on deleting project related keypair.
	ErrKeyPairHasRelatedProjects = errors.New("ERR_KEYPAIR_HAS_RELATED_PROJECTS")
	// ErrDeleteProjectsFailed defines error on deleting multiple projects.
	ErrDeleteProjectsFailed = errors.New("ERR_DELETE_PROJECTS_FAILED")
	// ErrProjectIsDisabled defines error on accessing disabled project via user data api/user oauth api.
	ErrProjectIsDisabled = errors.New("ERR_PROJECT_IS_DISABLED")
)
