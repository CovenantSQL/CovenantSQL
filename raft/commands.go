package raft

import "github.com/thunderdb/ThunderDB/crypto/hash"

// RPCHeader is a common sub-structure used to pass along protocol version and
// other information about the cluster. For older Raft implementations before
// versioning was added this will default to a zero-valued structure when read
// by newer Raft versions.
type RPCHeader struct {
	// ProtocolVersion is the version of the protocol the sender is
	// speaking.
	ProtocolVersion ProtocolVersion
}

// WithRPCHeader is an interface that exposes the RPC header.
type WithRPCHeader interface {
	GetRPCHeader() RPCHeader
}

// AppendEntriesRequest is the command used to append entries to the
// replicated log.
type AppendEntriesRequest struct {
	RPCHeader

	// Provide the current term and leader
	Term   uint64
	Leader []byte

	// Provide the previous entries for integrity checking
	PrevLogEntry uint64
	PrevLogTerm  uint64
	PrevLogHash  *hash.Hash

	// New entries to commit
	Entries []*Log

	// Commit index on the leader
	LeaderCommitIndex uint64
}

// GetRPCHeader implements WithRPCHeader.
func (r *AppendEntriesRequest) GetRPCHeader() RPCHeader {
	return r.RPCHeader
}

// AppendEntriesResponse is the response returned from an
// AppendEntriesRequest.
type AppendEntriesResponse struct {
	RPCHeader

	// Newer term if leader is out of date
	Term uint64

	// Last Log is a hint to help accelerate rebuilding slow nodes
	LastLog uint64

	// We may not succeed if we have a conflicting entry
	Success bool

	// There are scenarios where this request didn't succeed
	// but there's no need to wait/back-off the next attempt.
	NoRetryBackoff bool
}

// GetRPCHeader implements WithRPCHeader.
func (r *AppendEntriesResponse) GetRPCHeader() RPCHeader {
	return r.RPCHeader
}

// CollectLogRequest is the command used for leader to collect logs from peers during quorum recovery
type CollectLogRequest struct {
	RPCHeader

	// Provide the current term and leader
	Term   uint64
	Leader []byte

	// Provide the previous entries as start position and integrity check
	PrevLogEntry uint64
	PrevLogTerm  uint64
	PrevLogHash  *hash.Hash
}

// GetRPCHeader implements WithRPCHeader.
func (r *CollectLogRequest) GetRPCHeader() RPCHeader {
	return r.RPCHeader
}

// CollectLogResponse is the response returned from a
// CollectLogRequest.
type CollectLogResponse struct {
	RPCHeader

	// New entries to commit
	Entries []*Log
}

// GetRPCHeader implements WithRPCHeader.
func (r *CollectLogResponse) GetRPCHeader() RPCHeader {
	return r.RPCHeader
}
