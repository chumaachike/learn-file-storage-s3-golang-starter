package main

import (
	"errors"
	"fmt"
	"mime"
	"os"
	"path/filepath"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func getAssetPath(randomString, mediaType string) (string, error) {
	// Read file contents and detect content type
	extentions, err := mime.ExtensionsByType(mediaType)
	if err != nil || len(extentions) == 0 {
		return "", errors.New("unsuported media type")
	}
	return fmt.Sprintf("%s%s", randomString, extentions[0]), nil
}

func (cfg apiConfig) getAssetDiskPath(assetPath string) string {
	return filepath.Join(cfg.assetsRoot, assetPath)
}

func (cfg apiConfig) getAssetURL(assetPath string) string {
	return fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, assetPath)
}
