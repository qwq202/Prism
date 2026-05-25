package admin

import (
	"chat/connection"
	"chat/globals"
	"chat/utils"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func openAdminUserTestDB(t *testing.T) *sql.DB {
	t.Helper()

	previousSqlite := globals.SqliteEngine
	globals.SqliteEngine = true
	t.Cleanup(func() {
		globals.SqliteEngine = previousSqlite
	})

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "admin-user.db"))
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = db.Close()
	})

	connection.CreateUserTable(db)
	connection.CreatePackageTable(db)
	connection.CreateQuotaTable(db)
	connection.CreateConversationTable(db)
	connection.CreateMemoryTable(db)
	connection.CreateMaskTable(db)
	connection.CreateSharingTable(db)
	connection.CreateSubscriptionTable(db)
	connection.CreatePasskeyCredentialTable(db)
	connection.CreateInvitationTable(db)
	connection.CreateRedeemTable(db)
	connection.CreateBroadcastTable(db)
	connection.CreateBillingTable(db)
	connection.CreatePaymentOrdersTable(db)

	return db
}

func TestCreateUserCreatesAuthAndQuotaRows(t *testing.T) {
	db := openAdminUserTestDB(t)

	if err := createUser(db, "alice", "alice@example.com", "secret123"); err != nil {
		t.Fatalf("create user: %v", err)
	}

	var id int64
	var bindID int64
	if err := globals.QueryRowDb(db, "SELECT id, bind_id FROM auth WHERE username = ?", "alice").Scan(&id, &bindID); err != nil {
		t.Fatalf("fetch created user: %v", err)
	}
	if id == 0 {
		t.Fatalf("expected created user id")
	}
	if bindID != 1 {
		t.Fatalf("expected bind id 1 after root user, got %d", bindID)
	}

	var quota float32
	if err := globals.QueryRowDb(db, "SELECT quota FROM quota WHERE user_id = ?", id).Scan(&quota); err != nil {
		t.Fatalf("fetch created user quota: %v", err)
	}
	if quota != 0 {
		t.Fatalf("expected default quota 0 without system config, got %f", quota)
	}
}

func TestCreateUserRejectsDuplicateUsername(t *testing.T) {
	db := openAdminUserTestDB(t)

	if err := createUser(db, "alice", "alice@example.com", "secret123"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := createUser(db, "alice", "other@example.com", "secret123"); err == nil {
		t.Fatalf("expected duplicate username to be rejected")
	}
}

func TestCreateUserRejectsMalformedEmailDomain(t *testing.T) {
	db := openAdminUserTestDB(t)

	if err := createUser(db, "alice", "alice@example", "secret123"); err == nil {
		t.Fatalf("expected malformed email domain to be rejected")
	}
}

func TestPasswordMigrationRequiresExistingUser(t *testing.T) {
	db := openAdminUserTestDB(t)

	if err := passwordMigration(db, nil, 999, "secret123"); err == nil {
		t.Fatalf("expected missing user password migration to fail")
	}
}

func TestPasswordMigrationUpdatesExistingUser(t *testing.T) {
	db := openAdminUserTestDB(t)

	if err := createUser(db, "alice", "alice@example.com", "secret123"); err != nil {
		t.Fatalf("create user: %v", err)
	}

	var id int64
	if err := globals.QueryRowDb(db, "SELECT id FROM auth WHERE username = ?", "alice").Scan(&id); err != nil {
		t.Fatalf("fetch created user: %v", err)
	}

	if err := passwordMigration(db, nil, id, "newsecret123"); err != nil {
		t.Fatalf("update user password: %v", err)
	}

	var hash string
	if err := globals.QueryRowDb(db, "SELECT password FROM auth WHERE id = ?", id).Scan(&hash); err != nil {
		t.Fatalf("fetch updated password: %v", err)
	}
	if ok, _ := utils.VerifyPassword("newsecret123", hash); !ok {
		t.Fatalf("expected updated password hash to verify")
	}
}

func TestEmailMigrationValidatesUserAndAddress(t *testing.T) {
	db := openAdminUserTestDB(t)

	if err := createUser(db, "alice", "alice@example.com", "secret123"); err != nil {
		t.Fatalf("create alice: %v", err)
	}
	if err := createUser(db, "bob", "bob@example.com", "secret123"); err != nil {
		t.Fatalf("create bob: %v", err)
	}

	var aliceID int64
	if err := globals.QueryRowDb(db, "SELECT id FROM auth WHERE username = ?", "alice").Scan(&aliceID); err != nil {
		t.Fatalf("fetch alice: %v", err)
	}

	if err := emailMigration(db, 999, "new@example.com"); err == nil {
		t.Fatalf("expected missing user email migration to fail")
	}
	if err := emailMigration(db, aliceID, "not-an-email"); err == nil {
		t.Fatalf("expected invalid email to be rejected")
	}
	if err := emailMigration(db, aliceID, "alice@example"); err == nil {
		t.Fatalf("expected malformed email domain to be rejected")
	}
	if err := emailMigration(db, aliceID, "bob@example.com"); err == nil {
		t.Fatalf("expected duplicate email to be rejected")
	}
	if err := emailMigration(db, aliceID, " alice2@example.com "); err != nil {
		t.Fatalf("update email: %v", err)
	}

	var email string
	if err := globals.QueryRowDb(db, "SELECT email FROM auth WHERE id = ?", aliceID).Scan(&email); err != nil {
		t.Fatalf("fetch updated email: %v", err)
	}
	if email != "alice2@example.com" {
		t.Fatalf("expected trimmed email to be saved, got %q", email)
	}
}

func TestUpdateRootPasswordRequiresRootUser(t *testing.T) {
	db := openAdminUserTestDB(t)

	if err := UpdateRootPassword(db, nil, "newsecret123"); err != nil {
		t.Fatalf("update root password: %v", err)
	}

	if _, err := globals.ExecDb(db, "DELETE FROM auth WHERE username = ?", "root"); err != nil {
		t.Fatalf("delete root: %v", err)
	}
	if err := UpdateRootPassword(db, nil, "anothersecret123"); err == nil {
		t.Fatalf("expected missing root update to fail")
	}
}

func TestAdminOperationsKeepAtLeastOneActiveAdmin(t *testing.T) {
	db := openAdminUserTestDB(t)

	if err := setAdmin(db, 1, false); err == nil {
		t.Fatalf("expected demoting the last active admin to fail")
	}
	if err := banUser(db, 1, true); err == nil {
		t.Fatalf("expected banning the last active admin to fail")
	}
	if err := deleteUser(db, nil, 1); err == nil {
		t.Fatalf("expected deleting the last active admin to fail")
	}
	if err := batchUsers(db, []int64{1}, "ban", 0); err == nil {
		t.Fatalf("expected batch banning the last active admin to fail")
	}
}

func TestAdminOperationsAllowDisablingOneAdminWhenAnotherActiveAdminExists(t *testing.T) {
	db := openAdminUserTestDB(t)

	if err := createUser(db, "alice", "alice@example.com", "secret123"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := setAdmin(db, 2, true); err != nil {
		t.Fatalf("promote second admin: %v", err)
	}
	if err := setAdmin(db, 1, false); err != nil {
		t.Fatalf("demote one of multiple admins: %v", err)
	}

	var rootAdmin bool
	if err := globals.QueryRowDb(db, "SELECT is_admin FROM auth WHERE id = ?", 1).Scan(&rootAdmin); err != nil {
		t.Fatalf("query root admin state: %v", err)
	}
	if rootAdmin {
		t.Fatalf("expected root admin flag to be removed")
	}
}
