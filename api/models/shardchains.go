package models

// ShardChainsModel groups operations on ShardChain.
type ShardChainsModel struct{}

// CountShardChains returns the count of shard chains created.
func (m *ShardChainsModel) CountShardChains() (count int64, err error) {
	querySQL := `SELECT COUNT(*) FROM "shardChain";`
	return chaindb.SelectInt(querySQL)
}
