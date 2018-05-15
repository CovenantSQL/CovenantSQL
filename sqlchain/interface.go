package sqlchain

import "github.com/thunderdb/ThunderDB/crypto/hash"

// Hashable is an interface whitch make struct hashable
type Hashable interface {
	// Hash return the hash of a hashable value
	Hash() *hash.Hash
}
