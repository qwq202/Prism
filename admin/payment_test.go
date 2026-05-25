package admin

import (
	"chat/globals"
	"testing"
)

func TestGetPaymentOrdersFormSearchesDisplayedUsername(t *testing.T) {
	db := openAdminUserTestDB(t)

	if err := createUser(db, "alice", "alice@example.com", "secret123"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	var userID int64
	if err := globals.QueryRowDb(db, "SELECT id FROM auth WHERE username = ?", "alice").Scan(&userID); err != nil {
		t.Fatalf("select user: %v", err)
	}
	if _, err := globals.ExecDb(db, `
		INSERT INTO payment_orders (user_id, username, type, service, amount, order_id, name, device, state)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, userID, "alice", "quota", "manual", 12.5, "order-renamed-user", "Quota", "desktop", false); err != nil {
		t.Fatalf("insert payment order: %v", err)
	}
	if _, err := globals.ExecDb(db, "UPDATE auth SET username = ? WHERE id = ?", "alice-renamed", userID); err != nil {
		t.Fatalf("rename user: %v", err)
	}

	result := getPaymentOrdersForm(db, 0, "alice-renamed")
	if !result.Status {
		t.Fatalf("expected payment order search to pass, got %q", result.Message)
	}
	if len(result.Data) != 1 {
		t.Fatalf("expected one order for displayed username search, got %#v", result.Data)
	}
	if result.Data[0].Username != "alice-renamed" {
		t.Fatalf("expected displayed username from auth table, got %q", result.Data[0].Username)
	}
	if result.Total != 1 {
		t.Fatalf("expected one page, got %d", result.Total)
	}
}

func TestGetPaymentOrdersFormStillSearchesSnapshotUsername(t *testing.T) {
	db := openAdminUserTestDB(t)

	if _, err := globals.ExecDb(db, `
		INSERT INTO payment_orders (user_id, username, type, service, amount, order_id, name, device, state)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, 999, "guest-checkout", "quota", "manual", 5.0, "order-without-auth", "Quota", "mobile", true); err != nil {
		t.Fatalf("insert payment order: %v", err)
	}

	result := getPaymentOrdersForm(db, -1, "guest-checkout")
	if !result.Status {
		t.Fatalf("expected payment order search to pass, got %q", result.Message)
	}
	if len(result.Data) != 1 {
		t.Fatalf("expected one order for snapshot username search, got %#v", result.Data)
	}
	if result.Data[0].Username != "guest-checkout" {
		t.Fatalf("expected snapshot username fallback, got %q", result.Data[0].Username)
	}
}
