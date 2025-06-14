package crypto

import (
    "sync"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestAccountCreation(t *testing.T) {
    rm := NewRewardManager()

    t.Run("create new account", func(t *testing.T) {
        err := rm.CreateAccount("test1", 100.0)
        require.NoError(t, err)

        balance, err := rm.GetBalance("test1")
        assert.NoError(t, err)
        assert.Equal(t, 100.0, balance)
    })

    t.Run("create duplicate account", func(t *testing.T) {
        err := rm.CreateAccount("test1", 200.0)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "already exists")

        // Balance should remain unchanged
        balance, err := rm.GetBalance("test1")
        assert.NoError(t, err)
        assert.Equal(t, 100.0, balance)
    })

    t.Run("invalid initial balance", func(t *testing.T) {
        err := rm.CreateAccount("test2", -100.0)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "invalid initial balance")
    })
}

func TestBalanceOperations(t *testing.T) {
    rm := NewRewardManager()

    // Create test account
    err := rm.CreateAccount("test", 100.0)
    require.NoError(t, err)

    t.Run("get balance", func(t *testing.T) {
        balance, err := rm.GetBalance("test")
        assert.NoError(t, err)
        assert.Equal(t, 100.0, balance)
    })

    t.Run("get nonexistent balance", func(t *testing.T) {
        _, err := rm.GetBalance("nonexistent")
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "not found")
    })

    t.Run("charge for operation", func(t *testing.T) {
        err := rm.ChargeForOperation("test", "download")
        assert.NoError(t, err)

        balance, err := rm.GetBalance("test")
        assert.NoError(t, err)
        assert.Less(t, balance, 100.0) // Balance should decrease
    })

    t.Run("insufficient balance", func(t *testing.T) {
        // Create account with low balance
        err := rm.CreateAccount("poor", 0.1)
        require.NoError(t, err)

        err = rm.ChargeForOperation("poor", "download")
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "insufficient balance")
    })

    t.Run("reward for validation", func(t *testing.T) {
        initialBalance, err := rm.GetBalance("test")
        require.NoError(t, err)

        err = rm.RewardForValidation("test")
        assert.NoError(t, err)

        newBalance, err := rm.GetBalance("test")
        assert.NoError(t, err)
        assert.Greater(t, newBalance, initialBalance) // Balance should increase
    })

    t.Run("reward nonexistent validator", func(t *testing.T) {
        err := rm.RewardForValidation("nonexistent")
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "not found")
    })
}

func TestConcurrentOperations(t *testing.T) {
    rm := NewRewardManager()

    // Create test accounts
    err := rm.CreateAccount("test1", 1000.0)
    require.NoError(t, err)
    err = rm.CreateAccount("test2", 1000.0)
    require.NoError(t, err)

    t.Run("concurrent charges", func(t *testing.T) {
        const numOperations = 100
        var wg sync.WaitGroup
        wg.Add(numOperations)

        // Run multiple charges concurrently
        for i := 0; i < numOperations; i++ {
            go func() {
                defer wg.Done()
                err := rm.ChargeForOperation("test1", "download")
                assert.NoError(t, err)
            }()
        }

        wg.Wait()

        // Verify final balance is correct
        balance, err := rm.GetBalance("test1")
        assert.NoError(t, err)
        assert.Less(t, balance, 1000.0)
    })

    t.Run("concurrent rewards", func(t *testing.T) {
        const numOperations = 100
        var wg sync.WaitGroup
        wg.Add(numOperations)

        initialBalance, err := rm.GetBalance("test2")
        require.NoError(t, err)

        // Run multiple rewards concurrently
        for i := 0; i < numOperations; i++ {
            go func() {
                defer wg.Done()
                err := rm.RewardForValidation("test2")
                assert.NoError(t, err)
            }()
        }

        wg.Wait()

        // Verify final balance is correct
        finalBalance, err := rm.GetBalance("test2")
        assert.NoError(t, err)
        assert.Greater(t, finalBalance, initialBalance)
    })

    t.Run("mixed concurrent operations", func(t *testing.T) {
        const numOperations = 50
        var wg sync.WaitGroup
        wg.Add(numOperations * 2) // For both charges and rewards

        initialBalance, err := rm.GetBalance("test1")
        require.NoError(t, err)

        // Run charges and rewards concurrently
        for i := 0; i < numOperations; i++ {
            go func() {
                defer wg.Done()
                err := rm.ChargeForOperation("test1", "download")
                assert.NoError(t, err)
            }()

            go func() {
                defer wg.Done()
                err := rm.RewardForValidation("test1")
                assert.NoError(t, err)
            }()
        }

        wg.Wait()

        // Verify final balance is different from initial
        finalBalance, err := rm.GetBalance("test1")
        assert.NoError(t, err)
        assert.NotEqual(t, initialBalance, finalBalance)
    })
}

func TestOperationTypeValidation(t *testing.T) {
    rm := NewRewardManager()

    err := rm.CreateAccount("test", 100.0)
    require.NoError(t, err)

    t.Run("invalid operation type", func(t *testing.T) {
        err := rm.ChargeForOperation("test", "invalidop")
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "invalid operation type")

        // Balance should remain unchanged
        balance, err := rm.GetBalance("test")
        assert.NoError(t, err)
        assert.Equal(t, 100.0, balance)
    })

    t.Run("valid operation types", func(t *testing.T) {
        validOps := []string{"download", "upload", "store"}
        initialBalance := 100.0

        for _, op := range validOps {
            err := rm.ChargeForOperation("test", op)
            assert.NoError(t, err)

            balance, err := rm.GetBalance("test")
            assert.NoError(t, err)
            assert.Less(t, balance, initialBalance)
            initialBalance = balance
        }
    })
}
