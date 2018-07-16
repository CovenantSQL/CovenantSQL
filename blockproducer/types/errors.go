package types

import (
	"errors"
)

var (
	// ErrHashVerification indicates a failed hash verification.
	ErrHashVerification = errors.New("hash verification failed")

	// ErrSignVerification indicates a failed signature verification.
	ErrSignVerification = errors.New("signature verification failed")

	// ErrMerkleRootVerification indicates a failed merkle root verificatin.
	ErrMerkleRootVerification = errors.New("merkle root verification failed")

	// ErrNodePublicKeyNotMatch indicates that the public key given with a node does not match the
	// one in the key store.
	ErrNodePublicKeyNotMatch = errors.New("node publick key doesn't match")
)
