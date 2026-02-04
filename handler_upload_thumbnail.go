package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

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

	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to parse form file", err)
		return
	}
	defer file.Close()
	mediaType := header.Header.Get("Content-Type")
	imageData, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid image data", err)
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
	videoThumbnails[videoID] = thumbnail{
		mediaType: mediaType,
		data:      imageData,
	}
	thumbUrl := fmt.Sprintf("http://localhost:%s/api/thumbnails/%s", cfg.port, videoID)
	videoInfo.ThumbnailURL = &thumbUrl
	err = cfg.db.UpdateVideo(videoInfo)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "User not authorized", err)
		return
	}
	respondWithJSON(w, http.StatusOK, videoInfo)
}
