package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

type TokenStore struct {
	path  string
	mu    sync.RWMutex
	token string
}

func NewTokenStore(path string) (*TokenStore, bool, error) {
	tok, created, err := loadOrCreateTokenFile(path)
	if err != nil {
		return nil, false, err
	}
	return &TokenStore{path: path, token: tok}, created, nil
}

func (s *TokenStore) Get() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.token
}

// Rotate generates a new token, persists it, and swaps it in-memory.
func (s *TokenStore) Rotate() (string, error) {
	newTok, err := generateToken()
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := writeTokenFileAtomic(s.path, newTok); err != nil {
		return "", err
	}
	s.token = newTok
	return newTok, nil
}

func loadOrCreateTokenFile(path string) (token string, created bool, err error) {
	b, err := os.ReadFile(path)
	if err == nil {
		t := string(bytesTrimSpace(b))
		if t == "" {
			return "", false, errors.New("token file exists but is empty")
		}
		return t, false, nil
	}
	if !os.IsNotExist(err) {
		return "", false, err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", false, err
	}

	tok, err := generateToken()
	if err != nil {
		return "", false, err
	}
	if err := writeTokenFileAtomic(path, tok); err != nil {
		return "", false, err
	}
	return tok, true, nil
}

func writeTokenFileAtomic(path, tok string) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(tok+"\n"), 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func generateToken() (string, error) {
	buf := make([]byte, 32) // 256-bit
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func bytesTrimSpace(b []byte) []byte {
	i := 0
	j := len(b)
	for i < j && (b[i] == ' ' || b[i] == '\n' || b[i] == '\r' || b[i] == '\t') {
		i++
	}
	for j > i && (b[j-1] == ' ' || b[j-1] == '\n' || b[j-1] == '\r' || b[j-1] == '\t') {
		j--
	}
	return b[i:j]
}
