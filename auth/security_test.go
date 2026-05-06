package auth

import (
	"chat/connection"
	"chat/globals"
	"chat/utils"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
)

func openAuthSecurityTestDB(t *testing.T) *sql.DB {
	t.Helper()

	previousSqlite := globals.SqliteEngine
	globals.SqliteEngine = true
	t.Cleanup(func() {
		globals.SqliteEngine = previousSqlite
	})

	previousSecret := viper.GetString("secret")
	viper.Set("secret", strings.Repeat("s", 32))
	t.Cleanup(func() {
		viper.Set("secret", previousSecret)
	})

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "auth-security.db"))
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = db.Close()
	})

	connection.CreateUserTable(db)
	connection.CreateApiKeyTable(db)

	return db
}

func TestGenerateTokenOmitsPasswordHash(t *testing.T) {
	previousSecret := viper.GetString("secret")
	viper.Set("secret", strings.Repeat("s", 32))
	t.Cleanup(func() {
		viper.Set("secret", previousSecret)
	})

	token, err := (&User{
		Username: "alice",
		Password: utils.Sha2Encrypt("password"),
	}).GenerateToken()
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	instance, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(viper.GetString("secret")), nil
	})
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}

	claims, ok := instance.Claims.(jwt.MapClaims)
	if !ok || !instance.Valid {
		t.Fatalf("expected valid jwt claims")
	}
	if _, ok := claims["password"]; ok {
		t.Fatalf("expected token claims to omit password hash")
	}
}

func TestCreateApiKeyStoresHash(t *testing.T) {
	db := openAuthSecurityTestDB(t)
	user := GetUserByName(db, "root")
	if user == nil {
		t.Fatalf("expected root user")
	}

	key := user.CreateApiKey(db)
	if !strings.HasPrefix(key, "sk-") {
		t.Fatalf("expected raw api key to be returned once, got %q", key)
	}

	var stored string
	if err := globals.QueryRowDb(db, "SELECT api_key FROM apikey WHERE user_id = ?", user.GetID(db)).Scan(&stored); err != nil {
		t.Fatalf("query stored api key: %v", err)
	}
	if stored == key {
		t.Fatalf("expected api key to be stored as hash")
	}
	if !isHashedApiKey(stored) {
		t.Fatalf("expected stored api key hash prefix, got %q", stored)
	}

	if got := ParseApiKeyByHash(db, key); got == nil || got.Username != user.Username {
		t.Fatalf("expected hashed api key lookup to find user, got %#v", got)
	}
}

func TestParseApiKeyMigratesLegacyPlaintextKey(t *testing.T) {
	db := openAuthSecurityTestDB(t)
	user := GetUserByName(db, "root")
	if user == nil {
		t.Fatalf("expected root user")
	}

	const legacyKey = "sk-legacy-plaintext"
	if _, err := globals.ExecDb(db, "INSERT INTO apikey (user_id, api_key) VALUES (?, ?)", user.GetID(db), legacyKey); err != nil {
		t.Fatalf("insert legacy api key: %v", err)
	}

	if got := ParseApiKeyByHash(db, legacyKey); got == nil || got.Username != user.Username {
		t.Fatalf("expected legacy api key lookup to find user, got %#v", got)
	}

	var stored string
	if err := globals.QueryRowDb(db, "SELECT api_key FROM apikey WHERE user_id = ?", user.GetID(db)).Scan(&stored); err != nil {
		t.Fatalf("query migrated api key: %v", err)
	}
	if stored == legacyKey || !isHashedApiKey(stored) {
		t.Fatalf("expected legacy api key to migrate to hash, got %q", stored)
	}
}
