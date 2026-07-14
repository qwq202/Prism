package connection

import (
	"chat/globals"
	"database/sql"
	"strings"
)

type migrationExecer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

func validSqlError(err error) bool {
	if err == nil {
		return false
	}

	content := err.Error()
	lower := strings.ToLower(content)

	// Error 1060: Duplicate column name
	// Error 1061: Duplicate key name
	// Error 1050: Table already exists
	// SQLite: duplicate column name

	return !(strings.Contains(content, "Error 1060") ||
		strings.Contains(content, "Error 1061") ||
		strings.Contains(content, "Error 1050") ||
		strings.Contains(lower, "already exists") ||
		strings.Contains(lower, "duplicate column name"))
}

func checkSqlError(_ sql.Result, err error) error {
	if validSqlError(err) {
		return err
	}

	return nil
}

func execSql(execer migrationExecer, query string, args ...interface{}) error {
	return checkSqlError(execer.Exec(globals.PreflightSql(query), args...))
}

func runMigrationTx(db *sql.DB, migrate func(migrationExecer) error) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	if err := migrate(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

func doMigration(db *sql.DB) error {
	if globals.SqliteEngine {
		return runMigrationTx(db, doSqliteMigration)
	}

	return runMigrationTx(db, doMysqlMigration)
}

func doMysqlMigration(execer migrationExecer) error {

	// v3.10 migration

	// update `quota`, `used` field in `quota` table
	// migrate `DECIMAL(16, 4)` to `DECIMAL(24, 6)`

	if err := execSql(execer, `
		ALTER TABLE quota
		MODIFY COLUMN quota DECIMAL(24, 6),
		MODIFY COLUMN used DECIMAL(24, 6);
	`); err != nil {
		return err
	}

	if err := execSql(execer, `
		ALTER TABLE quota
		ADD COLUMN allow_subscription_quota_fallback BOOLEAN DEFAULT FALSE;
	`); err != nil {
		return err
	}

	// add new field `is_banned` in `auth` table
	if err := execSql(execer, `
		ALTER TABLE auth
		ADD COLUMN is_banned BOOLEAN DEFAULT FALSE;
	`); err != nil {
		return err
	}

	if err := execSql(execer, `
		ALTER TABLE auth
		ADD COLUMN created_at DATETIME DEFAULT CURRENT_TIMESTAMP;
	`); err != nil {
		return err
	}

	if err := execSql(execer, `
		UPDATE auth
		LEFT JOIN quota ON quota.user_id = auth.id
		SET auth.created_at = COALESCE(quota.created_at, auth.created_at, CURRENT_TIMESTAMP)
		WHERE quota.created_at IS NOT NULL OR auth.created_at IS NULL;
	`); err != nil {
		return err
	}

	// add new field `task_id` in `conversation` table to store task id (e.g., video job id)
	if err := execSql(execer, `
		ALTER TABLE conversation
		ADD COLUMN task_id VARCHAR(255) NULL;
	`); err != nil {
		return err
	}

	if err := execSql(execer, `
		ALTER TABLE conversation
		ADD COLUMN updated_at DATETIME DEFAULT CURRENT_TIMESTAMP;
	`); err != nil {
		return err
	}

	if err := execSql(execer, `
		ALTER TABLE conversation
		ADD COLUMN favorite BOOLEAN DEFAULT FALSE;
	`); err != nil {
		return err
	}

	if err := execSql(execer, `
		ALTER TABLE chat_request
		ADD COLUMN owner_token VARCHAR(64) NOT NULL DEFAULT '';
	`); err != nil {
		return err
	}

	if err := execSql(execer, `
		ALTER TABLE chat_request
		ADD COLUMN lease_expires_at BIGINT NOT NULL DEFAULT 0;
	`); err != nil {
		return err
	}

	// add batch_id to redeem table for batch history tracking
	if err := execSql(execer, `
		ALTER TABLE redeem
		ADD COLUMN batch_id VARCHAR(32) NULL;
	`); err != nil {
		return err
	}

	// create redeem_batch table for batch metadata
	if err := execSql(execer, `
		CREATE TABLE IF NOT EXISTS redeem_batch (
		  id VARCHAR(32) PRIMARY KEY,
		  quota DECIMAL(16, 4),
		  count INT DEFAULT 0,
		  used_count INT DEFAULT 0,
		  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`); err != nil {
		return err
	}

	// add type/schedule fields to broadcast table
	if err := execSql(execer, `
		ALTER TABLE broadcast
		ADD COLUMN type VARCHAR(20) DEFAULT 'broadcast';
	`); err != nil {
		return err
	}

	if err := execSql(execer, `
		ALTER TABLE broadcast
		ADD COLUMN start_at DATETIME NULL;
	`); err != nil {
		return err
	}

	if err := execSql(execer, `
		ALTER TABLE broadcast
		ADD COLUMN end_at DATETIME NULL;
	`); err != nil {
		return err
	}

	if err := execSql(execer, `
		ALTER TABLE broadcast
		ADD COLUMN is_active BOOLEAN DEFAULT TRUE;
	`); err != nil {
		return err
	}

	if err := execSql(execer, `
		ALTER TABLE passkey_credential
		ADD COLUMN public_key TEXT;
	`); err != nil {
		return err
	}

	if err := execSql(execer, `
		ALTER TABLE passkey_credential
		ADD COLUMN sign_count INT DEFAULT 0;
	`); err != nil {
		return err
	}

	if err := execSql(execer, `
		ALTER TABLE model_usage_metrics
		ADD INDEX idx_model_usage_metrics_model_created_at (model, created_at);
	`); err != nil {
		return err
	}

	if err := execSql(execer, `
		UPDATE quota
		SET quota = 0
		WHERE quota < 0;
	`); err != nil {
		return err
	}

	return nil
}

func doSqliteMigration(execer migrationExecer) error {
	// v3.10 added sqlite support, no migration needed before this version

	// v4 migration
	if err := execSql(execer, `
		ALTER TABLE quota
		ADD COLUMN allow_subscription_quota_fallback BOOLEAN DEFAULT FALSE;
	`); err != nil {
		return err
	}

	// add new field `task_id` in `conversation` table to store task id (e.g., video job id)
	if err := execSql(execer, `
		ALTER TABLE conversation
		ADD COLUMN task_id VARCHAR(255) NULL;
	`); err != nil {
		return err
	}

	if err := execSql(execer, `
		ALTER TABLE conversation
		ADD COLUMN updated_at DATETIME NULL;
	`); err != nil {
		return err
	}

	if err := execSql(execer, `
		ALTER TABLE conversation
		ADD COLUMN favorite BOOLEAN DEFAULT FALSE;
	`); err != nil {
		return err
	}

	if err := execSql(execer, `
		ALTER TABLE chat_request
		ADD COLUMN owner_token VARCHAR(64) NOT NULL DEFAULT '';
	`); err != nil {
		return err
	}

	if err := execSql(execer, `
		ALTER TABLE chat_request
		ADD COLUMN lease_expires_at BIGINT NOT NULL DEFAULT 0;
	`); err != nil {
		return err
	}

	if err := execSql(execer, `
		ALTER TABLE auth
		ADD COLUMN created_at DATETIME NULL;
	`); err != nil {
		return err
	}

	if err := execSql(execer, `
		UPDATE auth
		SET created_at = COALESCE(
			(SELECT quota.created_at FROM quota WHERE quota.user_id = auth.id),
			created_at,
			CURRENT_TIMESTAMP
		)
		WHERE created_at IS NULL;
	`); err != nil {
		return err
	}

	// add batch_id to redeem table
	if err := execSql(execer, `
		ALTER TABLE redeem
		ADD COLUMN batch_id VARCHAR(32) NULL;
	`); err != nil {
		return err
	}

	// create redeem_batch table
	if err := execSql(execer, `
		CREATE TABLE IF NOT EXISTS redeem_batch (
		  id VARCHAR(32) PRIMARY KEY,
		  quota DECIMAL(16, 4),
		  count INTEGER DEFAULT 0,
		  used_count INTEGER DEFAULT 0,
		  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`); err != nil {
		return err
	}

	// add type/schedule fields to broadcast table
	if err := execSql(execer, `
		ALTER TABLE broadcast
		ADD COLUMN type VARCHAR(20) DEFAULT 'broadcast';
	`); err != nil {
		return err
	}

	if err := execSql(execer, `
		ALTER TABLE broadcast
		ADD COLUMN start_at DATETIME NULL;
	`); err != nil {
		return err
	}

	if err := execSql(execer, `
		ALTER TABLE broadcast
		ADD COLUMN end_at DATETIME NULL;
	`); err != nil {
		return err
	}

	if err := execSql(execer, `
		ALTER TABLE broadcast
		ADD COLUMN is_active BOOLEAN DEFAULT TRUE;
	`); err != nil {
		return err
	}

	if err := execSql(execer, `
		ALTER TABLE passkey_credential
		ADD COLUMN public_key TEXT;
	`); err != nil {
		return err
	}

	if err := execSql(execer, `
		ALTER TABLE passkey_credential
		ADD COLUMN sign_count INT DEFAULT 0;
	`); err != nil {
		return err
	}

	if err := execSql(execer, `
		CREATE INDEX IF NOT EXISTS idx_model_usage_metrics_model_created_at
		ON model_usage_metrics (model, created_at);
	`); err != nil {
		return err
	}

	if err := execSql(execer, `
		UPDATE quota
		SET quota = 0
		WHERE quota < 0;
	`); err != nil {
		return err
	}

	return nil
}
