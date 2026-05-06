package auth

import (
	"chat/globals"
	"chat/utils"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

const hashedApiKeyPrefix = "ak-sha256:"

func hashApiKey(key string) string {
	return hashedApiKeyPrefix + utils.HmacSha256(key, viper.GetString("secret"))
}

func isHashedApiKey(value string) bool {
	return strings.HasPrefix(value, hashedApiKeyPrefix)
}

func findUserByApiKey(db *sql.DB, key string, hashed bool) (*User, error) {
	var user User
	if err := globals.QueryRowDb(db, `
			SELECT auth.id, auth.username, auth.password FROM auth
			INNER JOIN apikey ON auth.id = apikey.user_id
			WHERE apikey.api_key = ?
			`, key).Scan(&user.ID, &user.Username, &user.Password); err != nil {
		return nil, err
	}

	if !hashed {
		if _, err := globals.ExecDb(db, "UPDATE apikey SET api_key = ? WHERE user_id = ?", hashApiKey(key), user.ID); err != nil {
			globals.Warn(fmt.Sprintf("failed to migrate api key for user %s: %s", user.Username, err.Error()))
		}
	}

	return &user, nil
}

func ParseApiKeyByHash(db *sql.DB, key string) *User {
	key = strings.TrimSpace(key)
	if len(key) == 0 {
		return nil
	}

	if user, err := findUserByApiKey(db, hashApiKey(key), true); err == nil {
		return user
	}

	if user, err := findUserByApiKey(db, key, false); err == nil {
		return user
	}

	return nil
}

func (u *User) CreateApiKey(db *sql.DB) string {
	key := fmt.Sprintf("sk-%s", utils.GenerateChar(64))
	if _, err := globals.ExecDb(db, "INSERT INTO apikey (user_id, api_key) VALUES (?, ?)", u.GetID(db), hashApiKey(key)); err != nil {
		return ""
	}
	return key
}

func (u *User) GetApiKey(db *sql.DB) string {
	var key string
	if err := globals.QueryRowDb(db, "SELECT api_key FROM apikey WHERE user_id = ?", u.GetID(db)).Scan(&key); err != nil {
		return u.CreateApiKey(db)
	}

	if isHashedApiKey(key) {
		return ""
	}

	if strings.HasPrefix(key, "sk-") {
		if _, err := globals.ExecDb(db, "UPDATE apikey SET api_key = ? WHERE user_id = ?", hashApiKey(key), u.GetID(db)); err != nil {
			globals.Warn(fmt.Sprintf("failed to migrate api key for user %s: %s", u.Username, err.Error()))
		}
	}

	return key
}

func (u *User) ResetApiKey(db *sql.DB) (string, error) {
	if _, err := globals.ExecDb(db, "DELETE FROM apikey WHERE user_id = ?", u.GetID(db)); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	return u.CreateApiKey(db), nil
}
