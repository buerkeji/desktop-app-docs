//go:build !windows

package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
)

const desktopSecretKeyFile = "desktop.key"

func (s *DesktopStore) protectSecret(value []byte) ([]byte, error) {
	if len(value) == 0 {
		return []byte{}, nil
	}

	key, err := s.desktopSecretKey()
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("read nonce: %w", err)
	}

	return gcm.Seal(nonce, nonce, value, nil), nil
}

func (s *DesktopStore) unprotectSecret(value []byte) ([]byte, error) {
	if len(value) == 0 {
		return []byte{}, nil
	}

	key, err := s.desktopSecretKey()
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	if len(value) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce := value[:gcm.NonceSize()]
	ciphertext := value[gcm.NonceSize():]

	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt secret: %w", err)
	}

	return plain, nil
}

func (s *DesktopStore) desktopSecretKey() ([]byte, error) {
	keyPath := filepath.Join(s.dataDir, desktopSecretKeyFile)
	if existing, err := os.ReadFile(keyPath); err == nil && len(existing) == 32 {
		return existing, nil
	}

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate desktop key: %w", err)
	}

	if err := os.WriteFile(keyPath, key, 0o600); err != nil {
		return nil, fmt.Errorf("write desktop key: %w", err)
	}

	return key, nil
}
