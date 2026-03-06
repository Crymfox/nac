package crypto

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	testCases := []struct {
		name       string
		plaintext  string
		passphrase string
	}{
		{"empty", "", "secret"},
		{"short", "hello", "secret"},
		{"exact_block", "1234567890123456", "secret"},
		{"long", "this is a much longer string that will span multiple AES blocks easily", "super_secure_key_123!!"},
		{"json", `{"apiKey": "sk-1234567890abcdef", "organizationId": "org-xyz"}`, "n8n_encryption_key_test"},
		{"utf8", "こんにちは世界 🌍", "パスワード"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.plaintext == "" {
				// Empty string special case
				enc, err := Encrypt(tc.plaintext, tc.passphrase)
				if err != nil {
					t.Fatalf("Encrypt failed: %v", err)
				}
				if enc != "" {
					t.Errorf("expected empty string, got %q", enc)
				}
				dec, err := Decrypt(enc, tc.passphrase)
				if err != nil {
					t.Fatalf("Decrypt failed: %v", err)
				}
				if dec != "" {
					t.Errorf("expected empty string, got %q", dec)
				}
				return
			}

			// Encrypt
			enc, err := Encrypt(tc.plaintext, tc.passphrase)
			if err != nil {
				t.Fatalf("Encrypt failed: %v", err)
			}

			// Verify format
			raw, err := base64.StdEncoding.DecodeString(enc)
			if err != nil {
				t.Fatalf("Invalid base64: %v", err)
			}
			if !strings.HasPrefix(string(raw), "Salted__") {
				t.Errorf("Missing Salted__ prefix")
			}

			// Decrypt
			dec, err := Decrypt(enc, tc.passphrase)
			if err != nil {
				t.Fatalf("Decrypt failed: %v", err)
			}

			// Compare
			if dec != tc.plaintext {
				t.Errorf("Round trip failed. Expected %q, got %q", tc.plaintext, dec)
			}
		})
	}
}

func TestDecrypt_KnownOpenSSL(t *testing.T) {
	// These are actual values encrypted via:
	// echo -n '{"apiKey":"sk-12345"}' | openssl enc -e -aes-256-cbc -pass pass:mykey -md md5 | base64 -w0
	// This perfectly matches what n8n outputs.

	encrypted := "U2FsdGVkX19xLH5zjgQ+E2BaIfBAHIj8fSf3XPKkuPIeh7iOxnP33Poaaj0CN1gG"
	passphrase := "mykey"
	expected := `{"apiKey":"sk-12345"}`

	dec, err := Decrypt(encrypted, passphrase)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if dec != expected {
		t.Errorf("Expected %q, got %q", expected, dec)
	}
}

func TestDecrypt_WrongPassword(t *testing.T) {
	plaintext := "secret message"
	passphrase := "correct_horse_battery_staple"

	enc, err := Encrypt(plaintext, passphrase)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Decrypt(enc, "wrong_password")
	if err == nil {
		t.Fatal("Decrypt should fail with wrong password (padding error)")
	}
}

func TestDecrypt_InvalidData(t *testing.T) {
	passphrase := "secret"

	_, err := Decrypt("not-base64-!@#$", passphrase)
	if err == nil {
		t.Error("should fail on invalid base64")
	}

	_, err = Decrypt("YmFzZTY0YnV0bm90c2FsdGVk", passphrase) // "base64butnotsalted"
	if err == nil {
		t.Error("should fail on missing Salted__ prefix")
	}

	_, err = Decrypt("U2FsdGVkX18=", passphrase) // Just Salted__ prefix
	if err == nil {
		t.Error("should fail on too short payload")
	}
}
