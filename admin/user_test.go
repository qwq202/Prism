package admin

import (
	"chat/channel"
	"chat/connection"
	"chat/globals"
	"chat/utils"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
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
	connection.CreateDrawingWorkspaceTable(db)
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

func openAdminUserTestCache(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()

	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}

	cache := redis.NewClient(&redis.Options{Addr: server.Addr()})
	if err := cache.Ping(cache.Context()).Err(); err != nil {
		t.Fatalf("ping miniredis: %v", err)
	}

	t.Cleanup(func() {
		_ = cache.Close()
		server.Close()
	})

	return server, cache
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

func TestGetUsersFormIncludesSubscriptionWindowUsage(t *testing.T) {
	db := openAdminUserTestDB(t)
	_, cache := openAdminUserTestCache(t)

	previousPlanInstance := channel.PlanInstance
	channel.PlanInstance = &channel.PlanManager{
		Enabled: true,
		Plans: []channel.Plan{
			{
				Level:         1,
				Price:         0,
				Quota:         20,
				ResetInterval: int64((5 * time.Hour).Seconds()),
				WeeklyQuota:   100,
				Items:         []channel.PlanItem{},
			},
		},
	}
	t.Cleanup(func() {
		channel.PlanInstance = previousPlanInstance
	})

	if err := createUser(db, "alice", "alice@example.com", "secret123"); err != nil {
		t.Fatalf("create user: %v", err)
	}

	var userID int64
	if err := globals.QueryRowDb(db, "SELECT id FROM auth WHERE username = ?", "alice").Scan(&userID); err != nil {
		t.Fatalf("fetch alice id: %v", err)
	}

	expiresAt := time.Now().Add(24 * time.Hour).Format("2006-01-02 15:04:05")
	if _, err := globals.ExecDb(db, `
		INSERT INTO subscription (user_id, level, expired_at) VALUES (?, ?, ?)
	`, userID, 1, expiresAt); err != nil {
		t.Fatalf("insert subscription: %v", err)
	}

	plan := channel.PlanInstance.GetPlan(1)
	if !plan.ConsumePointPool(&AuthLike{ID: userID}, cache, "gpt-5.1", 4) {
		t.Fatalf("consume point pool")
	}

	form := getUsersForm(db, cache, 0, "alice")
	if !form.Status {
		t.Fatalf("expected successful users form, got %s", form.Message)
	}
	if len(form.Data) != 1 {
		t.Fatalf("expected one user, got %d", len(form.Data))
	}

	user, ok := form.Data[0].(UserData)
	if !ok {
		t.Fatalf("expected UserData, got %T", form.Data[0])
	}
	if len(user.SubscriptionWindows) != 2 {
		t.Fatalf("expected two subscription windows, got %#v", user.SubscriptionWindows)
	}

	short := user.SubscriptionWindows[0]
	if short.Id != channel.PlanSharedPointsItemID {
		t.Fatalf("expected short window first, got %s", short.Id)
	}
	if short.Used != 4 || short.Total != 20 || short.Remaining != 16 || short.RemainingPercent != 80 {
		t.Fatalf("unexpected short window data: %#v", short)
	}

	weekly := user.SubscriptionWindows[1]
	if weekly.Id != channel.PlanWeeklyPointsItemID {
		t.Fatalf("expected weekly window second, got %s", weekly.Id)
	}
	if weekly.Used != 4 || weekly.Total != 100 || weekly.Remaining != 96 || weekly.RemainingPercent != 96 {
		t.Fatalf("unexpected weekly window data: %#v", weekly)
	}
}

func TestGetUsersFormFiltersAndSortsSubscribedUsers(t *testing.T) {
	db := openAdminUserTestDB(t)
	_, cache := openAdminUserTestCache(t)

	users := []struct {
		username string
		level    int
		expires  time.Time
	}{
		{username: "alice", level: 1, expires: time.Now().Add(24 * time.Hour)},
		{username: "bob", level: 2, expires: time.Now().Add(24 * time.Hour)},
		{username: "carol", level: 3, expires: time.Now().Add(-24 * time.Hour)},
	}

	for _, entry := range users {
		if err := createUser(db, entry.username, entry.username+"@example.com", "secret123"); err != nil {
			t.Fatalf("create user %s: %v", entry.username, err)
		}
		var id int64
		if err := globals.QueryRowDb(db, "SELECT id FROM auth WHERE username = ?", entry.username).Scan(&id); err != nil {
			t.Fatalf("fetch %s id: %v", entry.username, err)
		}
		if _, err := globals.ExecDb(db, `
			INSERT INTO subscription (user_id, level, expired_at) VALUES (?, ?, ?)
		`, id, entry.level, entry.expires.Format("2006-01-02 15:04:05")); err != nil {
			t.Fatalf("insert subscription for %s: %v", entry.username, err)
		}
	}

	form := getUsersForm(db, cache, 0, "", userListFilter{
		Plan: "yes",
		Sort: "plan-desc",
	})
	if !form.Status {
		t.Fatalf("expected successful users form, got %s", form.Message)
	}
	if form.Total != 1 {
		t.Fatalf("expected one result page, got %d", form.Total)
	}
	if len(form.Data) != 2 {
		t.Fatalf("expected two active subscribed users, got %d", len(form.Data))
	}

	first := form.Data[0].(UserData)
	second := form.Data[1].(UserData)
	if first.Username != "bob" || first.Level != 2 {
		t.Fatalf("expected bob with level 2 first, got %#v", first)
	}
	if second.Username != "alice" || second.Level != 1 {
		t.Fatalf("expected alice with level 1 second, got %#v", second)
	}
}

func TestReleaseUsageForSubscribedUsersResetsPointWindows(t *testing.T) {
	db := openAdminUserTestDB(t)
	_, cache := openAdminUserTestCache(t)

	previousPlanInstance := channel.PlanInstance
	channel.PlanInstance = &channel.PlanManager{
		Enabled: true,
		Plans: []channel.Plan{
			{
				Level:         1,
				Quota:         20,
				ResetInterval: int64((5 * time.Hour).Seconds()),
				WeeklyQuota:   100,
				Items:         []channel.PlanItem{},
			},
		},
	}
	t.Cleanup(func() {
		channel.PlanInstance = previousPlanInstance
	})

	userIDs := make([]int64, 0, 2)
	for _, username := range []string{"alice", "bob"} {
		if err := createUser(db, username, username+"@example.com", "secret123"); err != nil {
			t.Fatalf("create user %s: %v", username, err)
		}
		var id int64
		if err := globals.QueryRowDb(db, "SELECT id FROM auth WHERE username = ?", username).Scan(&id); err != nil {
			t.Fatalf("fetch %s id: %v", username, err)
		}
		if _, err := globals.ExecDb(db, `
			INSERT INTO subscription (user_id, level, expired_at) VALUES (?, ?, ?)
		`, id, 1, time.Now().Add(24*time.Hour).Format("2006-01-02 15:04:05")); err != nil {
			t.Fatalf("insert subscription for %s: %v", username, err)
		}
		userIDs = append(userIDs, id)
	}

	plan := channel.PlanInstance.GetPlan(1)
	for _, id := range userIDs {
		if !plan.ConsumePointPool(&AuthLike{ID: id}, cache, "gpt-5.1", 4) {
			t.Fatalf("consume point pool for user %d", id)
		}
	}

	count, err := releaseUsageForSubscribedUsers(db, cache, releaseUsageTypeHour)
	if err != nil {
		t.Fatalf("release hourly usage for subscribed users: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected to reset 2 hourly windows, got %d", count)
	}
	for _, id := range userIDs {
		user := &AuthLike{ID: id}
		if got := plan.GetPointUsage(user, cache); got != 0 {
			t.Fatalf("expected hourly usage reset for user %d, got %f", id, got)
		}
		if got := plan.GetWeeklyPointUsage(user, cache); got != 4 {
			t.Fatalf("expected weekly usage preserved for user %d, got %f", id, got)
		}
	}

	count, err = releaseUsageForSubscribedUsers(db, cache, releaseUsageTypeWeek)
	if err != nil {
		t.Fatalf("release weekly usage for subscribed users: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected to reset 2 weekly windows, got %d", count)
	}
	for _, id := range userIDs {
		if got := plan.GetWeeklyPointUsage(&AuthLike{ID: id}, cache); got != 0 {
			t.Fatalf("expected weekly usage reset for user %d, got %f", id, got)
		}
	}
}

func TestReleaseUsageRejectsExpiredSubscription(t *testing.T) {
	db := openAdminUserTestDB(t)
	_, cache := openAdminUserTestCache(t)

	previousPlanInstance := channel.PlanInstance
	channel.PlanInstance = &channel.PlanManager{
		Enabled: true,
		Plans: []channel.Plan{
			{
				Level:         1,
				Quota:         20,
				ResetInterval: int64((5 * time.Hour).Seconds()),
				WeeklyQuota:   100,
				Items:         []channel.PlanItem{},
			},
		},
	}
	t.Cleanup(func() {
		channel.PlanInstance = previousPlanInstance
	})

	if err := createUser(db, "alice", "alice@example.com", "secret123"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	var userID int64
	if err := globals.QueryRowDb(db, "SELECT id FROM auth WHERE username = ?", "alice").Scan(&userID); err != nil {
		t.Fatalf("fetch alice id: %v", err)
	}
	if _, err := globals.ExecDb(db, `
		INSERT INTO subscription (user_id, level, expired_at) VALUES (?, ?, ?)
	`, userID, 1, time.Now().Add(-24*time.Hour).Format("2006-01-02 15:04:05")); err != nil {
		t.Fatalf("insert expired subscription: %v", err)
	}

	if err := releaseUsage(db, cache, userID, releaseUsageTypeHour); err == nil || err.Error() != "user is not subscribed" {
		t.Fatalf("expected expired subscription to be rejected, got %v", err)
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
