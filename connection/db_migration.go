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

	// Error 1060: Duplicate column name
	// Error 1050: Table already exists

	return !(strings.Contains(content, "Error 1060") || strings.Contains(content, "Error 1050"))
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

	// add new field `is_banned` in `auth` table
	if err := execSql(execer, `
		ALTER TABLE auth
		ADD COLUMN is_banned BOOLEAN DEFAULT FALSE;
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

	return nil
}

func doSqliteMigration(execer migrationExecer) error {
	// v3.10 added sqlite support, no migration needed before this version

	// v4 migration
	// add new field `task_id` in `conversation` table to store task id (e.g., video job id)
	if err := execSql(execer, `
		ALTER TABLE conversation
		ADD COLUMN task_id VARCHAR(255) NULL;
	`); err != nil {
		return err
	}

	return nil
}
