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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"
)

const (
	validConfig = `
SecureGateway:
  ListenAddr: 127.0.0.1:4665
  Auth:
    Users:
      user1: yD7LCvhH
      user2: AUZzFq4X
      user3: jhAWQTnh
    Grants:
      Policies:
        - User: user1
          Field: db1.tbl1.col1
          Action: read
        - User: admin_group
          Field: db1.tbl1.col1
          Action: write
        - User: read_group
          Field: tbl2_fields
          Action: read
        - User: admin_group
          Field: db1.tbl3.col*
          Action: write
      UserGroups:
        admin_group:
          - user2
          - user3
        read_group:
          - user1
          - user2
          - user3
      FieldGroups:
        tbl2_fields:
          - db1.tbl2.col*
    Encryption:
      - Key: key1.key
        Fields: 
          - tbl2_fields
      - Key: key2.key
        Fields:
          - db1.tbl3.col*
`
	nilConfig = `
SecureGateway: null
`
	withoutListenAddrConfig = `
SecureGateway:
  Auth:
    Users:
      user1: yD7LCvhH
      user2: AUZzFq4X
      user3: jhAWQTnh
    Grants:
      Policies:
        - User: user1
          Field: db1.tbl1.col1
          Action: read
        - User: admin_group
          Field: db1.tbl1.col1
          Action: write
        - User: read_group
          Field: tbl2_fields
          Action: read
        - User: admin_group
          Field: db1.tbl3.col*
          Action: write
      UserGroups:
        admin_group:
          - user2
          - user3
        read_group:
          - user1
          - user2
          - user3
      FieldGroups:
        tbl2_fields:
          - db1.tbl2.col*
    Encryption:
      - Key: key1.key
        Fields: 
          - tbl2_fields
      - Key: key2.key
        Fields:
          - db1.tbl3.col*
`
	withoutAuthConfig = `
SecureGateway:
  ListenAddr: 127.0.0.1:4665
`
	invalidGrantsConfig = `
SecureGateway:
  ListenAddr: 127.0.0.1:4665
  Auth:
    Users:
      user1: yD7LCvhH
      user2: AUZzFq4X
      user3: jhAWQTnh
    Grants:
      Policies:
        - User: user1
          Field: "invalid field"
          Action: read
        - User: admin_group
          Field: db1.tbl1.col1
          Action: write
        - User: read_group
          Field: tbl2_fields
          Action: read
        - User: admin_group
          Field: db1.tbl3.col*
          Action: write
      UserGroups:
        admin_group:
          - user2
          - user3
        read_group:
          - user1
          - user2
          - user3
      FieldGroups:
        tbl2_fields:
          - db1.tbl2.col*
    Encryption:
      - Key: key1.key
        Fields: 
          - tbl2_fields
      - Key: key2.key
        Fields:
          - db1.tbl3.col*
`
	overlappedFieldsEncryptionConfig = `
SecureGateway:
  ListenAddr: 127.0.0.1:4665
  Auth:
    Users:
      user1: yD7LCvhH
      user2: AUZzFq4X
      user3: jhAWQTnh
    Grants:
      Policies:
        - User: user1
          Field: db1.tbl1.col1
          Action: read
        - User: admin_group
          Field: db1.tbl1.col1
          Action: write
        - User: read_group
          Field: tbl2_fields
          Action: read
        - User: admin_group
          Field: db1.tbl3.col*
          Action: write
      UserGroups:
        admin_group:
          - user2
          - user3
        read_group:
          - user1
          - user2
          - user3
      FieldGroups:
        tbl2_fields:
          - db1.tbl2.col*
    Encryption:
      - Key: key1.key
        Fields: 
          - tbl2_fields
          - db1.tbl3.col1
      - Key: key2.key
        Fields:
          - db1.tbl3.col*
`
	anotherOverlappedFieldsEncryptionConfig = `
SecureGateway:
  ListenAddr: 127.0.0.1:4665
  Auth:
    Users:
      user1: yD7LCvhH
      user2: AUZzFq4X
      user3: jhAWQTnh
    Grants:
      Policies:
        - User: user1
          Field: db1.tbl1.col1
          Action: read
        - User: admin_group
          Field: db1.tbl1.col1
          Action: write
        - User: read_group
          Field: tbl2_fields
          Action: read
        - User: admin_group
          Field: db1.tbl3.col*
          Action: write
      UserGroups:
        admin_group:
          - user2
          - user3
        read_group:
          - user1
          - user2
          - user3
      FieldGroups:
        tbl2_fields:
          - db1.tbl2.col*
    Encryption:
      - Key: key2.key
        Fields:
          - db1.tbl2.col2
      - Key: key1.key
        Fields: 
          - tbl2_fields
`
	invalidEncryptionField = `
SecureGateway:
  ListenAddr: 127.0.0.1:4665
  Auth:
    Users:
      user1: yD7LCvhH
      user2: AUZzFq4X
      user3: jhAWQTnh
    Grants:
      Policies:
        - User: user1
          Field: db1.tbl1.col1
          Action: read
        - User: admin_group
          Field: db1.tbl1.col1
          Action: write
        - User: read_group
          Field: tbl2_fields
          Action: read
        - User: admin_group
          Field: db1.tbl3.col*
          Action: write
      UserGroups:
        admin_group:
          - user2
          - user3
        read_group:
          - user1
          - user2
          - user3
      FieldGroups:
        tbl2_fields:
          - db1.tbl2.col*
    Encryption:
      - Key: key1.key
        Fields: 
          - tbl2_fields
      - Key: key2.key
        Fields:
          - db1.tbl3.col*
          - invalid_field
`
	anotherInvalidEncryptionField = `
SecureGateway:
  ListenAddr: 127.0.0.1:4665
  Auth:
    Users:
      user1: yD7LCvhH
      user2: AUZzFq4X
      user3: jhAWQTnh
    Grants:
      Policies:
        - User: user1
          Field: db1.tbl1.col1
          Action: read
        - User: admin_group
          Field: db1.tbl1.col1
          Action: write
        - User: read_group
          Field: tbl2_fields
          Action: read
        - User: admin_group
          Field: db1.tbl3.col*
          Action: write
      UserGroups:
        admin_group:
          - user2
          - user3
        read_group:
          - user1
          - user2
          - user3
      FieldGroups:
        tbl2_fields:
          - db1.tbl2.col*
    Encryption:
      - Key: key1.key
        Fields: 
          - tbl2_fields
      - Key: key2.key
        Fields:
          - db1.tbl3.col*
          - db1..col1
`
	nonExistsKeyConfig = `
SecureGateway:
  ListenAddr: 127.0.0.1:4665
  Auth:
    Users:
      user1: yD7LCvhH
      user2: AUZzFq4X
      user3: jhAWQTnh
    Grants:
      Policies:
        - User: user1
          Field: db1.tbl1.col1
          Action: read
        - User: admin_group
          Field: db1.tbl1.col1
          Action: write
        - User: read_group
          Field: tbl2_fields
          Action: read
        - User: admin_group
          Field: db1.tbl3.col*
          Action: write
      UserGroups:
        admin_group:
          - user2
          - user3
        read_group:
          - user1
          - user2
          - user3
      FieldGroups:
        tbl2_fields:
          - db1.tbl2.col*
    Encryption:
      - Key: key1.key
        Fields: 
          - tbl2_fields
      - Key: key2.key
        Fields:
          - db1.tbl3.col*
      - Key: key3.key
        Fields:
          - db1.tbl4.col1
`
)

func generateKey(path string) (err error) {
	var privKey *asymmetric.PrivateKey
	if privKey, _, err = asymmetric.GenSecp256k1KeyPair(); err != nil {
		return
	}
	err = kms.SavePrivateKey(path, privKey, []byte{})
	return
}

func TestLoadConfig(t *testing.T) {
	log.SetLevel(log.FatalLevel)
	Convey("test load config", t, func() {
		// set global config
		var dirName string
		var err error
		dirName, err = ioutil.TempDir("", "sg_test")
		So(err, ShouldBeNil)
		defer os.RemoveAll(dirName)
		conf.GConf = &conf.Config{
			WorkingRoot: dirName,
		}
		withDirPrefix := func(p string) string {
			return filepath.Join(dirName, p)
		}

		// load non-exist config file
		_, err = loadConfig("non-exists")
		So(err, ShouldNotBeNil)

		// load invalid yaml config
		err = ioutil.WriteFile(withDirPrefix("invalid.yaml"), []byte("haha"), 0600)
		So(err, ShouldBeNil)
		_, err = loadConfig(withDirPrefix("invalid.yaml"))
		So(err, ShouldNotBeNil)

		// generate key for control group
		err = generateKey(withDirPrefix("key1.key"))
		So(err, ShouldBeNil)
		err = generateKey(withDirPrefix("key2.key"))
		So(err, ShouldBeNil)

		// test a correct control group
		err = ioutil.WriteFile(withDirPrefix("valid.yaml"), []byte(validConfig), 0600)
		So(err, ShouldBeNil)
		_, err = loadConfig(withDirPrefix("valid.yaml"))
		So(err, ShouldBeNil)

		// test without valid global config
		var (
			tempConf *conf.Config
			tempDir  string
		)
		tempConf = conf.GConf
		conf.GConf = nil
		tempDir, err = os.Getwd()
		So(err, ShouldBeNil)
		os.Chdir(dirName)
		_, err = loadConfig(withDirPrefix("valid.yaml"))
		So(err, ShouldBeNil)
		conf.GConf = tempConf
		os.Chdir(tempDir)

		// test nil config
		err = ioutil.WriteFile(withDirPrefix("nil.yaml"), []byte(nilConfig), 0600)
		So(err, ShouldBeNil)
		_, err = loadConfig(withDirPrefix("nil.yaml"))
		So(err, ShouldNotBeNil)
		So(errors.Cause(err), ShouldEqual, ErrInvalidConfig)

		// test invalid config without listen addr
		err = ioutil.WriteFile(withDirPrefix("without_listen_addr.yaml"), []byte(withoutListenAddrConfig), 0600)
		So(err, ShouldBeNil)
		_, err = loadConfig(withDirPrefix("without_listen_addr.yaml"))
		So(err, ShouldNotBeNil)
		So(errors.Cause(err), ShouldEqual, ErrInvalidConfig)

		// test invalid config without authentication section
		err = ioutil.WriteFile(withDirPrefix("without_auth.yaml"), []byte(withoutAuthConfig), 0600)
		So(err, ShouldBeNil)
		_, err = loadConfig(withDirPrefix("without_auth.yaml"))
		So(err, ShouldNotBeNil)
		So(errors.Cause(err), ShouldEqual, ErrInvalidConfig)

		// test invalid grants settings (casbin enforcer config)
		err = ioutil.WriteFile(withDirPrefix("invalid_grants.yaml"), []byte(invalidGrantsConfig), 0600)
		So(err, ShouldBeNil)
		_, err = loadConfig(withDirPrefix("invalid_grants.yaml"))
		So(err, ShouldNotBeNil)

		// test overlapped encryption fields config
		err = ioutil.WriteFile(withDirPrefix("overlapped_fields_encryption.yaml"),
			[]byte(overlappedFieldsEncryptionConfig), 0600)
		So(err, ShouldBeNil)
		_, err = loadConfig(withDirPrefix("overlapped_fields_encryption.yaml"))
		So(err, ShouldNotBeNil)
		So(errors.Cause(err), ShouldEqual, ErrFieldEncryption)

		// test another overlapped encryption fields config
		err = ioutil.WriteFile(withDirPrefix("another_overlapped_fields_encryption.yaml"),
			[]byte(anotherOverlappedFieldsEncryptionConfig), 0600)
		So(err, ShouldBeNil)
		_, err = loadConfig(withDirPrefix("another_overlapped_fields_encryption.yaml"))
		So(err, ShouldNotBeNil)
		So(errors.Cause(err), ShouldEqual, ErrFieldEncryption)

		// test invalid encryption field config
		err = ioutil.WriteFile(withDirPrefix("invalid_encryption_field.yaml"),
			[]byte(invalidEncryptionField), 0600)
		So(err, ShouldBeNil)
		_, err = loadConfig(withDirPrefix("invalid_encryption_field.yaml"))
		So(err, ShouldNotBeNil)
		So(errors.Cause(err), ShouldEqual, ErrInvalidField)

		// test invalid encryption field config
		err = ioutil.WriteFile(withDirPrefix("another_invalid_encryption_field.yaml"),
			[]byte(anotherInvalidEncryptionField), 0600)
		So(err, ShouldBeNil)
		_, err = loadConfig(withDirPrefix("another_invalid_encryption_field.yaml"))
		So(err, ShouldNotBeNil)
		So(errors.Cause(err), ShouldEqual, ErrInvalidField)

		// test non-existent key
		err = ioutil.WriteFile(withDirPrefix("non_exists_key.yaml"),
			[]byte(nonExistsKeyConfig), 0600)
		So(err, ShouldBeNil)
		_, err = loadConfig(withDirPrefix("non_exists_key.yaml"))
		So(err, ShouldNotBeNil)
	})
}
