package utils

import "testing"

func TestPasswordHashUsesBcryptAndVerifiesLegacySha256(t *testing.T) {
	const password = "correct horse battery staple"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if hash == Sha2Encrypt(password) {
		t.Fatalf("expected password hash to avoid raw sha256")
	}

	ok, upgrade := VerifyPassword(password, hash)
	if !ok || upgrade {
		t.Fatalf("expected bcrypt password to verify without upgrade, ok=%v upgrade=%v", ok, upgrade)
	}

	ok, upgrade = VerifyPassword(password, Sha2Encrypt(password))
	if !ok || !upgrade {
		t.Fatalf("expected legacy sha256 password to verify with upgrade, ok=%v upgrade=%v", ok, upgrade)
	}
}

func TestGenerateSecretsHaveExpectedShape(t *testing.T) {
	code := GenerateCode(6)
	if len(code) != 6 {
		t.Fatalf("expected 6 digit code, got %q", code)
	}
	for _, char := range code {
		if char < '0' || char > '9' {
			t.Fatalf("expected numeric verification code, got %q", code)
		}
	}

	secret := GenerateChar(64)
	if len(secret) != 64 {
		t.Fatalf("expected 64 byte secret, got %q", secret)
	}
}
