package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var ErrEmptySnapshot = errors.New("empty snapshot")

type Store interface {
	Load(context.Context) (Snapshot, error)
	Save(context.Context, Snapshot) error
	Name() string
}

type MemoryStore struct {
	mu       sync.RWMutex
	snapshot Snapshot
	hasData  bool
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func (s *MemoryStore) Load(_ context.Context) (Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.hasData {
		return Snapshot{}, ErrEmptySnapshot
	}
	return cloneSnapshot(s.snapshot), nil
}

func (s *MemoryStore) Save(_ context.Context, snapshot Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshot = cloneSnapshot(snapshot)
	s.hasData = true
	return nil
}

func (s *MemoryStore) Name() string {
	return "memory"
}

type PostgresStore struct {
	db       *sql.DB
	stateKey string
}

func NewPostgresStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	store := &PostgresStore{
		db:       db,
		stateKey: "primary",
	}
	if err := store.ensureSchema(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *PostgresStore) Load(ctx context.Context) (Snapshot, error) {
	var payload []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT payload
		FROM app_state_snapshots
		WHERE state_key = $1
	`, s.stateKey).Scan(&payload)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Snapshot{}, ErrEmptySnapshot
		}
		return Snapshot{}, err
	}

	var snapshot Snapshot
	if err := json.Unmarshal(payload, &snapshot); err != nil {
		return Snapshot{}, err
	}
	return NormalizeSnapshot(snapshot), nil
}

func (s *PostgresStore) Save(ctx context.Context, snapshot Snapshot) error {
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO app_state_snapshots (state_key, payload, updated_at)
		VALUES ($1, $2::jsonb, $3)
		ON CONFLICT (state_key)
		DO UPDATE SET payload = EXCLUDED.payload, updated_at = EXCLUDED.updated_at
	`, s.stateKey, payload, time.Now().UTC())
	return err
}

func (s *PostgresStore) Name() string {
	return "postgres"
}

func (s *PostgresStore) ensureSchema(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS app_state_snapshots (
			state_key TEXT PRIMARY KEY,
			payload JSONB NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	return err
}

func cloneSnapshot(snapshot Snapshot) Snapshot {
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return NormalizeSnapshot(snapshot)
	}
	var cloned Snapshot
	if err := json.Unmarshal(payload, &cloned); err != nil {
		return NormalizeSnapshot(snapshot)
	}
	return NormalizeSnapshot(cloned)
}
