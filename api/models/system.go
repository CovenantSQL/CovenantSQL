package models

// SystemModel groups operations on the system.
type SystemModel struct{}

// RunningStatus holds information of the network.
type RunningStatus struct {
	BlockHeight      int64 `json:"block_height"`
	CountAccounts    int64 `json:"count_accounts"`
	CountShardChains int64 `json:"count_shardchains"`
	QPS              int64 `json:"qps"`
	StorageSize      int64 `json:"storage_size"`
}

// GetRunningStatus gets current running status of the system.
func (m *SystemModel) GetRunningStatus() (status *RunningStatus, err error) {
	status = new(RunningStatus)

	var blocksModel BlocksModel
	if status.BlockHeight, err = blocksModel.GetMaxHeight(); err != nil {
		return nil, err
	}

	var accountsModel AccountsModel
	if status.CountAccounts, err = accountsModel.CountAccounts(); err != nil {
		return nil, err
	}

	var shardChainsModel ShardChainsModel
	if status.CountShardChains, err = shardChainsModel.CountShardChains(); err != nil {
		return nil, err
	}

	return status, nil
}
