package auth

import (
	"chat/connection"
	"chat/globals"
	"chat/utils"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	connection.CreateQuotaTable(db)

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

func TestJWTSigningKeyRejectsUnexpectedAlgorithm(t *testing.T) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS384, jwt.MapClaims{
		"username": "alice",
		"exp":      time.Now().Add(time.Hour).Unix(),
	})

	if _, err := jwtSigningKey(token); err == nil {
		t.Fatalf("expected non-HS256 jwt signing method to be rejected")
	}
}

func TestParseTokenClaimsRejectsMalformedClaims(t *testing.T) {
	now := time.Unix(100, 0)
	tests := []struct {
		name   string
		claims jwt.MapClaims
	}{
		{
			name:   "missing exp",
			claims: jwt.MapClaims{"username": "alice"},
		},
		{
			name:   "string exp",
			claims: jwt.MapClaims{"username": "alice", "exp": "200"},
		},
		{
			name:   "expired exp",
			claims: jwt.MapClaims{"username": "alice", "exp": float64(99)},
		},
		{
			name:   "missing username",
			claims: jwt.MapClaims{"exp": float64(200)},
		},
		{
			name:   "non-string username",
			claims: jwt.MapClaims{"username": 123, "exp": float64(200)},
		},
		{
			name:   "blank username",
			claims: jwt.MapClaims{"username": "  ", "exp": float64(200)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if user, ok := parseTokenClaims(tt.claims, now); ok || user != nil {
				t.Fatalf("expected malformed claims to be rejected, got ok=%v user=%#v", ok, user)
			}
		})
	}
}

func TestParseTokenClaimsAcceptsSafeClaims(t *testing.T) {
	user, ok := parseTokenClaims(jwt.MapClaims{
		"username": " alice ",
		"password": utils.Sha2Encrypt("legacy"),
		"exp":      float64(time.Now().Add(time.Hour).Unix()),
	}, time.Now())
	if !ok {
		t.Fatalf("expected safe claims to parse")
	}
	if user.Username != "alice" {
		t.Fatalf("expected trimmed username, got %q", user.Username)
	}
	if user.Password == "" {
		t.Fatalf("expected legacy password claim to remain compatible")
	}
}

func TestConstantTimeStringEqualRequiresSameValueAndLength(t *testing.T) {
	if !constantTimeStringEqual("123456", "123456") {
		t.Fatalf("expected equal strings to match")
	}
	if constantTimeStringEqual("123456", "654321") {
		t.Fatalf("expected different strings to be rejected")
	}
	if constantTimeStringEqual("123456", "0123456") {
		t.Fatalf("expected equal hash comparison to still require same original length")
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

func TestUseQuotaDeductsQuotaAndUsedAtomically(t *testing.T) {
	db := openAuthSecurityTestDB(t)
	user := GetUserByName(db, "root")
	if user == nil {
		t.Fatalf("expected root user")
	}

	if !user.SetQuota(db, 5) {
		t.Fatalf("set quota")
	}
	if !user.UseQuota(db, 3) {
		t.Fatalf("expected quota usage to succeed")
	}

	if got := user.GetQuota(db); got != 2 {
		t.Fatalf("expected remaining quota 2, got %f", got)
	}
	if got := user.GetUsedQuota(db); got != 3 {
		t.Fatalf("expected used quota 3, got %f", got)
	}

	if user.UseQuota(db, 3) {
		t.Fatalf("expected over-quota usage to fail")
	}
	if got := user.GetQuota(db); got != 2 {
		t.Fatalf("expected failed usage to keep quota 2, got %f", got)
	}
	if got := user.GetUsedQuota(db); got != 3 {
		t.Fatalf("expected failed usage to keep used quota 3, got %f", got)
	}
}

func TestForceUseQuotaRecordsDebtAndUsage(t *testing.T) {
	db := openAuthSecurityTestDB(t)
	user := GetUserByName(db, "root")
	if user == nil {
		t.Fatalf("expected root user")
	}

	if !user.SetQuota(db, 1) {
		t.Fatalf("set quota")
	}
	if !user.ForceUseQuota(db, 3) {
		t.Fatalf("expected forced quota usage to succeed")
	}

	if got := user.GetQuota(db); got != -2 {
		t.Fatalf("expected remaining quota -2, got %f", got)
	}
	if got := user.GetUsedQuota(db); got != 3 {
		t.Fatalf("expected used quota 3, got %f", got)
	}
}
