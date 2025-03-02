package processor

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// LeaderElection defines the interface for leader election
type LeaderElection interface {
	Start(ctx context.Context) error
	Stop() error
	// IsLeader checks if the current instance is the leader
	IsLeader() bool
}

type DatabaseLeaderElection struct {
	db         *pgxpool.Pool
	instanceID string
	isLeader   bool
	stopCh     chan struct{}
	mu         sync.Mutex
}

func NewDatabaseLeaderElection(db *pgxpool.Pool, instanceID string) *DatabaseLeaderElection {
	return &DatabaseLeaderElection{
		db:         db,
		instanceID: instanceID,
		isLeader:   false,
		stopCh:     make(chan struct{}),
	}
}

func (l *DatabaseLeaderElection) Start(ctx context.Context) error {
	// Try to become the leader immediately
	l.tryBecomeLeader(ctx)

	// Start a goroutine to periodically try to become the leader
	go l.leaderElectionLoop(ctx)

	return nil
}

func (l *DatabaseLeaderElection) Stop() error {
	close(l.stopCh)

	// If we are the leader, release leadership
	l.mu.Lock()
	isLeader := l.isLeader
	l.isLeader = false
	l.mu.Unlock()

	if isLeader {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Release leadership
		_, err := l.db.Exec(ctx, `
			DELETE FROM leader_election
			WHERE id = 'outbox_leader' AND instance_id = $1
		`, l.instanceID)
		if err != nil {
			return fmt.Errorf("failed to release leadership: %w", err)
		}
	}

	return nil
}

// IsLeader checks if the current instance is the leader
func (l *DatabaseLeaderElection) IsLeader() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.isLeader
}

// leaderElectionLoop regularly attempts to claim leadership
func (l *DatabaseLeaderElection) leaderElectionLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-l.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.tryBecomeLeader(ctx)
		}
	}
}

// tryBecomeLeader attempts to claim leadership
func (l *DatabaseLeaderElection) tryBecomeLeader(ctx context.Context) {
	// First, check if there's a current leader and if it's still active
	var instanceID string
	var lastHeartbeat time.Time

	err := l.db.QueryRow(ctx, `
		SELECT instance_id, last_heartbeat
		FROM leader_election
		WHERE id = 'outbox_leader'
	`).Scan(&instanceID, &lastHeartbeat)

	// If there's no leader or the leader's heartbeat is too old, try to become the leader
	if err != nil || time.Since(lastHeartbeat) > 10*time.Second {
		// Try to insert or update the leader record
		_, err := l.db.Exec(ctx, `
			INSERT INTO leader_election (id, instance_id, last_heartbeat)
			VALUES ('outbox_leader', $1, NOW())
			ON CONFLICT (id) DO UPDATE
			SET instance_id = $1, last_heartbeat = NOW()
			WHERE leader_election.last_heartbeat < NOW() - INTERVAL '10 seconds'
		`, l.instanceID)

		if err != nil {
			log.Printf("Failed to update leader election: %v", err)
			l.setIsLeader(false)
			return
		}

		// Check if we became the leader
		err = l.db.QueryRow(ctx, `
			SELECT instance_id
			FROM leader_election
			WHERE id = 'outbox_leader'
		`).Scan(&instanceID)

		if err != nil {
			log.Printf("Failed to check leader: %v", err)
			l.setIsLeader(false)
			return
		}

		l.setIsLeader(instanceID == l.instanceID)
	} else if instanceID == l.instanceID {
		// If we're already the leader, update the heartbeat
		_, err := l.db.Exec(ctx, `
			UPDATE leader_election
			SET last_heartbeat = NOW()
			WHERE id = 'outbox_leader' AND instance_id = $1
		`, l.instanceID)

		if err != nil {
			log.Printf("Failed to update heartbeat: %v", err)
			l.setIsLeader(false)
			return
		}

		l.setIsLeader(true)
	} else {
		// Someone else is the leader
		l.setIsLeader(false)
	}
}

func (l *DatabaseLeaderElection) setIsLeader(isLeader bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.isLeader != isLeader {
		if isLeader {
			log.Printf("Instance %s became the leader", l.instanceID)
		} else if l.isLeader {
			log.Printf("Instance %s lost leadership", l.instanceID)
		}
	}

	l.isLeader = isLeader
}
