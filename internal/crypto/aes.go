package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
)

// DecryptAuthFile reads and decrypts a Claude Code auth.v2 file.
// Format: base64(iv):base64(authTag):base64(ciphertext)
func DecryptAuthFile(keyPath, filePath string) ([]byte, error) {
	key, err := readKey(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read key: %w", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read auth file: %w", err)
	}

	parts := strings.Split(strings.TrimSpace(string(data)), ":")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid auth file format: expected 3 parts, got %d", len(parts))
	}

	iv, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("decode iv: %w", err)
	}

	authTag, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode auth tag: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCMWithNonceSize(block, len(iv))
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	// GCM expects ciphertext + tag appended
	sealed := append(ciphertext, authTag...)
	plaintext, err := gcm.Open(nil, iv, sealed, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	return plaintext, nil
}

// EncryptAuthFile encrypts data and writes it in Claude Code auth.v2 format.
func EncryptAuthFile(keyPath, filePath string, plaintext []byte) error {
	key, err := readKey(keyPath)
	if err != nil {
		return fmt.Errorf("read key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("create cipher: %w", err)
	}

	// Claude Code uses 16-byte nonce
	const nonceSize = 16
	gcm, err := cipher.NewGCMWithNonceSize(block, nonceSize)
	if err != nil {
		return fmt.Errorf("create gcm: %w", err)
	}

	iv := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return fmt.Errorf("generate iv: %w", err)
	}

	sealed := gcm.Seal(nil, iv, plaintext, nil)
	// sealed = ciphertext + authTag (last 16 bytes)
	tagSize := gcm.Overhead()
	ciphertext := sealed[:len(sealed)-tagSize]
	authTag := sealed[len(sealed)-tagSize:]

	encoded := base64.StdEncoding.EncodeToString(iv) + ":" +
		base64.StdEncoding.EncodeToString(authTag) + ":" +
		base64.StdEncoding.EncodeToString(ciphertext)

	return os.WriteFile(filePath, []byte(encoded), 0600)
}

// Decrypt decrypts raw auth data (already read from file) using the given key bytes.
func Decrypt(key []byte, data string) ([]byte, error) {
	parts := strings.Split(strings.TrimSpace(data), ":")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid format: expected 3 parts, got %d", len(parts))
	}

	iv, _ := base64.StdEncoding.DecodeString(parts[0])
	authTag, _ := base64.StdEncoding.DecodeString(parts[1])
	ciphertext, _ := base64.StdEncoding.DecodeString(parts[2])

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, len(iv))
	if err != nil {
		return nil, err
	}

	sealed := append(ciphertext, authTag...)
	return gcm.Open(nil, iv, sealed, nil)
}

func readKey(keyPath string) ([]byte, error) {
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
}
