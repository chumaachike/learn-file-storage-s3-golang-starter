package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

// handlerUploadThumbnail handles uploading a thumbnail for a video.
// It validates the request, ensures the user owns the video, stores the thumbnail,
// updates the video's metadata, and responds with the updated record.
func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	// Parse video ID from request path
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	// Authenticate request with JWT from Authorization header
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// Get video form database
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to retrieve metadata from db", err)
		return
	}

	// Ensure the video belongs to the authenticated user
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "illegal access", nil)
		return
	}

	// Parse multipart form with a memory cap
	const maxMemory = 10 << 20 // 10 MB
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid multipart form", err)
		return
	}

	// Extract the uploaded file
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to parse file ", err)
		return
	}
	defer file.Close()

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Content-Type", err)
		return
	}

	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Invalid file type", nil)
		return
	}

	// Read file contents and detect content type
	key := make([]byte, 32)
	rand.Read(key)
	rand_string := base64.RawURLEncoding.EncodeToString(key)
	assetPath, err := getAssetPath(rand_string, mediaType)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unsupported file type", err)
		return
	}

	// Reset file pointer after reading
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to reset file reader", err)
		return
	}

	// Create file and copy media to file

	filePath := cfg.getAssetDiskPath(assetPath)

	newFile, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to create file", err)
		return
	}
	defer newFile.Close()
	if _, err := io.Copy(newFile, file); err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to copy data to file", err)
		return
	}

	// Use relative URL (to be served by your API)
	publicURL := cfg.getAssetURL(assetPath)

	video.ThumbnailURL = &publicURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to update video", err)
		return
	}

	// Respond with updated video metadata
	respondWithJSON(w, http.StatusOK, video)
}
