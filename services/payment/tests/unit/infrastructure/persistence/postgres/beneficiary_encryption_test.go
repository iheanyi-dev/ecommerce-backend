package postgres_test

import (
	"bytes"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/entities"
	postgresinfra "github.com/iheanyi-dev/ecommerce-backend/services/payment/infrastructure/persistence/postgres"
)

func TestBeneficiaryEncryptorRoundTrip(t *testing.T) {
	encryptor, err := postgresinfra.NewBeneficiaryEncryptor(bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatalf("create encryptor: %v", err)
	}

	original := entities.PayoutBeneficiary{
		AccountName:   "Test Store",
		AccountNumber: "0123456789",
		BankCode:      "058",
		BankName:      "Example Bank",
		CountryCode:   "NG",
	}

	encrypted, err := encryptor.Encrypt(original)
	if err != nil {
		t.Fatalf("encrypt beneficiary: %v", err)
	}

	if bytes.Contains(encrypted, []byte(original.AccountNumber)) {
		t.Fatal("encrypted snapshot contains plaintext account number")
	}

	decrypted, err := encryptor.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("decrypt beneficiary: %v", err)
	}

	if decrypted != original {
		t.Fatalf("decrypted beneficiary mismatch: got %+v, want %+v", decrypted, original)
	}
}

func TestBeneficiaryEncryptorUsesUniqueNonces(t *testing.T) {
	encryptor, err := postgresinfra.NewBeneficiaryEncryptor(bytes.Repeat([]byte{9}, 32))
	if err != nil {
		t.Fatalf("create encryptor: %v", err)
	}

	beneficiary := entities.PayoutBeneficiary{
		AccountName:   "Test Store",
		AccountNumber: "0123456789",
		BankCode:      "058",
		BankName:      "Example Bank",
		CountryCode:   "NG",
	}

	first, err := encryptor.Encrypt(beneficiary)
	if err != nil {
		t.Fatalf("first encryption: %v", err)
	}
	second, err := encryptor.Encrypt(beneficiary)
	if err != nil {
		t.Fatalf("second encryption: %v", err)
	}

	if bytes.Equal(first, second) {
		t.Fatal("encrypting the same beneficiary twice produced identical ciphertext")
	}
}

func TestBeneficiaryEncryptorRejectsTamperedSnapshot(t *testing.T) {
	encryptor, err := postgresinfra.NewBeneficiaryEncryptor(bytes.Repeat([]byte{3}, 32))
	if err != nil {
		t.Fatalf("create encryptor: %v", err)
	}

	encrypted, err := encryptor.Encrypt(entities.PayoutBeneficiary{
		AccountName:   "Test Store",
		AccountNumber: "0123456789",
		BankCode:      "058",
		BankName:      "Example Bank",
		CountryCode:   "NG",
	})
	if err != nil {
		t.Fatalf("encrypt beneficiary: %v", err)
	}

	encrypted[len(encrypted)-1] ^= 0xff

	if _, err := encryptor.Decrypt(encrypted); err == nil {
		t.Fatal("expected tampered snapshot to be rejected")
	}
}

func TestNewBeneficiaryEncryptorRejectsInvalidKeyLength(t *testing.T) {
	_, err := postgresinfra.NewBeneficiaryEncryptor([]byte("too-short"))
	if err == nil {
		t.Fatal("expected invalid encryption key length to be rejected")
	}
}

func TestBeneficiaryEncryptorRejectsInvalidSnapshots(t *testing.T) {
	encryptor, err := postgresinfra.NewBeneficiaryEncryptor(bytes.Repeat([]byte{4}, 32))
	if err != nil {
		t.Fatalf("create encryptor: %v", err)
	}

	tests := []struct {
		name     string
		snapshot []byte
	}{
		{name: "empty", snapshot: nil},
		{name: "unknown version", snapshot: []byte{99}},
		{name: "truncated", snapshot: []byte{1, 2, 3}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := encryptor.Decrypt(test.snapshot); err == nil {
				t.Fatal("expected invalid snapshot to be rejected")
			}
		})
	}
}
