package personalization

import (
	"chat/globals"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var ErrRevisionConflict = errors.New("personalization settings changed on another device")

func normalizeDatabaseText(value interface{}) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case []byte:
		return strings.TrimSpace(string(typed))
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}

func Load(db *sql.DB, userID int64) (*Record, error) {
	if db == nil {
		return nil, fmt.Errorf("database is not initialized")
	}
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user id")
	}

	var raw string
	var revision int64
	var updatedAt interface{}
	err := globals.QueryRowDb(db, `
		SELECT data, revision, updated_at
		FROM personalization_settings
		WHERE user_id = ?
		LIMIT 1
	`, userID).Scan(&raw, &revision, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var settings Settings
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		return nil, fmt.Errorf("decode personalization settings: %w", err)
	}
	settings, err = normalizeSettings(settings)
	if err != nil {
		return nil, fmt.Errorf("validate stored personalization settings: %w", err)
	}

	return &Record{
		Settings:  settings,
		Revision:  revision,
		UpdatedAt: normalizeDatabaseText(updatedAt),
	}, nil
}

func Save(db *sql.DB, userID int64, settings Settings, baseRevision int64) (*Record, error) {
	if db == nil {
		return nil, fmt.Errorf("database is not initialized")
	}
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user id")
	}
	if baseRevision < 0 {
		return nil, fmt.Errorf("invalid base revision")
	}

	normalized, err := normalizeSettings(settings)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(normalized)
	if err != nil {
		return nil, err
	}

	if baseRevision == 0 {
		_, err = globals.ExecDb(db, `
			INSERT INTO personalization_settings (user_id, data, revision)
			VALUES (?, ?, 1)
		`, userID, string(payload))
		if err != nil {
			current, loadErr := Load(db, userID)
			if loadErr == nil && current != nil {
				return current, ErrRevisionConflict
			}
			return nil, err
		}
		return Load(db, userID)
	}

	result, err := globals.ExecDb(db, `
		UPDATE personalization_settings
		SET data = ?, revision = revision + 1, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ? AND revision = ?
	`, string(payload), userID, baseRevision)
	if err != nil {
		return nil, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		current, loadErr := Load(db, userID)
		if loadErr != nil {
			return nil, loadErr
		}
		return current, ErrRevisionConflict
	}

	return Load(db, userID)
}
