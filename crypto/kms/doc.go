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

// Package kms implements Key Management System
// According the best practices from "sections 3.5 and 3.6 of the PCI DSS standard"
// and "ANSI X9.17 - Financial Institution Key Management". we store a Elliptic Curve
// Master Key as the "Key Encrypting Key". The KEK is used to encrypt/decrypt and sign
// the PrivateKey which will be use with ECDH to generate Data Encrypting Key.
package kms
