package auth

import (
	"chat/channel"
	"chat/globals"
	"chat/utils"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
)

func jwtSigningKey(token *jwt.Token) (interface{}, error) {
	if token.Method == nil || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
		return nil, fmt.Errorf("unexpected jwt signing method: %v", token.Header["alg"])
	}

	return []byte(viper.GetString("secret")), nil
}

func parseJWTExpiration(value interface{}) (int64, bool) {
	switch exp := value.(type) {
	case float64:
		return int64(exp), true
	case json.Number:
		value, err := exp.Int64()
		return value, err == nil
	default:
		return 0, false
	}
}

func parseTokenClaims(claims jwt.MapClaims, now time.Time) (*User, bool) {
	exp, ok := parseJWTExpiration(claims["exp"])
	if !ok || exp < now.Unix() {
		return nil, false
	}

	username, ok := claims["username"].(string)
	username = strings.TrimSpace(username)
	if !ok || username == "" {
		return nil, false
	}

	user := &User{
		Username: username,
	}
	if password, ok := claims["password"].(string); ok {
		user.Password = password
	}
	if session, ok := claims["session"].(string); ok {
		user.SessionHash = strings.TrimSpace(session)
	}

	return user, true
}

func ParseToken(c *gin.Context, token string) *User {
	instance, err := jwt.Parse(token, jwtSigningKey)
	if err != nil {
		return nil
	}
	if claims, ok := instance.Claims.(jwt.MapClaims); ok && instance.Valid {
		user, ok := parseTokenClaims(claims, time.Now())
		if !ok {
			return nil
		}
		if !user.Validate(c) {
			return nil
		}
		return user
	}
	return nil
}

func getCode(c *gin.Context, cache *redis.Client, email string) string {
	code, err := cache.Get(c, fmt.Sprintf("nio:otp:%s", email)).Result()
	if err != nil {
		return ""
	}
	return code
}

func constantTimeStringEqual(expected string, actual string) bool {
	expectedHash := sha256.Sum256([]byte(expected))
	actualHash := sha256.Sum256([]byte(actual))
	return subtle.ConstantTimeCompare(expectedHash[:], actualHash[:]) == 1 && len(expected) == len(actual)
}

func checkCode(c *gin.Context, cache *redis.Client, email, code string) bool {
	storage := getCode(c, cache, email)
	if len(storage) == 0 {
		return false
	}

	if !constantTimeStringEqual(storage, code) {
		return false
	}

	cache.Del(c, fmt.Sprintf("nio:otp:%s", email))
	return true
}

func setCode(c *gin.Context, cache *redis.Client, email, code string) {
	cache.Set(c, fmt.Sprintf("nio:otp:%s", email), code, 5*time.Minute)
}

func generateCode(c *gin.Context, cache *redis.Client, email string) string {
	code := utils.GenerateCode(6)
	setCode(c, cache, email, code)
	return code
}

func Verify(c *gin.Context, email string, checkout bool) error {
	cache := utils.GetCacheFromContext(c)
	code := generateCode(c, cache, email)

	if checkout {
		if err := channel.SystemInstance.IsValidMail(email); err != nil {
			return err
		}
	}

	return channel.SystemInstance.SendVerifyMail(email, code)
}

func SignUp(c *gin.Context, form RegisterForm) (string, error) {
	db := utils.GetDBFromContext(c)
	cache := utils.GetCacheFromContext(c)

	username := strings.TrimSpace(form.Username)
	password := strings.TrimSpace(form.Password)
	email := strings.TrimSpace(form.Email)
	code := strings.TrimSpace(form.Code)

	enableVerify := channel.SystemInstance.IsMailValid()

	if !utils.All(
		validateUsername(username),
		validatePassword(password),
		validateEmail(email),
		!enableVerify || validateCode(code),
	) {
		return "", errors.New("invalid username/password/email format")
	}

	if err := channel.SystemInstance.IsValidMail(form.Email); err != nil {
		return "", err
	}

	if IsUserExist(db, username) {
		return "", fmt.Errorf("username is already taken, please try another one username (your current username: %s)", username)
	}

	if IsEmailExist(db, email) {
		return "", fmt.Errorf("email is already taken, please try another one email (your current email: %s)", email)
	}

	if enableVerify && !checkCode(c, cache, email, code) {
		return "", errors.New("invalid email verification code")
	}

	hash, err := utils.HashPassword(password)
	if err != nil {
		return "", err
	}

	user := &User{
		Username: username,
		Password: hash,
		Email:    email,
		BindID:   getMaxBindId(db) + 1,
		Token:    utils.Sha2Encrypt(email + username),
	}

	if _, err := globals.ExecDb(db, `
			INSERT INTO auth (username, password, email, bind_id, token)
			VALUES (?, ?, ?, ?, ?)
			`, user.Username, user.Password, user.Email, user.BindID, user.Token); err != nil {
		return "", err
	}

	user.CreateInitialQuota(db)
	return user.GenerateToken()
}

func Login(c *gin.Context, form LoginForm) (string, error) {
	db := utils.GetDBFromContext(c)
	username := strings.TrimSpace(form.Username)
	password := strings.TrimSpace(form.Password)

	if !utils.All(
		validateUsernameOrEmail(username),
		validatePassword(password),
	) {
		return "", errors.New("invalid username or password format")
	}

	var user User
	if err := globals.QueryRowDb(db, `
			SELECT auth.id, auth.username, auth.password FROM auth 
			WHERE auth.username = ? OR auth.email = ?
			`, username, username).Scan(&user.ID, &user.Username, &user.Password); err != nil {
		return "", errors.New("invalid username or password")
	}

	if ok, upgrade := utils.VerifyPassword(password, user.Password); !ok {
		return "", errors.New("invalid username or password")
	} else if upgrade {
		if err := user.UpdatePassword(db, utils.GetCacheFromContext(c), password); err != nil {
			globals.Warn(fmt.Sprintf("failed to upgrade password hash for user %s: %s", user.Username, err.Error()))
		}
	}

	if user.IsBanned(db) {
		return "", errors.New("current user is banned")
	}

	return user.GenerateToken()
}

func DeepLogin(c *gin.Context, token string) (string, error) {
	if !useDeeptrain() {
		return "", errors.New("deeptrain mode is disabled")
	}

	user := Validate(token)
	if user == nil {
		return "", errors.New("cannot validate access token")
	}

	db := utils.GetDBFromContext(c)
	if !IsUserExist(db, user.Username) {
		if globals.CloseRegistration {
			return "", errors.New("this site is not open for registration")
		}

		// register
		password := utils.GenerateChar(64)
		hash, err := utils.HashPassword(password)
		if err != nil {
			return "", err
		}

		_, err = globals.ExecDb(db, "INSERT INTO auth (bind_id, username, token, password) VALUES (?, ?, ?, ?)",
			user.ID, user.Username, utils.Extract(token, 255, ""), hash)
		if err != nil {
			return "", err
		}

		u := &User{
			Username: user.Username,
			Password: hash,
		}

		u.CreateInitialQuota(db)
		return u.GenerateToken()
	}

	// login
	_ = globals.QueryRowDb(db, "UPDATE auth SET token = ? WHERE username = ?", utils.Extract(token, 255, ""), user.Username)
	var password string
	err := globals.QueryRowDb(db, "SELECT password FROM auth WHERE username = ?", user.Username).Scan(&password)
	if err != nil {
		return "", err
	}
	u := &User{
		Username: user.Username,
		Password: password,
	}

	if u.IsBanned(db) {
		return "", errors.New("current user is banned")
	}

	return u.GenerateToken()
}

func Reset(c *gin.Context, form ResetForm) error {
	db := utils.GetDBFromContext(c)
	cache := utils.GetCacheFromContext(c)

	email := strings.TrimSpace(form.Email)
	code := strings.TrimSpace(form.Code)
	password := strings.TrimSpace(form.Password)

	if !utils.All(
		validateEmail(email),
		validateCode(code),
		validatePassword(password),
	) {
		return errors.New("invalid email/code/password format")
	}

	if !IsEmailExist(db, email) {
		return errors.New("email is not registered")
	}

	if !checkCode(c, cache, email, code) {
		return errors.New("invalid email verification code")
	}

	user := GetUserByEmail(db, email)
	if user == nil {
		return errors.New("cannot find user by email")
	}

	if err := user.UpdatePassword(db, cache, password); err != nil {
		return err
	}

	cache.Del(c, fmt.Sprintf("nio:otp:%s", email))

	return nil
}

func (u *User) UpdatePassword(db *sql.DB, cache *redis.Client, password string) error {
	hash, err := utils.HashPassword(password)
	if err != nil {
		return err
	}

	if _, err := globals.ExecDb(db, `
			UPDATE auth SET password = ? WHERE id = ?
			`, hash, u.GetID(db)); err != nil {
		return err
	}

	u.Password = hash
	if cache != nil {
		cache.Del(context.Background(), fmt.Sprintf("nio:user:%s", u.Username))
	}

	return nil
}

func (u *User) UpdateEmail(c *gin.Context, db *sql.DB, cache *redis.Client, email, code string) error {
	email = strings.TrimSpace(email)
	code = strings.TrimSpace(code)

	if !utils.All(
		validateEmail(email),
		validateCode(code),
	) {
		return errors.New("invalid email/code format")
	}

	if err := channel.SystemInstance.IsValidMail(email); err != nil {
		return err
	}

	if strings.EqualFold(email, strings.TrimSpace(u.GetEmail(db))) {
		return errors.New("new email is the same as current email")
	}

	if IsEmailExist(db, email) {
		return errors.New("email is already taken")
	}

	if !checkCode(c, cache, email, code) {
		return errors.New("invalid email verification code")
	}

	if _, err := globals.ExecDb(db, `
			UPDATE auth SET email = ? WHERE id = ?
			`, email, u.GetID(db)); err != nil {
		return err
	}

	u.Email = email
	return nil
}

func (u *User) UpdatePasswordWithEmailCode(c *gin.Context, db *sql.DB, cache *redis.Client, password, code string) error {
	password = strings.TrimSpace(password)
	code = strings.TrimSpace(code)
	email := strings.TrimSpace(u.GetEmail(db))

	if !utils.All(
		validateEmail(email),
		validateCode(code),
		validatePassword(password),
	) {
		return errors.New("invalid email/code/password format")
	}

	if !checkCode(c, cache, email, code) {
		return errors.New("invalid email verification code")
	}

	return u.UpdatePassword(db, cache, password)
}

func (u *User) UpdatePasswordWithOldPassword(db *sql.DB, cache *redis.Client, oldPassword, password string) error {
	oldPassword = strings.TrimSpace(oldPassword)
	password = strings.TrimSpace(password)

	if !utils.All(
		validatePassword(oldPassword),
		validatePassword(password),
	) {
		return errors.New("invalid password format")
	}

	var stored string
	if err := globals.QueryRowDb(db, "SELECT password FROM auth WHERE id = ?", u.GetID(db)).Scan(&stored); err != nil {
		return errors.New("invalid old password")
	}

	if ok, _ := utils.VerifyPassword(oldPassword, stored); !ok {
		return errors.New("invalid old password")
	}

	return u.UpdatePassword(db, cache, password)
}

func (u *User) Validate(c *gin.Context) bool {
	if u.Username == "" {
		return false
	}
	cache := utils.GetCacheFromContext(c)
	db := utils.GetDBFromContext(c)

	if u.Password == "" && u.SessionHash == "" {
		return false
	}

	storedPassword := ""
	if cache != nil {
		if password, err := cache.Get(c, fmt.Sprintf("nio:user:%s", u.Username)).Result(); err == nil && len(password) > 0 {
			storedPassword = password
		}
	}
	if storedPassword == "" {
		if err := globals.QueryRowDb(db, "SELECT password FROM auth WHERE username = ?", u.Username).Scan(&storedPassword); err != nil {
			if err != sql.ErrNoRows {
				globals.Warn(fmt.Sprintf("validate user password error: %s", err.Error()))
			}
			return false
		}
	}

	if u.Password != "" && !constantTimeStringEqual(storedPassword, u.Password) {
		return false
	}
	if u.SessionHash != "" && !constantTimeStringEqual(sessionHashForPassword(storedPassword), u.SessionHash) {
		return false
	}

	if cache != nil {
		cache.Set(c, fmt.Sprintf("nio:user:%s", u.Username), storedPassword, 30*time.Minute)
	}

	return !u.IsBanned(db)
}

func sessionHashForPassword(password string) string {
	password = strings.TrimSpace(password)
	if password == "" {
		return ""
	}

	mac := hmac.New(sha256.New, []byte(viper.GetString("secret")))
	_, _ = mac.Write([]byte(password))
	return hex.EncodeToString(mac.Sum(nil))
}

func (u *User) GenerateToken() (string, error) {
	claims := jwt.MapClaims{
		"username": u.Username,
		"exp":      time.Now().Add(time.Hour * 24 * 30).Unix(),
	}
	if session := sessionHashForPassword(u.Password); session != "" {
		claims["session"] = session
	}

	instance := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := instance.SignedString([]byte(viper.GetString("secret")))
	if err != nil {
		return "", err
	} else if token == "" {
		return "", errors.New("unable to generate token")
	}
	return token, nil
}

func (u *User) GenerateTokenSafe(db *sql.DB) (string, error) {
	if len(u.Username) == 0 || len(u.Password) == 0 {
		if u.ID > 0 {
			if err := globals.QueryRowDb(db, "SELECT username, password FROM auth WHERE id = ?", u.ID).Scan(&u.Username, &u.Password); err != nil {
				return "", err
			}
		} else if u.Username != "" {
			if err := globals.QueryRowDb(db, "SELECT password FROM auth WHERE username = ?", u.Username).Scan(&u.Password); err != nil {
				return "", err
			}
		} else {
			return "", errors.New("missing user identity")
		}
	}
	if len(u.Username) == 0 {
		if err := globals.QueryRowDb(db, "SELECT username FROM auth WHERE id = ?", u.ID).Scan(&u.Username); err != nil {
			return "", err
		}
	}

	return u.GenerateToken()
}
