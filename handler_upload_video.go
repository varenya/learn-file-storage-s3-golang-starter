package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	videoString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid video Id", err)
		return
	}
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid JWT", err)
		return
	}
	fmt.Printf("uploading video %s for user %s", videoID.String(), userID)
	const maxMemory = 1 << 30 // 1GB
	r.Body = http.MaxBytesReader(w, r.Body, maxMemory)
	incomingFile, headers, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Check the uploaded file", err)
		return
	}
	defer incomingFile.Close()
	contentType := headers.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Invalid format", nil)
		return
	}
	f, err := os.CreateTemp("", "tubely-video")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error trying to create a file", err)
		return
	}
	defer os.Remove(f.Name())
	defer f.Close()
	_, err = io.Copy(f, incomingFile)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to copy over the file", err)
		return
	}
	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error reseting the file seek", err)
		return
	}
	mediaTypes := strings.Split(mediaType, "/")
	fileName := fmt.Sprintf("%s.%s", randomFileName(), mediaTypes[1])
	_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Key:         &fileName,
		Bucket:      &cfg.s3Bucket,
		ContentType: &contentType,
		Body:        f,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error uploading the file in S3", err)
		return
	}
	videoInfo, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Video info not available", err)
		return
	}
	if videoInfo.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User not authorized", err)
		return
	}
	videoUrl := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, fileName)
	videoInfo.VideoURL = &videoUrl
	err = cfg.db.UpdateVideo(videoInfo)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating video entry in db", err)
		return
	}
	respondWithJSON(w, http.StatusOK, videoInfo)
}
