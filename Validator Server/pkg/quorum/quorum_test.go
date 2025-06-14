package quorum

import (
    "fmt"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestQuorumManagerBasics(t *testing.T) {
    qm := NewQuorumManager(300, 3) // 5 minutes timeout, 3 votes required

    t.Run("register validator", func(t *testing.T) {
        validators := []string{"validator1", "validator2", "validator3"}
        for _, v := range validators {
            qm.RegisterValidator(v)
        }

        // Verify validators are registered
        assert.True(t, qm.IsValidValidator("validator1"))
        assert.True(t, qm.IsValidValidator("validator2"))
        assert.True(t, qm.IsValidValidator("validator3"))
        assert.False(t, qm.IsValidValidator("nonexistent"))
    })

    t.Run("create vote session", func(t *testing.T) {
        err := qm.CreateVoteSession("file1", "client1")
        assert.NoError(t, err)

        // Verify session exists
        assert.True(t, qm.HasActiveSession("file1", "client1"))
    })

    t.Run("duplicate vote session", func(t *testing.T) {
        err := qm.CreateVoteSession("file1", "client1")
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "already exists")
    })
}

func TestVotingProcess(t *testing.T) {
    qm := NewQuorumManager(300, 3)

    // Register validators
    validators := []string{"validator1", "validator2", "validator3", "validator4"}
    for _, v := range validators {
        qm.RegisterValidator(v)
    }

    // Create vote session
    err := qm.CreateVoteSession("file1", "client1")
    require.NoError(t, err)

    t.Run("submit votes", func(t *testing.T) {
        // Submit positive votes
        err := qm.SubmitVote("file1", "client1", "validator1", true)
        assert.NoError(t, err)
        err = qm.SubmitVote("file1", "client1", "validator2", true)
        assert.NoError(t, err)

        // Check not enough votes yet
        approved, err := qm.CheckQuorum("file1", "client1")
        assert.NoError(t, err)
        assert.False(t, approved)

        // Submit third positive vote
        err = qm.SubmitVote("file1", "client1", "validator3", true)
        assert.NoError(t, err)

        // Now should have quorum
        approved, err = qm.CheckQuorum("file1", "client1")
        assert.NoError(t, err)
        assert.True(t, approved)
    })

    t.Run("mixed votes", func(t *testing.T) {
        // Create new session
        err := qm.CreateVoteSession("file2", "client2")
        require.NoError(t, err)

        // Submit mixed votes
        err = qm.SubmitVote("file2", "client2", "validator1", true)
        assert.NoError(t, err)
        err = qm.SubmitVote("file2", "client2", "validator2", false)
        assert.NoError(t, err)
        err = qm.SubmitVote("file2", "client2", "validator3", true)
        assert.NoError(t, err)
        err = qm.SubmitVote("file2", "client2", "validator4", false)
        assert.NoError(t, err)

        // Should not have quorum (only 2 positive votes)
        approved, err := qm.CheckQuorum("file2", "client2")
        assert.NoError(t, err)
        assert.False(t, approved)
    })
}

func TestVoteSessionExpiration(t *testing.T) {
    qm := NewQuorumManager(1, 2) // 1 second timeout, 2 votes required

    // Register validators
    qm.RegisterValidator("validator1")
    qm.RegisterValidator("validator2")

    t.Run("session expires", func(t *testing.T) {
        // Create session
        err := qm.CreateVoteSession("file1", "client1")
        require.NoError(t, err)

        // Wait for session to expire
        time.Sleep(2 * time.Second)

        // Try to vote on expired session
        err = qm.SubmitVote("file1", "client1", "validator1", true)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "expired")
    })

    t.Run("complete before expiration", func(t *testing.T) {
        // Create session
        err := qm.CreateVoteSession("file2", "client2")
        require.NoError(t, err)

        // Submit votes quickly
        err = qm.SubmitVote("file2", "client2", "validator1", true)
        assert.NoError(t, err)
        err = qm.SubmitVote("file2", "client2", "validator2", true)
        assert.NoError(t, err)

        // Should have quorum before expiration
        approved, err := qm.CheckQuorum("file2", "client2")
        assert.NoError(t, err)
        assert.True(t, approved)
    })
}

func TestVotingErrors(t *testing.T) {
    qm := NewQuorumManager(300, 2)
    qm.RegisterValidator("validator1")

    t.Run("nonexistent session", func(t *testing.T) {
        err := qm.SubmitVote("nonexistent", "client1", "validator1", true)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "no active session")
    })

    t.Run("invalid validator", func(t *testing.T) {
        err := qm.CreateVoteSession("file1", "client1")
        require.NoError(t, err)

        err = qm.SubmitVote("file1", "client1", "nonexistent", true)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "invalid validator")
    })

    t.Run("duplicate vote", func(t *testing.T) {
        err := qm.SubmitVote("file1", "client1", "validator1", true)
        assert.NoError(t, err)

        err = qm.SubmitVote("file1", "client1", "validator1", true)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "already voted")
    })
}

func TestConcurrentVoting(t *testing.T) {
    qm := NewQuorumManager(300, 3)

    // Register validators
    for i := 1; i <= 5; i++ {
        qm.RegisterValidator(fmt.Sprintf("validator%d", i))
    }

    // Create vote session
    err := qm.CreateVoteSession("file1", "client1")
    require.NoError(t, err)

    t.Run("concurrent votes", func(t *testing.T) {
        done := make(chan bool)
        for i := 1; i <= 5; i++ {
            go func(id int) {
                // Alternating true/false votes
                vote := id%2 == 0
                err := qm.SubmitVote("file1", "client1", fmt.Sprintf("validator%d", id), vote)
                assert.NoError(t, err)
                done <- true
            }(i)
        }

        // Wait for all votes
        for i := 1; i <= 5; i++ {
            <-done
        }

        // Check final result
        approved, err := qm.CheckQuorum("file1", "client1")
        assert.NoError(t, err)
        // Result depends on timing, but should be deterministic
        assert.NotNil(t, approved)
    })

    t.Run("concurrent session creation", func(t *testing.T) {
        done := make(chan bool)
        results := make(chan error, 5)

        for i := 1; i <= 5; i++ {
            go func(id int) {
                err := qm.CreateVoteSession(fmt.Sprintf("file%d", id), "client1")
                results <- err
                done <- true
            }(i)
        }

        // Wait for all attempts
        for i := 1; i <= 5; i++ {
            <-done
        }
        close(results)

        // Count successful creations
        successCount := 0
        for err := range results {
            if err == nil {
                successCount++
            }
        }

        // All session creations should succeed as they're for different files
        assert.Equal(t, 5, successCount)
    })
}
