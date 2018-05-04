/*
 * MIT License
 *
 * Copyright (c) 2016-2018. ThunderDB
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

// Package hash provides abstracted hash functionality.
//
// This package provides a generic hash type and associated functions that
// allows the specific hash algorithm to be abstracted.
//
// Q: WHY SHA-256 twice?
//
// A:	SHA-256(SHA-256(x)) was proposed by Ferguson and Schneier in their excellent
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
//                                --From: https://crypto.stackexchange.com/a/884
package hash
