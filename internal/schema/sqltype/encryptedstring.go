package sqltype

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"fmt"
	"io"
)

var encryptionKey string

// SetEncryptionKey sets the encryption key.
// This should be called only one time, as soon as the encryption key is known.
//
// This will panic if the key is invalid.
func SetEncryptionKey(key string) {
	if len(key) == 0 {
		encryptionKey = ""
		return
	}
	if len(key) == 32 {
		encryptionKey = key
		return
	}
	if len(key) == 64 {
		decodedKey, err := hex.DecodeString(key)
		if err != nil {
			panic(fmt.Errorf("could not decode hexadecimal-encoded encryption key: %w", err))
		}
		encryptionKey = string(decodedKey)
		return
	}
	panic("encryption key must be 32 (raw) or 64 (hexadecimal-encoded) characters long")
}

// EncryptedString is a custom type for an encrypted string.
type EncryptedString string

var _ driver.Valuer = (*EncryptedString)(nil)
var _ sql.Scanner = (*EncryptedString)(nil)

// Value implements driver.Valuer: converts Go slice to JSON for the DB
func (a EncryptedString) Value() (driver.Value, error) {
	if encryptionKey == "" {
		return nil, fmt.Errorf("encryption key not set; please call SetEncryptionKey() before using this type")
	}

	block, err := aes.NewCipher([]byte(encryptionKey))
	if err != nil {
		return nil, fmt.Errorf("could not create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("could not create GCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("could not create nonce: %w", err)
	}
	encrypted := gcm.Seal(nonce, nonce, []byte(a), nil)
	return encrypted, nil
}

// Scan implements sql.Scanner: converts JSON from the DB to Go slice
func (a *EncryptedString) Scan(src any) error {
	if src == nil {
		return nil
	}
	var content []byte
	switch v := src.(type) {
	case []byte:
		content = v
	case string:
		content = []byte(v)
	default:
		return fmt.Errorf("invalid underlying type: %T", src)
	}

	if encryptionKey == "" {
		return fmt.Errorf("encryption key not set; please call SetEncryptionKey() before using this type")
	}

	block, err := aes.NewCipher([]byte(encryptionKey))
	if err != nil {
		return fmt.Errorf("could not create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("could not create GCM: %w", err)
	}
	nonce := content[:gcm.NonceSize()]
	decrypted, err := gcm.Open(nil, nonce, content[gcm.NonceSize():], nil)
	if err != nil {
		return fmt.Errorf("could not decrypt content: %w", err)
	}
	*a = EncryptedString(decrypted)
	return nil
}
