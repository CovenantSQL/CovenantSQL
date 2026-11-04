/*
 * Copyright 2026 The CovenantSQL Authors.
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

/*
 * Renames the global symbols of libsecp256k1 to start with cql_, so that a program can link this
 * package together with another copy of libsecp256k1, such as the one in go-ethereum's
 * crypto/secp256k1. Include it before the sources of libsecp256k1. TestLink fails to link while
 * a global symbol is missing here.
 */

#define secp256k1_context_clone cql_secp256k1_context_clone
#define secp256k1_context_create cql_secp256k1_context_create
#define secp256k1_context_destroy cql_secp256k1_context_destroy
#define secp256k1_context_preallocated_clone cql_secp256k1_context_preallocated_clone
#define secp256k1_context_preallocated_clone_size cql_secp256k1_context_preallocated_clone_size
#define secp256k1_context_preallocated_create cql_secp256k1_context_preallocated_create
#define secp256k1_context_preallocated_destroy cql_secp256k1_context_preallocated_destroy
#define secp256k1_context_preallocated_size cql_secp256k1_context_preallocated_size
#define secp256k1_context_randomize cql_secp256k1_context_randomize
#define secp256k1_context_set_error_callback cql_secp256k1_context_set_error_callback
#define secp256k1_context_set_illegal_callback cql_secp256k1_context_set_illegal_callback
#define secp256k1_context_set_sha256_compression cql_secp256k1_context_set_sha256_compression
#define secp256k1_context_static cql_secp256k1_context_static
#define secp256k1_ec_pubkey_cmp cql_secp256k1_ec_pubkey_cmp
#define secp256k1_ec_pubkey_combine cql_secp256k1_ec_pubkey_combine
#define secp256k1_ec_pubkey_create cql_secp256k1_ec_pubkey_create
#define secp256k1_ec_pubkey_negate cql_secp256k1_ec_pubkey_negate
#define secp256k1_ec_pubkey_parse cql_secp256k1_ec_pubkey_parse
#define secp256k1_ec_pubkey_serialize cql_secp256k1_ec_pubkey_serialize
#define secp256k1_ec_pubkey_sort cql_secp256k1_ec_pubkey_sort
#define secp256k1_ec_pubkey_tweak_add cql_secp256k1_ec_pubkey_tweak_add
#define secp256k1_ec_pubkey_tweak_mul cql_secp256k1_ec_pubkey_tweak_mul
#define secp256k1_ec_seckey_negate cql_secp256k1_ec_seckey_negate
#define secp256k1_ec_seckey_tweak_add cql_secp256k1_ec_seckey_tweak_add
#define secp256k1_ec_seckey_tweak_mul cql_secp256k1_ec_seckey_tweak_mul
#define secp256k1_ec_seckey_verify cql_secp256k1_ec_seckey_verify
#define secp256k1_ecdsa_sign cql_secp256k1_ecdsa_sign
#define secp256k1_ecdsa_signature_normalize cql_secp256k1_ecdsa_signature_normalize
#define secp256k1_ecdsa_signature_parse_compact cql_secp256k1_ecdsa_signature_parse_compact
#define secp256k1_ecdsa_signature_parse_der cql_secp256k1_ecdsa_signature_parse_der
#define secp256k1_ecdsa_signature_serialize_compact cql_secp256k1_ecdsa_signature_serialize_compact
#define secp256k1_ecdsa_signature_serialize_der cql_secp256k1_ecdsa_signature_serialize_der
#define secp256k1_ecdsa_verify cql_secp256k1_ecdsa_verify
#define secp256k1_ecmult_gen_prec_table cql_secp256k1_ecmult_gen_prec_table
#define secp256k1_nonce_function_default cql_secp256k1_nonce_function_default
#define secp256k1_nonce_function_rfc6979 cql_secp256k1_nonce_function_rfc6979
#define secp256k1_pre_g cql_secp256k1_pre_g
#define secp256k1_pre_g_128 cql_secp256k1_pre_g_128
#define secp256k1_selftest cql_secp256k1_selftest
#define secp256k1_tagged_sha256 cql_secp256k1_tagged_sha256
