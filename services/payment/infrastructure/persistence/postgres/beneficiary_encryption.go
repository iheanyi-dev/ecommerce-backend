package postgres

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/shared/config"
)

const beneficiarySnapshotVersion byte = 1

// BeneficiaryEncryptor encrypts payout beneficiary snapshots before persistence.
type BeneficiaryEncryptor struct {
	aead cipher.AEAD
}

func NewBeneficiaryEncryptor(key []byte) (*BeneficiaryEncryptor, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("payout encryption key must be exactly 32 bytes")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create payout encryption cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create payout encryption mode: %w", err)
	}

	return &BeneficiaryEncryptor{aead: aead}, nil
}

func (e *BeneficiaryEncryptor) Encrypt(
	beneficiary entities.PayoutBeneficiary,
) ([]byte, error) {
	if e == nil || e.aead == nil {
		return nil, fmt.Errorf("payout beneficiary encryptor is not initialized")
	}

	plaintext, err := json.Marshal(beneficiary)
	if err != nil {
		return nil, fmt.Errorf("marshal payout beneficiary: %w", err)
	}

	nonce := make([]byte, e.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate payout encryption nonce: %w", err)
	}

	// The version byte is authenticated as additional data.
	ciphertext := e.aead.Seal(nil, nonce, plaintext, []byte{beneficiarySnapshotVersion})

	result := make([]byte, 1, 1+len(nonce)+len(ciphertext))
	result[0] = beneficiarySnapshotVersion
	result = append(result, nonce...)
	result = append(result, ciphertext...)

	return result, nil
}

func (e *BeneficiaryEncryptor) Decrypt(
	snapshot []byte,
) (entities.PayoutBeneficiary, error) {
	var beneficiary entities.PayoutBeneficiary

	if e == nil || e.aead == nil {
		return beneficiary, fmt.Errorf("payout beneficiary encryptor is not initialized")
	}
	if len(snapshot) == 0 {
		return beneficiary, fmt.Errorf("payout beneficiary snapshot is empty")
	}
	if snapshot[0] != beneficiarySnapshotVersion {
		return beneficiary, fmt.Errorf("unsupported payout beneficiary snapshot version: %d", snapshot[0])
	}

	minimumSize := 1 + e.aead.NonceSize() + e.aead.Overhead()
	if len(snapshot) < minimumSize {
		return beneficiary, fmt.Errorf("payout beneficiary snapshot is truncated")
	}

	nonceStart := 1
	nonceEnd := nonceStart + e.aead.NonceSize()

	plaintext, err := e.aead.Open(
		nil,
		snapshot[nonceStart:nonceEnd],
		snapshot[nonceEnd:],
		[]byte{beneficiarySnapshotVersion},
	)
	if err != nil {
		return beneficiary, fmt.Errorf("decrypt payout beneficiary snapshot: %w", err)
	}

	if err := json.Unmarshal(plaintext, &beneficiary); err != nil {
		return beneficiary, fmt.Errorf("decode payout beneficiary snapshot: %w", err)
	}

	return beneficiary, nil
}

// NewConfiguredBeneficiaryEncryptor constructs the encryptor using the
// AES-256 key loaded from Payment service configuration.
func NewConfiguredBeneficiaryEncryptor(
	cfg *config.Config,
) (*BeneficiaryEncryptor, error) {
	return NewBeneficiaryEncryptor(cfg.BeneficiaryEncryptionKey)
}
