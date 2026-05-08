package auth

import (
	"bytes"
	"chat/channel"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/binary"
	"errors"
	"math/big"
	"net/url"
	"strings"
)

const (
	passkeyFlagUserPresent  = 0x01
	passkeyFlagUserVerified = 0x04
	passkeyFlagAttestedData = 0x40
)

type passkeyAttestedCredential struct {
	RPIDHash            []byte
	Flags               byte
	SignCount           uint32
	CredentialID        []byte
	CredentialPublicKey []byte
}

type passkeyAssertionData struct {
	RPIDHash  []byte
	Flags     byte
	SignCount uint32
}

type passkeyECDSASignature struct {
	R *big.Int
	S *big.Int
}

func parsePasskeyAttestationObject(attestationObject []byte) (*passkeyAttestedCredential, error) {
	value, _, err := decodePasskeyCBOR(attestationObject)
	if err != nil {
		return nil, err
	}

	m, ok := value.(map[interface{}]interface{})
	if !ok {
		return nil, errors.New("invalid passkey attestation object")
	}

	authData, ok := passkeyMapBytes(m, "authData")
	if !ok {
		return nil, errors.New("missing passkey authenticator data")
	}

	return parsePasskeyAttestedCredentialData(authData)
}

func parsePasskeyAttestedCredentialData(authData []byte) (*passkeyAttestedCredential, error) {
	if len(authData) < 37 {
		return nil, errors.New("invalid passkey authenticator data")
	}

	item := &passkeyAttestedCredential{
		RPIDHash:  append([]byte(nil), authData[:32]...),
		Flags:     authData[32],
		SignCount: binary.BigEndian.Uint32(authData[33:37]),
	}

	if item.Flags&passkeyFlagAttestedData == 0 {
		return nil, errors.New("passkey attested credential data is missing")
	}

	pos := 37 + 16
	if len(authData) < pos+2 {
		return nil, errors.New("invalid passkey attested credential data")
	}

	credentialIDLength := int(binary.BigEndian.Uint16(authData[pos : pos+2]))
	pos += 2
	if credentialIDLength <= 0 || len(authData) < pos+credentialIDLength {
		return nil, errors.New("invalid passkey credential id")
	}

	item.CredentialID = append([]byte(nil), authData[pos:pos+credentialIDLength]...)
	pos += credentialIDLength
	if pos >= len(authData) {
		return nil, errors.New("missing passkey credential public key")
	}

	_, keyLength, err := decodePasskeyCBOR(authData[pos:])
	if err != nil {
		return nil, err
	}
	if keyLength <= 0 || len(authData) < pos+keyLength {
		return nil, errors.New("invalid passkey credential public key")
	}

	item.CredentialPublicKey = append([]byte(nil), authData[pos:pos+keyLength]...)
	return item, nil
}

func parsePasskeyAssertionData(authData []byte) (*passkeyAssertionData, error) {
	if len(authData) < 37 {
		return nil, errors.New("invalid passkey authenticator data")
	}

	return &passkeyAssertionData{
		RPIDHash:  append([]byte(nil), authData[:32]...),
		Flags:     authData[32],
		SignCount: binary.BigEndian.Uint32(authData[33:37]),
	}, nil
}

func passkeyRPIDFromOrigin(clientOrigin string) string {
	parsed, err := url.Parse(strings.TrimSpace(clientOrigin))
	if err != nil {
		return ""
	}
	return strings.ToLower(parsed.Hostname())
}

func passkeyExpectedRPID(clientOrigin string) string {
	rpID := strings.ToLower(channel.SystemInstance.GetPasskeyRPID())
	if rpID != "" {
		return rpID
	}
	return passkeyRPIDFromOrigin(clientOrigin)
}

func passkeyRPIDHashMatches(authDataHash []byte, clientOrigin string) bool {
	rpID := passkeyExpectedRPID(clientOrigin)
	if rpID == "" {
		return false
	}

	hash := sha256.Sum256([]byte(rpID))
	return bytes.Equal(authDataHash, hash[:])
}

func passkeyVerifySignature(coseKey []byte, signatureBase []byte, signature []byte) error {
	value, _, err := decodePasskeyCBOR(coseKey)
	if err != nil {
		return err
	}

	m, ok := value.(map[interface{}]interface{})
	if !ok {
		return errors.New("invalid passkey public key")
	}

	kty, ok := passkeyMapInt(m, int64(1))
	if !ok {
		return errors.New("missing passkey public key type")
	}

	alg, _ := passkeyMapInt(m, int64(3))
	switch {
	case kty == 2 && (alg == 0 || alg == -7):
		return passkeyVerifyECDSA(m, signatureBase, signature)
	case kty == 3 && (alg == 0 || alg == -257):
		return passkeyVerifyRSA(m, signatureBase, signature)
	default:
		return errors.New("unsupported passkey public key algorithm")
	}
}

func passkeyVerifyECDSA(m map[interface{}]interface{}, signatureBase []byte, signature []byte) error {
	crv, ok := passkeyMapInt(m, int64(-1))
	if !ok || crv != 1 {
		return errors.New("unsupported passkey elliptic curve")
	}

	xBytes, ok := passkeyMapBytes(m, int64(-2))
	if !ok || len(xBytes) != 32 {
		return errors.New("invalid passkey public key x")
	}
	yBytes, ok := passkeyMapBytes(m, int64(-3))
	if !ok || len(yBytes) != 32 {
		return errors.New("invalid passkey public key y")
	}

	var parsed passkeyECDSASignature
	if _, err := asn1.Unmarshal(signature, &parsed); err != nil {
		return err
	}
	if parsed.R == nil || parsed.S == nil {
		return errors.New("invalid passkey ecdsa signature")
	}

	hash := sha256.Sum256(signatureBase)
	key := ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}
	if !key.Curve.IsOnCurve(key.X, key.Y) {
		return errors.New("invalid passkey public key curve point")
	}
	if !ecdsa.Verify(&key, hash[:], parsed.R, parsed.S) {
		return errors.New("invalid passkey signature")
	}

	return nil
}

func passkeyVerifyRSA(m map[interface{}]interface{}, signatureBase []byte, signature []byte) error {
	nBytes, ok := passkeyMapBytes(m, int64(-1))
	if !ok || len(nBytes) == 0 {
		return errors.New("invalid passkey rsa modulus")
	}
	eBytes, ok := passkeyMapBytes(m, int64(-2))
	if !ok || len(eBytes) == 0 {
		return errors.New("invalid passkey rsa exponent")
	}

	exponent := new(big.Int).SetBytes(eBytes).Int64()
	if exponent <= 1 {
		return errors.New("invalid passkey rsa exponent")
	}

	hash := sha256.Sum256(signatureBase)
	key := rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: int(exponent),
	}
	return rsa.VerifyPKCS1v15(&key, crypto.SHA256, hash[:], signature)
}
