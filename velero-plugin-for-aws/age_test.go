package main

import (
	"bytes"
	"io"
	"testing"

	"filippo.io/age"
)

func TestAgeEncryptDecrypt_Stream(t *testing.T) {
	id, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("generate identity: %v", err)
	}
	pub := id.Recipient().String()
	priv := id.String()

	ager := NewAge(pub, priv)

	original := []byte("Test msg")

	cipherRc, err := ager.Encrypt(bytes.NewReader(original))
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	cipherData, err := io.ReadAll(cipherRc)
	if err != nil {
		t.Fatalf("reading ciphertext: %v", err)
	}
	cipherRc.Close()

	if len(cipherData) == 0 {
		t.Fatalf("cipherData empty")
	}

	plainRc, err := ager.Decrypt(bytes.NewReader(cipherData))
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	recovered, err := io.ReadAll(plainRc)
	if err != nil {
		t.Fatalf("reading plaintext: %v", err)
	}
	plainRc.Close()

	if !bytes.Equal(recovered, original) {
		t.Errorf("Round-trip mismatch:\n got: %q\nwant: %q", recovered, original)
	}
}
