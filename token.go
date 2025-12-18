package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
)

func LoadOrCreateToken(path string) (token string, created bool, err error) {
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

	// Create directory and token file with 0600 perms
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", false, err
	}

	tok, err := generateToken()
	if err != nil {
		return "", false, err
	}

	// Write atomically-ish
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(tok+"\n"), 0o600); err != nil {
		return "", false, err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return "", false, err
	}

	return tok, true, nil
}

func generateToken() (string, error) {
	buf := make([]byte, 32) // 256-bit
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	// URL-safe, no padding
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func bytesTrimSpace(b []byte) []byte {
	// tiny trim to avoid pulling in bytes package for one call
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
