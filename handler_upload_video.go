package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	// Parse ID from request path and into UUID
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

	// Authenticate user with token
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Could not validate toke", err)
	}

	// Get video from database
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
	const maxMemory = 1 << 30
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid multipart form", err)
		return
	}

	// Extract uploaded file
	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to parse video file", err)
		return
	}

	defer file.Close()

	// Validate the uploaded file to ensure it's an MP4 video
	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Content-Type", err)
		return
	}
	newFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to create temp file", err)
	}

	defer os.Remove(newFile.Name())

	defer newFile.Close()

	// Copy the contents from the form file to temp
	if _, err := io.Copy(newFile, file); err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to copy data to file", err)
		return
	}

	//Reset file pointer after reading
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to reset file reader", err)
		return
	}

	key := make([]byte, 16) // 16 bytes → 32 hex characters
	if _, err := rand.Read(key); err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to generate random key", err)
		return
	}
	randString := hex.EncodeToString(key) // 32 chars long

	assetPath, err := getAssetPath(randString, mediaType)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to asset file type", err)
	}

	//Reset new file pointer
	if _, err := newFile.Seek(0, 0); err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to reset file pointer", err)
		return
	}
	aspRatio, err := getVideoAspectRatio(newFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to get video aspect ratio", err)
		return
	}
	processedVideo, err := processVideoForFastStart(newFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to process video", err)
		return
	}
	processedFile, err := os.Open(processedVideo)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to open processed file", err)
		return
	}
	prefix := ""
	switch aspRatio {
	case "16:9":
		prefix = "landscape"
	case "9:16":
		prefix = "portrait"
	default:
		prefix = "other"
	}
	s3Key := fmt.Sprintf("%s/%s", prefix, assetPath)
	// Put video in s3 bucket
	input := &s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &s3Key,
		Body:        processedFile, // file contents
		ContentType: &mediaType,    // e.g. "video/mp4"
	}

	_, err = cfg.s3Client.PutObject(r.Context(), input)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to upload to s3 bucket", err)
		return
	}

	//Get public IRL
	publicURL := fmt.Sprintf("%s,%s", cfg.s3Bucket, s3Key)
	video.VideoURL = &publicURL

	// 🚨 only store bucket+key in DB
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to update video", err)
		return
	}

	// ✅ now return presigned URL to client
	video, err = cfg.dbVideoToSignedVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to presign video url", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
