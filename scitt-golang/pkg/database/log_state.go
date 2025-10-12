package database

import (
	"database/sql"
	"fmt"
)

// TreeState represents the state of the Merkle tree at a specific size
type TreeState struct {
	TreeSize              int64  `json:"tree_size"`
	RootHash              string `json:"root_hash"`
	CheckpointStorageKey  string `json:"checkpoint_storage_key"`
	CheckpointSignedNote  string `json:"checkpoint_signed_note"`
	UpdatedAt             string `json:"updated_at,omitempty"`
}

// GetCurrentTreeSize returns the current size of the Merkle tree
func GetCurrentTreeSize(db *sql.DB) (int64, error) {
	var treeSize int64
	err := db.QueryRow("SELECT tree_size FROM current_tree_size WHERE id = 1").Scan(&treeSize)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get current tree size: %w", err)
	}
	return treeSize, nil
}

// UpdateTreeSize updates the current tree size
func UpdateTreeSize(db *sql.DB, newSize int64) error {
	_, err := db.Exec(`
		UPDATE current_tree_size
		SET tree_size = ?, last_updated = CURRENT_TIMESTAMP
		WHERE id = 1
	`, newSize)

	if err != nil {
		return fmt.Errorf("failed to update tree size: %w", err)
	}

	return nil
}

// RecordTreeState records the tree state at a specific size (for checkpoints)
func RecordTreeState(db *sql.DB, state TreeState) error {
	_, err := db.Exec(`
		INSERT INTO tree_state (
			tree_size, root_hash, checkpoint_storage_key, checkpoint_signed_note
		) VALUES (?, ?, ?, ?)
	`, state.TreeSize, state.RootHash, state.CheckpointStorageKey, state.CheckpointSignedNote)

	if err != nil {
		return fmt.Errorf("failed to record tree state: %w", err)
	}

	return nil
}

// GetTreeState retrieves the tree state for a specific size
func GetTreeState(db *sql.DB, treeSize int64) (*TreeState, error) {
	var state TreeState
	err := db.QueryRow(`
		SELECT tree_size, root_hash, checkpoint_storage_key, checkpoint_signed_note, updated_at
		FROM tree_state
		WHERE tree_size = ?
	`, treeSize).Scan(
		&state.TreeSize,
		&state.RootHash,
		&state.CheckpointStorageKey,
		&state.CheckpointSignedNote,
		&state.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get tree state: %w", err)
	}

	return &state, nil
}

// GetTreeStateHistory returns historical tree states (most recent first)
func GetTreeStateHistory(db *sql.DB, limit int) ([]TreeState, error) {
	query := "SELECT tree_size, root_hash, checkpoint_storage_key, checkpoint_signed_note, updated_at FROM tree_state ORDER BY tree_size DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tree state history: %w", err)
	}
	defer rows.Close()

	var states []TreeState
	for rows.Next() {
		var state TreeState
		if err := rows.Scan(
			&state.TreeSize,
			&state.RootHash,
			&state.CheckpointStorageKey,
			&state.CheckpointSignedNote,
			&state.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan tree state: %w", err)
		}
		states = append(states, state)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tree state rows: %w", err)
	}

	return states, nil
}

// GetLatestCheckpoint returns the most recent tree state (checkpoint)
func GetLatestCheckpoint(db *sql.DB) (*TreeState, error) {
	var state TreeState
	err := db.QueryRow(`
		SELECT tree_size, root_hash, checkpoint_storage_key, checkpoint_signed_note, updated_at
		FROM tree_state
		ORDER BY tree_size DESC
		LIMIT 1
	`).Scan(
		&state.TreeSize,
		&state.RootHash,
		&state.CheckpointStorageKey,
		&state.CheckpointSignedNote,
		&state.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest checkpoint: %w", err)
	}

	return &state, nil
}
