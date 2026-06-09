package auth

import (
	"chat/channel"
	"chat/connection"
	"chat/globals"
	"chat/utils"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v4"
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
	connection.CreateQuotaTable(db)

	return db
}

func newAuthSecurityTestContext(t *testing.T, db *sql.DB) *gin.Context {
	t.Helper()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set("db", db)

	cache := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:1",
		MaxRetries:   -1,
		DialTimeout:  time.Millisecond,
		ReadTimeout:  time.Millisecond,
		WriteTimeout: time.Millisecond,
	})
	t.Cleanup(func() {
		_ = cache.Close()
	})
	c.Set("cache", cache)

	return c
}

func TestGenerateTokenOmitsPasswordHash(t *testing.T) {
	previousSecret := viper.GetString("secret")
	viper.Set("secret", strings.Repeat("s", 32))
	t.Cleanup(func() {
		viper.Set("secret", previousSecret)
	})

	passwordHash := utils.Sha2Encrypt("password")
	token, err := (&User{
		Username: "alice",
		Password: passwordHash,
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
	if claims["session"] == "" {
		t.Fatalf("expected token to include a session fingerprint")
	}
	if claims["session"] == passwordHash {
		t.Fatalf("expected session fingerprint not to expose password hash")
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

func TestGetUserInfoUsesAuthCreatedAtForRegisterDays(t *testing.T) {
	db := openAuthSecurityTestDB(t)
	connection.CreateSubscriptionTable(db)

	userCreatedAt := time.Now().UTC().Add(-48 * time.Hour).Format("2006-01-02 15:04:05")
	quotaCreatedAt := time.Now().UTC().Add(-30 * 24 * time.Hour).Format("2006-01-02 15:04:05")
	if _, err := globals.ExecDb(db, `
		INSERT INTO auth (username, password, email, bind_id, token, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, "alice", utils.Sha2Encrypt("password"), "alice@example.com", 1001, "alice-token", userCreatedAt); err != nil {
		t.Fatalf("insert auth user: %v", err)
	}

	user := &User{Username: "alice"}
	if _, err := globals.ExecDb(db, `
		INSERT INTO quota (user_id, quota, used, created_at)
		VALUES (?, ?, ?, ?)
	`, user.GetID(db), 100, 12, quotaCreatedAt); err != nil {
		t.Fatalf("insert quota: %v", err)
	}

	info, err := user.GetUserInfo(db)
	if err != nil {
		t.Fatalf("get user info: %v", err)
	}

	if info.RegisterDays < 1.9 || info.RegisterDays > 2.1 {
		t.Fatalf("expected register days to use auth.created_at, got %.2f", info.RegisterDays)
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

func TestPasswordChangeInvalidatesSessionToken(t *testing.T) {
	db := openAuthSecurityTestDB(t)

	var oldHash string
	if err := globals.QueryRowDb(db, "SELECT password FROM auth WHERE username = ?", "root").Scan(&oldHash); err != nil {
		t.Fatalf("fetch root password hash: %v", err)
	}

	user := &User{
		Username: "root",
		Password: oldHash,
	}
	token, err := user.GenerateToken()
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	c := newAuthSecurityTestContext(t, db)
	if parsed := ParseToken(c, token); parsed == nil {
		t.Fatalf("expected token to validate before password change")
	}

	if err := user.UpdatePassword(db, utils.GetCacheFromContext(c), "newsecret123"); err != nil {
		t.Fatalf("update password: %v", err)
	}

	if parsed := ParseToken(c, token); parsed != nil {
		t.Fatalf("expected old token to be rejected after password change")
	}

	nextToken, err := user.GenerateToken()
	if err != nil {
		t.Fatalf("generate next token: %v", err)
	}
	if parsed := ParseToken(c, nextToken); parsed == nil {
		t.Fatalf("expected fresh token to validate after password change")
	}
}

func TestTokenWithoutSessionFingerprintIsRejected(t *testing.T) {
	db := openAuthSecurityTestDB(t)

	instance := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": "root",
		"exp":      time.Now().Add(time.Hour).Unix(),
	})
	token, err := instance.SignedString([]byte(viper.GetString("secret")))
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	c := newAuthSecurityTestContext(t, db)
	if parsed := ParseToken(c, token); parsed != nil {
		t.Fatalf("expected token without session fingerprint to be rejected")
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

func TestCanEnableModelRequiresMinimumBillingQuota(t *testing.T) {
	db := openAuthSecurityTestDB(t)
	user := GetUserByName(db, "root")
	if user == nil {
		t.Fatalf("expected root user")
	}

	if !user.SetQuota(db, 0) {
		t.Fatalf("set quota")
	}

	previousCharge := channel.ChargeInstance
	channel.ChargeInstance = &channel.ChargeManager{
		Models: map[string]*channel.Charge{
			"paid-model": {
				Type:   globals.TimesBilling,
				Output: 9,
			},
		},
	}
	t.Cleanup(func() {
		channel.ChargeInstance = previousCharge
	})

	err := CanEnableModel(db, user, "paid-model", nil)
	if err == nil {
		t.Fatalf("expected paid model to be blocked with zero quota")
	}
	if !strings.Contains(err.Error(), "user quota is not enough error") {
		t.Fatalf("expected not enough quota error, got %q", err.Error())
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
