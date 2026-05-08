package auth

import (
	"bytes"
	"chat/channel"
	"chat/globals"
	"chat/utils"
	"crypto/rand"
	"crypto/sha256"
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
	Type       string   `json:"type"`
	ID         string   `json:"id"`
	Transports []string `json:"transports,omitempty"`
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

type passkeyLoginOptionsForm struct {
	Username string `json:"username" binding:"required"`
}

type passkeyAuthenticationOptions struct {
	PublicKey passkeyPublicKeyCredentialRequestOptions `json:"publicKey"`
}

type passkeyPublicKeyCredentialRequestOptions struct {
	Challenge        string                           `json:"challenge"`
	Timeout          int                              `json:"timeout"`
	RPID             string                           `json:"rpId,omitempty"`
	AllowCredentials []passkeyPublicKeyCredentialHint `json:"allowCredentials"`
	UserVerification string                           `json:"userVerification"`
}

type passkeyLoginForm struct {
	Username          string `json:"username" binding:"required"`
	ID                string `json:"id" binding:"required"`
	RawID             string `json:"raw_id" binding:"required"`
	Type              string `json:"type" binding:"required"`
	AuthenticatorData string `json:"authenticator_data" binding:"required"`
	ClientDataJSON    string `json:"client_data_json" binding:"required"`
	Signature         string `json:"signature" binding:"required"`
	UserHandle        string `json:"user_handle"`
}

type passkeyClientData struct {
	Type      string `json:"type"`
	Challenge string `json:"challenge"`
	Origin    string `json:"origin"`
}

type passkeyStoredCredential struct {
	ID                int64
	UserID            int64
	Username          string
	CredentialID      string
	Transports        string
	AttestationObject string
	PublicKey         string
	SignCount         uint32
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

func passkeyLoginChallengeKey(userID int64) string {
	return fmt.Sprintf("nio:passkey:login:%d", userID)
}

func passkeyUserByUsernameOrEmail(db *sql.DB, usernameOrEmail string) (*User, error) {
	usernameOrEmail = strings.TrimSpace(usernameOrEmail)
	if usernameOrEmail == "" {
		return nil, errors.New("invalid username or email")
	}

	var user User
	if err := globals.QueryRowDb(db, `
		SELECT id, username
		FROM auth
		WHERE username = ? OR email = ?
	`, usernameOrEmail, usernameOrEmail).Scan(&user.ID, &user.Username); err != nil {
		return nil, errors.New("user passkey not found")
	}

	return &user, nil
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
		SELECT credential_id, COALESCE(transports, '')
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
		var transports string
		if err := rows.Scan(&credentialID, &transports); err != nil {
			return nil, err
		}
		if credentialID = strings.TrimSpace(credentialID); credentialID != "" {
			credentials = append(credentials, passkeyPublicKeyCredentialHint{
				Type:       "public-key",
				ID:         credentialID,
				Transports: splitPasskeyTransports(transports),
			})
		}
	}

	return credentials, rows.Err()
}

func splitPasskeyTransports(value string) []string {
	parts := strings.Split(value, ",")
	transports := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			transports = append(transports, part)
		}
	}

	return transports
}

func getPasskeyStoredCredential(db *sql.DB, credentialID string, userID int64) (*passkeyStoredCredential, error) {
	var item passkeyStoredCredential
	err := globals.QueryRowDb(db, `
		SELECT
			p.id, p.user_id, a.username, p.credential_id, COALESCE(p.transports, ''),
			COALESCE(p.attestation_object, ''), COALESCE(p.public_key, ''), COALESCE(p.sign_count, 0)
		FROM passkey_credential p
		JOIN auth a ON a.id = p.user_id
		WHERE p.credential_id = ? AND (? = 0 OR p.user_id = ?)
	`, credentialID, userID, userID).Scan(
		&item.ID,
		&item.UserID,
		&item.Username,
		&item.CredentialID,
		&item.Transports,
		&item.AttestationObject,
		&item.PublicKey,
		&item.SignCount,
	)
	if err != nil {
		return nil, errors.New("user passkey not found")
	}

	return &item, nil
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
	credentialIDBytes, err := decodePasskeyBase64(credentialID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  "invalid passkey credential id",
		})
		return
	}

	attestationObjectBytes, err := decodePasskeyBase64(form.AttestationObject)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  "invalid passkey attestation",
		})
		return
	}

	attestedCredential, err := parsePasskeyAttestationObject(attestationObjectBytes)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}
	if !bytes.Equal(attestedCredential.CredentialID, credentialIDBytes) {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  "passkey credential id mismatch",
		})
		return
	}
	if !passkeyRPIDHashMatches(attestedCredential.RPIDHash, clientData.Origin) {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  "invalid passkey relying party",
		})
		return
	}

	name := strings.TrimSpace(form.Name)
	if name == "" {
		name = fmt.Sprintf("Passkey %s", time.Now().Format("2006-01-02 15:04"))
	}

	if _, err := globals.ExecDb(db, `
		INSERT INTO passkey_credential (
			user_id, credential_id, name, transports, attestation_object, client_data_json, public_key, sign_count
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, userID, credentialID, utils.Extract(name, 255, ""), strings.Join(form.Transports, ","), form.AttestationObject, form.ClientDataJSON, passkeyBase64(attestedCredential.CredentialPublicKey), attestedCredential.SignCount); err != nil {
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

func CreatePasskeyLoginOptionsAPI(c *gin.Context) {
	if useDeeptrain() {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  "this api is not available for deeptrain mode",
		})
		return
	}
	if err := passkeyDisabledError(); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	var form passkeyLoginOptionsForm
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  "bad request",
		})
		return
	}

	db := utils.GetDBFromContext(c)
	user, err := passkeyUserByUsernameOrEmail(db, form.Username)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	allowCredentials, err := listPasskeyCredentialIDs(db, user.ID)
	if err != nil || len(allowCredentials) == 0 {
		if err == nil {
			err = errors.New("user passkey not found")
		}
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	challenge, err := randomPasskeyChallenge()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	cache := utils.GetCacheFromContext(c)
	cache.Set(c, passkeyLoginChallengeKey(user.ID), challenge, 5*time.Minute)

	options := passkeyAuthenticationOptions{
		PublicKey: passkeyPublicKeyCredentialRequestOptions{
			Challenge:        challenge,
			Timeout:          60000,
			RPID:             channel.SystemInstance.GetPasskeyRPID(),
			AllowCredentials: allowCredentials,
			UserVerification: channel.SystemInstance.GetPasskeyUserVerification(),
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"status": true,
		"data":   options,
	})
}

func resolvePasskeyPublicKey(db *sql.DB, credential *passkeyStoredCredential) ([]byte, error) {
	if strings.TrimSpace(credential.PublicKey) != "" {
		return decodePasskeyBase64(credential.PublicKey)
	}

	attestationObjectBytes, err := decodePasskeyBase64(credential.AttestationObject)
	if err != nil {
		return nil, err
	}
	attestedCredential, err := parsePasskeyAttestationObject(attestationObjectBytes)
	if err != nil {
		return nil, err
	}

	publicKey := passkeyBase64(attestedCredential.CredentialPublicKey)
	if _, err := globals.ExecDb(db, `
		UPDATE passkey_credential
		SET public_key = ?, sign_count = ?
		WHERE id = ?
	`, publicKey, attestedCredential.SignCount, credential.ID); err != nil {
		return nil, err
	}

	credential.PublicKey = publicKey
	credential.SignCount = attestedCredential.SignCount
	return attestedCredential.CredentialPublicKey, nil
}

func VerifyPasskeyLoginAPI(c *gin.Context) {
	if useDeeptrain() {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  "this api is not available for deeptrain mode",
		})
		return
	}
	if err := passkeyDisabledError(); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	var form passkeyLoginForm
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

	db := utils.GetDBFromContext(c)
	user, err := passkeyUserByUsernameOrEmail(db, form.Username)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  err.Error(),
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

	storedCredential, err := getPasskeyStoredCredential(db, credentialID, user.ID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  err.Error(),
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

	cache := utils.GetCacheFromContext(c)
	challenge, err := cache.Get(c, passkeyLoginChallengeKey(user.ID)).Result()
	if err != nil || challenge == "" || challenge != clientData.Challenge {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  "invalid passkey challenge",
		})
		return
	}
	if clientData.Type != "webauthn.get" || !passkeyOriginAllowed(c.GetHeader("Origin"), clientData.Origin) {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  "invalid passkey origin",
		})
		return
	}

	authenticatorDataBytes, err := decodePasskeyBase64(form.AuthenticatorData)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  "invalid passkey authenticator data",
		})
		return
	}
	assertion, err := parsePasskeyAssertionData(authenticatorDataBytes)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}
	if assertion.Flags&passkeyFlagUserPresent == 0 {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  "passkey user presence is required",
		})
		return
	}
	if channel.SystemInstance.GetPasskeyUserVerification() == "required" && assertion.Flags&passkeyFlagUserVerified == 0 {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  "passkey user verification is required",
		})
		return
	}
	if !passkeyRPIDHashMatches(assertion.RPIDHash, clientData.Origin) {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  "invalid passkey relying party",
		})
		return
	}

	signatureBytes, err := decodePasskeyBase64(form.Signature)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  "invalid passkey signature",
		})
		return
	}

	publicKeyBytes, err := resolvePasskeyPublicKey(db, storedCredential)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	clientDataHash := sha256.Sum256(clientDataBytes)
	signatureBase := append(append([]byte(nil), authenticatorDataBytes...), clientDataHash[:]...)
	if err := passkeyVerifySignature(publicKeyBytes, signatureBase, signatureBytes); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	if assertion.SignCount > storedCredential.SignCount {
		_, _ = globals.ExecDb(db, `
			UPDATE passkey_credential
			SET sign_count = ?
			WHERE id = ?
		`, assertion.SignCount, storedCredential.ID)
	}

	cache.Del(c, passkeyLoginChallengeKey(user.ID))
	if user.IsBanned(db) {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  "current user is banned",
		})
		return
	}

	token, err := (&User{ID: storedCredential.UserID, Username: storedCredential.Username}).GenerateTokenSafe(db)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": true,
		"token":  token,
	})
}
