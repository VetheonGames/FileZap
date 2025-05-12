package crypto

import (
"fmt"
"sync"
"time"
)

// Transaction represents a crypto transaction
type Transaction struct {
FromID     string
ToID       string
Amount     float64
Type       string // "upload", "validate", "download"
Timestamp  int64
}

// Account represents a user's balance
type Account struct {
ID          string
Balance     float64
LastUpdated int64
}

// RewardManager handles cryptocurrency rewards and costs
type RewardManager struct {
accounts     map[string]*Account
transactions []Transaction
costs        map[string]float64 // operation costs
rewards      map[string]float64 // operation rewards
mu           sync.RWMutex
}

// NewRewardManager creates a new reward manager with default costs/rewards
func NewRewardManager() *RewardManager {
rm := &RewardManager{
accounts:     make(map[string]*Account),
transactions: make([]Transaction, 0),
costs: map[string]float64{
"upload":   1.0,  // Cost to upload a file
"validate": 0.1,  // Cost to request validation
"download": 0.5,  // Cost to download a file
},
rewards: map[string]float64{
"validate": 0.05, // Reward for validating a file
},
}
return rm
}

// CreateAccount creates a new account with initial balance
func (rm *RewardManager) CreateAccount(id string, initialBalance float64) error {
rm.mu.Lock()
defer rm.mu.Unlock()

if _, exists := rm.accounts[id]; exists {
return fmt.Errorf("account already exists")
}

rm.accounts[id] = &Account{
ID:          id,
Balance:     initialBalance,
LastUpdated: time.Now().Unix(),
}

return nil
}

// GetBalance returns an account's current balance
func (rm *RewardManager) GetBalance(id string) (float64, error) {
rm.mu.RLock()
defer rm.mu.RUnlock()

account, exists := rm.accounts[id]
if !exists {
return 0, fmt.Errorf("account not found")
}

return account.Balance, nil
}

// ProcessTransaction handles a new transaction
func (rm *RewardManager) ProcessTransaction(tx Transaction) error {
rm.mu.Lock()
defer rm.mu.Unlock()

// Verify accounts exist
from, fromExists := rm.accounts[tx.FromID]
if !fromExists {
return fmt.Errorf("sender account not found")
}

to, toExists := rm.accounts[tx.ToID]
if !toExists {
return fmt.Errorf("recipient account not found")
}

// Check sufficient balance
if from.Balance < tx.Amount {
return fmt.Errorf("insufficient balance")
}

// Update balances
from.Balance -= tx.Amount
to.Balance += tx.Amount

// Record transaction
tx.Timestamp = time.Now().Unix()
rm.transactions = append(rm.transactions, tx)

return nil
}

// ChargeForOperation charges an account for an operation
func (rm *RewardManager) ChargeForOperation(accountID, operation string) error {
cost, exists := rm.costs[operation]
if !exists {
return fmt.Errorf("invalid operation type")
}

tx := Transaction{
FromID:    accountID,
ToID:      "system", // System account collects fees
Amount:    cost,
Type:      operation,
Timestamp: time.Now().Unix(),
}

return rm.ProcessTransaction(tx)
}

// RewardForValidation rewards validators for participating
func (rm *RewardManager) RewardForValidation(validatorID string) error {
reward, exists := rm.rewards["validate"]
if !exists {
return fmt.Errorf("invalid operation type")
}

tx := Transaction{
FromID:    "system", // System account pays rewards
ToID:      validatorID,
Amount:    reward,
Type:      "validate",
Timestamp: time.Now().Unix(),
}

return rm.ProcessTransaction(tx)
}

// GetTransactionHistory returns all transactions for an account
func (rm *RewardManager) GetTransactionHistory(accountID string) []Transaction {
rm.mu.RLock()
defer rm.mu.RUnlock()

var history []Transaction
for _, tx := range rm.transactions {
if tx.FromID == accountID || tx.ToID == accountID {
history = append(history, tx)
}
}

return history
}

// SetOperationCost updates the cost for an operation
func (rm *RewardManager) SetOperationCost(operation string, cost float64) {
rm.mu.Lock()
defer rm.mu.Unlock()
rm.costs[operation] = cost
}

// SetOperationReward updates the reward for an operation
func (rm *RewardManager) SetOperationReward(operation string, reward float64) {
rm.mu.Lock()
defer rm.mu.Unlock()
rm.rewards[operation] = reward
}
