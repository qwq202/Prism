package auth

import (
	"chat/channel"
	"chat/globals"
	"chat/utils"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type PasskeyCredentialInfo struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

type passkeyRegistrationOptions struct {
	PublicKey passkeyPublicKeyCredentialCreationOptions `json:"publicKey"`
}

type passkeyPublicKeyCredentialCreationOptions struct {
	Challenge              string                           `json:"challenge"`
	RP                     passkeyRelyingParty              `json:"rp"`
	User                   passkeyUserEntity                `json:"user"`
	PubKeyCredParams       []passkeyCredentialParameter     `json:"pubKeyCredParams"`
	Timeout                int                              `json:"timeout"`
	AuthenticatorSelection passkeyAuthenticatorSelection    `json:"authenticatorSelection"`
	Attestation            string                           `json:"attestation"`
	ExcludeCredentials     []passkeyPublicKeyCredentialHint `json:"excludeCredentials"`
}

type passkeyRelyingParty struct {
	Name string `json:"name"`
	ID   string `json:"id,omitempty"`
}

type passkeyUserEntity struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

type passkeyCredentialParameter struct {
	Type string `json:"type"`
	Alg  int    `json:"alg"`
}

type passkeyAuthenticatorSelection struct {
	AuthenticatorAttachment string `json:"authenticatorAttachment,omitempty"`
	UserVerification        string `json:"userVerification"`
}

type passkeyPublicKeyCredentialHint struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type passkeyRegistrationForm struct {
	Name              string   `json:"name"`
	ID                string   `json:"id" binding:"required"`
	RawID             string   `json:"raw_id" binding:"required"`
	Type              string   `json:"type" binding:"required"`
	ClientDataJSON    string   `json:"client_data_json" binding:"required"`
	AttestationObject string   `json:"attestation_object" binding:"required"`
	Transports        []string `json:"transports"`
}

type passkeyClientData struct {
	Type      string `json:"type"`
	Challenge string `json:"challenge"`
	Origin    string `json:"origin"`
}

func passkeyBase64(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func decodePasskeyBase64(data string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(strings.TrimSpace(data))
}

func randomPasskeyChallenge() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return passkeyBase64(buf), nil
}

func passkeyChallengeKey(userID int64) string {
	return fmt.Sprintf("nio:passkey:challenge:%d", userID)
}

func listPasskeyCredentials(db *sql.DB, userID int64) ([]PasskeyCredentialInfo, error) {
	rows, err := globals.QueryDb(db, `
		SELECT id, COALESCE(name, ''), COALESCE(created_at, '')
		FROM passkey_credential
		WHERE user_id = ?
		ORDER BY id DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	credentials := make([]PasskeyCredentialInfo, 0)
	for rows.Next() {
		var item PasskeyCredentialInfo
		if err := rows.Scan(&item.ID, &item.Name, &item.CreatedAt); err != nil {
			return nil, err
		}
		credentials = append(credentials, item)
	}

	return credentials, rows.Err()
}

func listPasskeyCredentialIDs(db *sql.DB, userID int64) ([]passkeyPublicKeyCredentialHint, error) {
	rows, err := globals.QueryDb(db, `
		SELECT credential_id
		FROM passkey_credential
		WHERE user_id = ?
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	credentials := make([]passkeyPublicKeyCredentialHint, 0)
	for rows.Next() {
		var credentialID string
		if err := rows.Scan(&credentialID); err != nil {
			return nil, err
		}
		if credentialID = strings.TrimSpace(credentialID); credentialID != "" {
			credentials = append(credentials, passkeyPublicKeyCredentialHint{
				Type: "public-key",
				ID:   credentialID,
			})
		}
	}

	return credentials, rows.Err()
}

func passkeyOriginAllowed(requestOrigin, clientOrigin string) bool {
	clientOrigin = strings.TrimSuffix(strings.TrimSpace(clientOrigin), "/")
	if clientOrigin == "" {
		return false
	}

	parsed, err := url.Parse(clientOrigin)
	if err != nil || parsed.Hostname() == "" {
		return false
	}

	scheme := strings.ToLower(parsed.Scheme)
	host := strings.ToLower(parsed.Hostname())
	isLocalhost := host == "localhost" || host == "127.0.0.1" || host == "::1"
	if scheme != "https" && !isLocalhost && !channel.SystemInstance.AllowPasskeyInsecureOrigin() {
		return false
	}

	origins := channel.SystemInstance.GetPasskeyOrigins()
	if len(origins) > 0 {
		for _, origin := range origins {
			if strings.EqualFold(strings.TrimSuffix(origin, "/"), clientOrigin) {
				return true
			}
		}
		return false
	}

	rpID := strings.ToLower(channel.SystemInstance.GetPasskeyRPID())
	if rpID != "" {
		return host == rpID || strings.HasSuffix(host, "."+rpID)
	}

	requestOrigin = strings.TrimSuffix(strings.TrimSpace(requestOrigin), "/")
	return requestOrigin == "" || strings.EqualFold(requestOrigin, clientOrigin)
}

func passkeyDisabledError() error {
	if channel.SystemInstance == nil || !channel.SystemInstance.IsPasskeyEnabled() {
		return errors.New("passkey authentication is not enabled")
	}

	return nil
}

func ListPasskeysAPI(c *gin.Context) {
	user := RequireAuth(c)
	if user == nil {
		return
	}

	db := utils.GetDBFromContext(c)
	credentials, err := listPasskeyCredentials(db, user.GetID(db))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":      true,
		"enabled":     channel.SystemInstance != nil && channel.SystemInstance.IsPasskeyEnabled(),
		"credentials": credentials,
	})
}

func CreatePasskeyRegistrationOptionsAPI(c *gin.Context) {
	user := RequireAuth(c)
	if user == nil {
		return
	}
	if err := passkeyDisabledError(); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	db := utils.GetDBFromContext(c)
	userID := user.GetID(db)
	challenge, err := randomPasskeyChallenge()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	cache := utils.GetCacheFromContext(c)
	cache.Set(c, passkeyChallengeKey(userID), challenge, 5*time.Minute)

	excludeCredentials, err := listPasskeyCredentialIDs(db, userID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	attachment := channel.SystemInstance.GetPasskeyAuthenticatorAttachment()
	if attachment == "any" {
		attachment = ""
	}

	options := passkeyRegistrationOptions{
		PublicKey: passkeyPublicKeyCredentialCreationOptions{
			Challenge: challenge,
			RP: passkeyRelyingParty{
				Name: channel.SystemInstance.GetPasskeyRPDisplayName(),
				ID:   channel.SystemInstance.GetPasskeyRPID(),
			},
			User: passkeyUserEntity{
				ID:          passkeyBase64([]byte(strconv.FormatInt(userID, 10))),
				Name:        user.Username,
				DisplayName: user.Username,
			},
			PubKeyCredParams: []passkeyCredentialParameter{
				{Type: "public-key", Alg: -7},
				{Type: "public-key", Alg: -257},
			},
			Timeout: 60000,
			AuthenticatorSelection: passkeyAuthenticatorSelection{
				AuthenticatorAttachment: attachment,
				UserVerification:        channel.SystemInstance.GetPasskeyUserVerification(),
			},
			Attestation:        "none",
			ExcludeCredentials: excludeCredentials,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"status": true,
		"data":   options,
	})
}

func RegisterPasskeyAPI(c *gin.Context) {
	user := RequireAuth(c)
	if user == nil {
		return
	}
	if err := passkeyDisabledError(); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	var form passkeyRegistrationForm
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  "bad request",
		})
		return
	}

	if form.Type != "public-key" {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  "invalid passkey credential type",
		})
		return
	}

	clientDataBytes, err := decodePasskeyBase64(form.ClientDataJSON)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  "invalid passkey client data",
		})
		return
	}

	var clientData passkeyClientData
	if err := json.Unmarshal(clientDataBytes, &clientData); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  "invalid passkey client data",
		})
		return
	}

	db := utils.GetDBFromContext(c)
	userID := user.GetID(db)
	cache := utils.GetCacheFromContext(c)
	challenge, err := cache.Get(c, passkeyChallengeKey(userID)).Result()
	if err != nil || challenge == "" || challenge != clientData.Challenge {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  "invalid passkey challenge",
		})
		return
	}

	if clientData.Type != "webauthn.create" || !passkeyOriginAllowed(c.GetHeader("Origin"), clientData.Origin) {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  "invalid passkey origin",
		})
		return
	}

	credentialID := strings.TrimSpace(form.RawID)
	if credentialID == "" {
		credentialID = strings.TrimSpace(form.ID)
	}
	if _, err := decodePasskeyBase64(credentialID); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  "invalid passkey credential id",
		})
		return
	}
	if _, err := decodePasskeyBase64(form.AttestationObject); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  "invalid passkey attestation",
		})
		return
	}

	name := strings.TrimSpace(form.Name)
	if name == "" {
		name = fmt.Sprintf("Passkey %s", time.Now().Format("2006-01-02 15:04"))
	}

	if _, err := globals.ExecDb(db, `
		INSERT INTO passkey_credential (
			user_id, credential_id, name, transports, attestation_object, client_data_json
		) VALUES (?, ?, ?, ?, ?, ?)
	`, userID, credentialID, utils.Extract(name, 255, ""), strings.Join(form.Transports, ","), form.AttestationObject, form.ClientDataJSON); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	cache.Del(c, passkeyChallengeKey(userID))
	c.JSON(http.StatusOK, gin.H{"status": true})
}

func DeletePasskeyAPI(c *gin.Context) {
	user := RequireAuth(c)
	if user == nil {
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  "bad request",
		})
		return
	}

	db := utils.GetDBFromContext(c)
	if _, err := globals.ExecDb(db, `
		DELETE FROM passkey_credential
		WHERE id = ? AND user_id = ?
	`, id, user.GetID(db)); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": true})
}
