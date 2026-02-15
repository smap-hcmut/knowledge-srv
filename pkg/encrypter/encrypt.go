package encrypter

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

var (
	// ErrInvalidKeyLength is returned when the encryption key has an invalid length
	ErrInvalidKeyLength = errors.New("encryption key must be 16, 24, or 32 bytes long")
	// ErrCiphertextTooShort is returned when the ciphertext is too short to decrypt
	ErrCiphertextTooShort = errors.New("ciphertext is too short")
	// ErrDecryptionFailed is returned when decryption fails
	ErrDecryptionFailed = errors.New("decryption failed: invalid ciphertext or key")
)

// validateKey checks if the key has a valid length for AES (16, 24, or 32 bytes)
func validateKey(key []byte) error {
	keyLen := len(key)
	if keyLen != 16 && keyLen != 24 && keyLen != 32 {
		return fmt.Errorf("%w: got %d bytes", ErrInvalidKeyLength, keyLen)
	}
	return nil
}

func (e implEncrypter) createByteKey() ([]byte, error) {
	key := []byte(e.key)
	if err := validateKey(key); err != nil {
		return nil, err
	}
	return key, nil
}

// getGCM creates a GCM cipher from the key
func (e implEncrypter) getGCM() (cipher.AEAD, error) {
	key, err := e.createByteKey()
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return gcm, nil
}

// Encrypt encrypts a plaintext string using AES-GCM and returns a base64-encoded ciphertext.
func (e implEncrypter) Encrypt(plaintext string) (string, error) {
	return e.EncryptBytesToString([]byte(plaintext))
}

// Decrypt decrypts a base64-encoded ciphertext string and returns the plaintext.
func (e implEncrypter) Decrypt(ciphertextStr string) (string, error) {
	plaintext, err := e.DecryptStringToBytes(ciphertextStr)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// EncryptBytesToString encrypts a byte slice using AES-GCM and returns a base64-encoded ciphertext.
func (e implEncrypter) EncryptBytesToString(data []byte) (string, error) {
	gcm, err := e.getGCM()
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptStringToBytes decrypts a base64-encoded ciphertext string and returns the plaintext bytes.
func (e implEncrypter) DecryptStringToBytes(ciphertext string) ([]byte, error) {
	gcm, err := e.getGCM()
	if err != nil {
		return nil, err
	}

	ciphertextByte, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertextByte) < nonceSize {
		return nil, ErrCiphertextTooShort
	}

	nonce, ciphertextByte := ciphertextByte[:nonceSize], ciphertextByte[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextByte, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	return plaintext, nil
}
