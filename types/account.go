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

package types

import (
	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

//go:generate hsp

// SQLChainRole defines roles of account in a SQLChain.
type SQLChainRole byte

const (
	// Miner defines the miner role as a SQLChain user.
	Miner SQLChainRole = iota
	// Customer defines the customer role as a SQLChain user.
	Customer
	// NumberOfRoles defines the SQLChain roles number.
	NumberOfRoles
)

// UserPermission defines permissions of a SQLChain user.
type UserPermission int32

const (
	// Unknown defines the initial permission.
	UnknownPerm UserPermission = iota
	// Admin defines the admin user permission.
	Admin
	// Write defines the writer user permission.
	Write
	// Read defines the reader user permission.
	Read
	// NumberOfUserPermission defines the user permission number.
	NumberOfUserPermission
)

// CheckRead returns true if user owns read permission.
func (up *UserPermission) CheckRead() bool {
	return *up >= Admin && *up < NumberOfUserPermission
}

// CheckWrite returns true if user owns write permission.
func (up *UserPermission) CheckWrite() bool {
	return *up >= Admin && *up <= Write
}

// CheckAdmin returns true if user owns admin permission.
func (up *UserPermission) CheckAdmin() bool {
	return *up == Admin
}

// Status defines status of a SQLChain user/miner.
type Status int32

const (
	// Unknown defines initial status.
	UnknownStatus Status = iota
	// Normal defines no bad thing happens.
	Normal
	// Reminder defines the user needs to increase advance payment.
	Reminder
	// Arrears defines the user is in arrears.
	Arrears
	// Arbitration defines the user/miner is in an arbitration.
	Arbitration
	// NumberOfStatus defines the number of status.
	NumberOfStatus
)

func (s *Status) EnableQuery() bool {
	return *s >= Normal && *s <= Reminder
}

// SQLChainUser defines a SQLChain user.
type SQLChainUser struct {
	Address        proto.AccountAddress
	Permission     UserPermission
	AdvancePayment uint64
	Arrears        uint64
	Deposit        uint64
	Status         Status
}

// UserArrears defines user's arrears.
type UserArrears struct {
	User    proto.AccountAddress
	Arrears uint64
}

// MinerInfo defines a miner.
type MinerInfo struct {
	Address        proto.AccountAddress
	NodeID         proto.NodeID
	Name           string
	PendingIncome  uint64
	ReceivedIncome uint64
	UserArrears    []*UserArrears
	Deposit        uint64
	Status         Status
	EncryptionKey  string
}

// SQLChainProfile defines a SQLChainProfile related to an account.
type SQLChainProfile struct {
	ID                proto.DatabaseID
	Address           proto.AccountAddress
	Period            uint64
	GasPrice          uint64
	LastUpdatedHeight uint32

	TokenType TokenType

	Owner proto.AccountAddress
	// first miner in the list is leader
	Miners []*MinerInfo

	Users []*SQLChainUser

	Genesis *Block
}

// ProviderProfile defines a provider list.
type ProviderProfile struct {
	Provider      proto.AccountAddress
	Space         uint64 // reserved storage space in bytes
	Memory        uint64 // reserved memory in bytes
	LoadAvgPerCPU uint64 // max loadAvg15 per CPU
	TargetUser    proto.AccountAddress
	Deposit       uint64 // default 10 Particle
	GasPrice      uint64
	TokenType     TokenType // default Particle
	NodeID        proto.NodeID
}

// Account store its balance, and other mate data.
type Account struct {
	Address      proto.AccountAddress
	TokenBalance [SupportTokenNumber]uint64
	Rating       float64
	NextNonce    pi.AccountNonce
}
