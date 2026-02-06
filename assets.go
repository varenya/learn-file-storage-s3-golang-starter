package main

import (
	"crypto/rand"
	"encoding/base64"
	"os"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0o755)
	}
	return nil
}

func randomFileName() string {
	randomBytes := make([]byte, 32)
	rand.Read(randomBytes)
	randomFileName := base64.RawURLEncoding.EncodeToString(randomBytes)
	return randomFileName
}
