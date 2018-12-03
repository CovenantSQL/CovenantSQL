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

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/CovenantSQL/CovenantSQL/cmd/cql-mysql-adapter/cursor"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-mysql-adapter/handler"
	cw "github.com/CovenantSQL/CovenantSQL/cmd/cql-secure-gateway/casbin"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/sqlparser"
	"github.com/pkg/errors"
)

type autoDecryptWrapper struct {
	rows          cursor.Rows
	decryptConfig map[int]*encryptionConfig
}

func (w *autoDecryptWrapper) Close() error {
	if w.rows != nil {
		return w.Close()
	}
	return nil
}

func (w *autoDecryptWrapper) Columns() ([]string, error) {
	if w.rows != nil {
		return w.Columns()
	}
	return nil, nil
}

func (w *autoDecryptWrapper) Err() error {
	if w.rows != nil {
		return w.Err()
	}
	return nil
}

func (w *autoDecryptWrapper) Next() bool {
	if w.rows != nil {
		return w.Next()
	}

	return false
}

func (w *autoDecryptWrapper) Scan(dest ...interface{}) (err error) {
	if w.rows != nil {
		// build decrypt array
		var (
			decryptArray = make([]interface{}, len(dest))
			v            *encryptionConfig
			exists       bool
		)

		for i := range dest {
			if v, exists = w.decryptConfig[i]; exists && v != nil {
				byteArray := make([]byte, 0)
				decryptArray[i] = &byteArray
			} else {
				decryptArray[i] = dest[i]
			}
		}

		if err = w.rows.Scan(decryptArray...); err != nil {
			return
		}

		// decrypt
		for i, v := range w.decryptConfig {
			if v == nil {
				continue
			}

			encryptedDataPtr, _ := decryptArray[i].(*[]byte)

			if encryptedDataPtr == nil || len(*encryptedDataPtr) == 0 {
				// nil value
				if err = w.convertAssign(dest[i], nil); err != nil {
					return
				}
			}

			var (
				result        []byte
				encryptedData = *encryptedDataPtr
			)
			if result, err = crypto.DecryptAndCheck(v.Key, encryptedData); err != nil {
				return
			}

			if err = w.convertAssign(dest[i], result); err != nil {
				return
			}
		}
	}

	return
}

func (w *autoDecryptWrapper) convertAssign(dest interface{}, src []byte) (err error) {
	switch d := dest.(type) {
	case *string:
		if d == nil {
			return ErrNilPointer
		}
		*d = string(src)
		return nil

	case *interface{}:
		if d == nil {
			return ErrNilPointer
		}
		*d = w.cloneBytes(src)
		return nil
	case *[]byte:
		if d == nil {
			return ErrNilPointer
		}
		*d = w.cloneBytes(src)
	case *sql.RawBytes:
		if d == nil {
			return ErrNilPointer
		}
		*d = src
		return nil
	case *bool:
		var bv driver.Value
		if bv, err = driver.Bool.ConvertValue(src); err == nil {
			*d = bv.(bool)
		}
		return
	}

	if scanner, ok := dest.(sql.Scanner); ok {
		return scanner.Scan(src)
	}

	dpv := reflect.ValueOf(dest)
	if dpv.Kind() != reflect.Ptr {
		return ErrNotPointer
	}
	if dpv.IsNil() {
		return ErrNilPointer
	}
	sv := reflect.ValueOf(src)

	dv := reflect.Indirect(dpv)
	if sv.IsValid() && sv.Type().AssignableTo(dv.Type()) {
		dv.Set(reflect.ValueOf(w.cloneBytes(src)))
		return nil
	}
	if dv.Kind() == sv.Kind() && sv.Type().ConvertibleTo(dv.Type()) {
		dv.Set(sv.Convert(dv.Type()))
		return nil
	}
	s := string(src)
	switch dv.Kind() {
	case reflect.Ptr:
		if src == nil {
			dv.Set(reflect.Zero(dv.Type()))
			return nil
		}
		dv.Set(reflect.New(dv.Type().Elem()))
		return w.convertAssign(dv.Interface(), src)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var i64 int64
		if i64, err = strconv.ParseInt(s, 10, dv.Type().Bits()); err != nil {
			return
		}
		dv.SetInt(i64)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var u64 uint64
		if u64, err = strconv.ParseUint(s, 10, dv.Type().Bits()); err != nil {
			return
		}
		dv.SetUint(u64)
		return nil
	case reflect.Float32, reflect.Float64:
		var f64 float64
		if f64, err = strconv.ParseFloat(s, dv.Type().Bits()); err != nil {
			return
		}
		dv.SetFloat(f64)
		return nil
	case reflect.String:
		dv.SetString(s)
		return nil
	}

	return ErrUnsupportedType
}

func (w *autoDecryptWrapper) cloneBytes(b []byte) []byte {
	if b == nil {
		return nil
	}
	c := make([]byte, len(b))
	copy(c, b)
	return c
}

// Handler override the default mysql-adapter handler logic and added query privilege logic.
type Handler struct {
	*handler.Handler
	r          *Resolver
	authConfig *authConfig
}

// NewHandler returns the secure gateway handler object.
func NewHandler(authConfig *authConfig) (h *Handler) {
	var r = NewResolver()
	return &Handler{
		Handler:    handler.NewHandlerWithResolver(r.Resolver),
		r:          r,
		authConfig: authConfig,
	}
}

// Resolve resolves the string query to handler query.
func (h *Handler) Resolve(user string, dbID string, query string) (q cursor.Query, err error) {
	if err = h.EnsureDatabase(dbID); err != nil {
		return
	}
	if q, err = h.r.ResolveSingleQuery(dbID, query); err != nil {
		return
	}

	// further check with privilege and encryption
	err = h.ensurePrivilege(user, q)

	return
}

// Query executes the resolved read query with auto fields decryption.
func (h *Handler) Query(q cursor.Query, args ...interface{}) (rows cursor.Rows, err error) {
	if !q.IsRead() {
		err = errors.Wrap(ErrQueryLogicError, "not a read query")
		return
	}
	// execute with auto fields decryption on result rows
	var origRows cursor.Rows
	if origRows, err = h.QueryString(q.GetDatabase(), q.GetQuery(), args...); err != nil {
		return
	}

	// decrypt args
	query := q.(*Query)
	rows = &autoDecryptWrapper{
		rows:          origRows,
		decryptConfig: query.DecryptConfig,
	}
	return
}

// Exec executes the resolved write query with auto fields encryption.
func (h *Handler) Exec(q cursor.Query, args ...interface{}) (res sql.Result, err error) {
	if q.IsRead() {
		err = errors.Wrap(ErrQueryLogicError, "not a write query")
		return
	}
	if q.IsDDL() {
		defer h.r.ReloadMeta()
	}
	query := q.(*Query)
	// execute with auto fields encryption
	for i, v := range query.EncryptConfig {
		if v == nil {
			continue
		}
		// convert argument to byte array
		if len(args) >= i {
			// not enough arguments
			err = errors.Wrapf(ErrQueryLogicError, "not enough arguments, argument %d should be provided", i)
			return
		}

		if args[i] == nil {
			// nil type, not need for encryption
			continue
		}

		// mysql types will be converted to int/float numeric or byte array
		var (
			rawData []byte
			ok      bool
		)
		if rawData, ok = args[i].([]byte); !ok {
			// not byte type, convert to string representation
			sv := reflect.ValueOf(args[i])

			switch sv.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				rawData = []byte(strconv.FormatInt(sv.Int(), sv.Type().Bits()))
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				rawData = []byte(strconv.FormatUint(sv.Uint(), sv.Type().Bits()))
			case reflect.Float32, reflect.Float64:
				rawData = []byte(strconv.FormatFloat(sv.Float(), 'f', -1, sv.Type().Bits()))
			default:
				err = ErrUnsupportedType
				return
			}
		}

		if rawData == nil {
			// set argument to nil
			args[i] = nil
			continue
		}

		if args[i], err = crypto.EncryptAndSign(v.Key.PubKey(), rawData); err != nil {
			return
		}
	}

	// send query
	return h.ExecString(q.GetDatabase(), q.GetQuery(), args...)
}

func (h *Handler) ensurePrivilege(user string, rawQuery cursor.Query) (err error) {
	var (
		query                 *Query
		ok                    bool
		act                   string
		f                     *cw.Field
		hasAuth               bool
		encConfig             *encryptionConfig
		encryptedFields       = map[string]*encryptionConfig{}
		encryptUnsupportedOps = map[string]bool{}
		unauthorizedMessages  []string
		encryptionMessages    []string
	)
	if query, ok = rawQuery.(*Query); !ok {
		err = errors.Wrap(ErrQueryLogicError, "not a valid query")
		return
	}

	if query.IsShow() {
		return
	}

	if query.IsDDL() || query.IsExplain() {
		err = errors.Wrap(ErrUnauthorizedQuery, "ddl or explain query is not permitted")
		return
	}

	// check authority and encryption setting
	for _, cr := range query.PhysicalColumnRely {
		// build field
		if f, err = cw.NewField(query.GetDatabase(), cr.TableName, cr.ColName); err != nil {
			err = errors.Wrap(err, "not a valid field")
			return
		}

		// get encryption config
		if encConfig, err = h.authConfig.GetEncryptionConfig(f); err != nil {
			err = errors.Wrapf(err, "check field encryption failed")
			return
		}

		encryptedFields[f.String()] = encConfig

		// role base access control
		act = cw.ReadAction
		encryptUnsupportedOps = map[string]bool{}

	OperationVisitLoop:
		for _, ops := range cr.Ops {
			for _, op := range ops {
				if encConfig != nil {
					// has encryption config
					switch op {
					case AliasOp, CaseOp, SubQueryOp, DeleteOp, AssignmentOp:
					default:
						encryptUnsupportedOps[string(op)] = true
					}
				}

				if op == AssignmentOp || op == DeleteOp {
					// contains write operation
					act = cw.WriteAction

					// not encrypted field
					if encConfig == nil {
						break OperationVisitLoop
					}
				}
			}
		}

		if hasAuth, err = h.authConfig.Enforcer.EnforceSafe(user, f.String(), act); err != nil {
			err = errors.Wrapf(err, "check auth failure")
			return
		}

		if !hasAuth {
			unauthorizedMessages = append(unauthorizedMessages, fmt.Sprintf("%s:%s", f.String(), act))
		}

		if len(encryptUnsupportedOps) != 0 {
			ops := make([]string, 0, len(encryptUnsupportedOps))
			for op := range encryptUnsupportedOps {
				ops = append(ops, op)
			}
			encryptionMessages = append(encryptionMessages,
				fmt.Sprintf("%s:%s", f.String(), strings.Join(ops, "|")))
		}
	}

	if len(unauthorizedMessages) != 0 || !hasAuth {
		err = errors.Wrapf(ErrUnauthorizedQuery, "fields: %s", strings.Join(unauthorizedMessages, ","))
		return
	}
	if len(encryptionMessages) != 0 {
		err = errors.Wrapf(ErrUnsupportedEncryptionFieldQuery,
			"fields: %s", strings.Join(encryptionMessages, ","))
		return
	}

	// check encryption
	if query.IsRead() {
		// check fields to decrypt, analyzed result columns
		for i, c := range query.PhysicalColumnTransformations {
			// get physical columns to decrypt
			var (
				pc        *Column
				encConfig *encryptionConfig
			)
			if encConfig, pc, err = h.resolveResultColumnDecryptKey(c, query, encryptedFields); err != nil {
				tb := sqlparser.NewTrackedBuffer(nil)
				err = errors.Wrapf(err, "could not resolve column to decrypt: %s", tb.WriteNode(&c.ColName).String())
				return
			}

			if pc != nil {
				query.DecryptConfig[i] = encConfig
			}
		}
	} else {
		// check field binding parameters
		err = h.walkColumns(func(col *Column) (kontinue bool, err error) {
			if col == nil || col.Computation == nil {
				return false, nil
			} else if col.Computation.Op != AssignmentOp {
				return true, nil
			}

			// process assignment
			if len(col.Computation.Operands) != 2 {
				tb := sqlparser.NewTrackedBuffer(nil)
				err = errors.Wrapf(ErrQueryLogicError,
					"should provide bind parameters for encrypted field: %s", tb.WriteNode(&col.ColName).String())
				return
			}

			// first operand should be a physical column
			if !col.Computation.Operands[0].IsPhysical {
				tb := sqlparser.NewTrackedBuffer(nil)
				err = errors.Wrapf(ErrQueryLogicError,
					"query not supported for encryption", tb.WriteNode(&col.ColName).String())
				return
			}

			// second operand should be a bind parameter value
			if col.Computation.Operands[1].Computation == nil || col.Computation.Operands[1].Computation.Op != ParameterOp {
				tb := sqlparser.NewTrackedBuffer(nil)
				err = errors.Wrapf(ErrQueryLogicError,
					"query not supported for encryption", tb.WriteNode(&col.ColName).String())
				return
			}

			var (
				paramVal *sqlparser.SQLVal
				ok       bool
			)

			if paramVal, ok = col.Computation.Operands[1].Computation.Data.(*sqlparser.SQLVal); !ok ||
				paramVal.Type != sqlparser.PosArg && paramVal.Type != sqlparser.ValArg {
				tb := sqlparser.NewTrackedBuffer(nil)
				err = errors.Wrapf(ErrQueryLogicError,
					"column assigned value not supported for encryption", tb.WriteNode(&col.ColName).String())
				return
			}
			if paramVal.Type == sqlparser.ValArg {
				tb := sqlparser.NewTrackedBuffer(nil)
				err = errors.Wrapf(ErrQueryLogicError,
					"column assigned name argument temporary not supported for encryption", tb.WriteNode(&col.ColName).String())
				return
			}

			// find column encryption config
			var (
				f        *cw.Field
				paramKey = string(paramVal.Val)
				paramPos int
			)

			// convert parameter key to parameter position
			if len(paramKey) < 2 {
				tb := sqlparser.NewTrackedBuffer(nil)
				err = errors.Wrapf(ErrQueryLogicError,
					"invalid column value assignment", tb.WriteNode(&col.ColName).String())
				return
			}
			if paramPos, err = strconv.Atoi(paramKey[2:]); err != nil {
				tb := sqlparser.NewTrackedBuffer(nil)
				err = errors.Wrapf(err, "resolve parameter offset for column %s failed", tb.WriteNode(&col.ColName).String())
				return
			}
			if f, err = cw.NewField(query.GetDatabase(), col.ColName.Qualifier.Name.String(), col.ColName.Name.String()); err != nil {
				tb := sqlparser.NewTrackedBuffer(nil)
				err = errors.Wrapf(err, "could not resolve column to encrypt: %s", tb.WriteNode(&col.ColName).String())
				return
			}

			var exists bool
			if query.EncryptConfig[paramPos], exists = encryptedFields[f.String()]; !exists {
				tb := sqlparser.NewTrackedBuffer(nil)
				err = errors.Wrapf(err, "could not resolve column encryption config: %s", tb.WriteNode(&col.ColName).String())
				return
			}

			return false, nil
		}, query.PhysicalColumnTransformations)
		if err != nil {
			err = errors.Wrap(err, "could not resolve column for encryption")
			return
		}
	}

	return
}

func (h *Handler) resolveResultColumnDecryptKey(c *Column, query *Query, encryptionKeys map[string]*encryptionConfig) (encConfig *encryptionConfig, pc *Column, err error) {
	var (
		pcs        = NewColumns()
		encConfigs []*encryptionConfig
		visit      func(col *Column) (kontinue bool, err error)
	)

	visit = func(col *Column) (kontinue bool, err error) {
		if col.Computation == nil {
			if col.IsPhysical {
				// check if the column is encrypted column
				var (
					f             *cw.Field
					tempEncConfig *encryptionConfig
					exists        bool
				)
				if f, err = cw.NewField(query.GetDatabase(), col.ColName.Qualifier.Name.String(),
					col.ColName.Name.String()); err != nil {
					tb := sqlparser.NewTrackedBuffer(nil)
					err = errors.Wrapf(err, "could not resolve column to decrypt: %s", tb.WriteNode(&col.ColName).String())
					return
				}

				if tempEncConfig, exists = encryptionKeys[f.String()]; exists {
					encConfigs = append(encConfigs, tempEncConfig)
					pcs = append(pcs, col)
				}
			}
			return false, nil
		}

		if col.Computation.Op == FilterOp {
			if len(col.Computation.Operands) > 0 {
				// first column is the select expression
				if err = h.walkColumns(visit, NewColumns(col.Computation.Operands[0])); err != nil {
					return
				}
			}

			return false, nil
		}

		return true, nil
	}

	if err = h.walkColumns(visit, NewColumns(c)); err != nil {
		return
	}

	if len(pcs) > 1 {
		// multiple physical columns detected
		err = errors.Wrapf(ErrQueryLogicError, "multiple encrypted columns related to result column")
		return
	}

	if len(pcs) == 0 {
		return
	}

	encConfig = encConfigs[0]
	pc = pcs[0]

	return
}

func (h *Handler) walkColumns(visit func(col *Column) (kontinue bool, err error), cols Columns) (err error) {
	for _, c := range cols {
		if c == nil {
			continue
		}
		var kontinue bool
		if kontinue, err = visit(c); err != nil {
			return
		} else if kontinue {
			if c.Computation != nil && c.Computation.Operands != nil {
				if err = h.walkColumns(visit, c.Computation.Operands); err != nil {
					return
				}
			}
		}
	}
	return nil
}
