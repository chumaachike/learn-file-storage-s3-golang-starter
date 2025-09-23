package main

import (
	"errors"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
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

// func (cfg apiConfig) getVideoURL(assetPath, prefix string) string {
// 	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s/%s", cfg.s3Bucket, cfg.s3Region, prefix, assetPath)
// }

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if video.VideoURL == nil {
		return video, nil // no video, nothing to sign
	}

	bucketKey := strings.Split(*video.VideoURL, ",")
	if len(bucketKey) != 2 {
		return video, fmt.Errorf("invalid VideoURL format: %q", *video.VideoURL)
	}

	presignedURL, err := generatePresignedURL(cfg.s3Client, bucketKey[0], bucketKey[1], 60*time.Minute)
	if err != nil {
		return database.Video{}, err
	}

	video.VideoURL = &presignedURL
	return video, nil
}
