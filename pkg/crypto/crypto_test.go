package crypto

import "testing"

func TestEncryptDecryptRoundTrip(t *testing.T) {
	m, err := NewCryptoManager()
	if err != nil {
		t.Fatalf("NewCryptoManager: %v", err)
	}

	secrets := []string{"password123", "", "long secret with spaces and symbols !@#$%^&*()", "üñíçødé"}
	for _, secret := range secrets {
		encrypted, err := m.Encrypt(secret)
		if err != nil {
			t.Fatalf("Encrypt(%q): %v", secret, err)
		}
		if encrypted == secret && secret != "" {
			t.Errorf("Encrypt(%q) returned plaintext", secret)
		}

		decrypted, err := m.Decrypt(encrypted)
		if err != nil {
			t.Fatalf("Decrypt: %v", err)
		}
		if decrypted != secret {
			t.Errorf("round trip mismatch: got %q, want %q", decrypted, secret)
		}
	}
}

func TestDecryptRejectsTamperedData(t *testing.T) {
	m, err := NewCryptoManager()
	if err != nil {
		t.Fatalf("NewCryptoManager: %v", err)
	}

	encrypted, err := m.Encrypt("secret")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	if _, err := m.Decrypt("not-valid-base64!!!"); err == nil {
		t.Error("expected error for invalid base64")
	}
	if _, err := m.Decrypt(encrypted[:len(encrypted)-4] + "AAAA"); err == nil {
		t.Error("expected error for tampered ciphertext")
	}
}
