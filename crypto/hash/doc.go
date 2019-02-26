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

// Package hash provides abstracted hash functionality.
//
// This package provides a generic hash type and associated functions that
// allows the specific hash algorithm to be abstracted.
//
// Q: WHY SHA-256 twice?
//
// A: SHA-256(SHA-256(x)) was proposed by Ferguson and Schneier in their excellent
// book "Practical Cryptography" (later updated by Ferguson, Schneier, and Kohno
// and renamed "Cryptography Engineering") as a way to make SHA-256 invulnerable
// to "length-extension" attack. They called it "SHA-256d". We started using
// SHA-256d for everything when we launched the Tahoe-LAFS project in 2006, on
// the principle that it is hardly less efficient than SHA-256, and that it
// frees us from having to reason about whether length-extension attacks are
// dangerous every place that we use a hash function. I wouldn't be surprised
// if the inventors of Bitcoin used it for similar reasons. Why not use SHA-256d
// instead of SHA-256?
//
// Note that the SHA-3 project required all candidates to have some method of
// preventing length-extension attacks. Some of them use a method that is rather
// like SHA-256d, i.e. they do an extra "finalization" hash of their state at the
// end, before emitting a result.
//                                --From: https://crypto.stackexchange.com/a/884.
package hash
