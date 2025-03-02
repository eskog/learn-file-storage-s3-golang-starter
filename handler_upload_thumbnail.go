package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

const maxMemory int64 = 10 << 20 // 10 leftshift 20 times = 10 * 1024 * 1024 = 10 MB

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

	r.ParseMultipartForm(maxMemory)
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "could not process request", err)
		return
	}

	mediatype := header.Header.Get("Content-Type")
	log.Print(mediatype)
	if mediatype != "image/png" && mediatype != "image/jpeg" {
		respondWithError(w, http.StatusBadRequest, "Only images allowed", errors.New("only images allowed"))
		return
	}
	extensions := strings.SplitAfter(mediatype, "/")
	extension := extensions[1]

	var fileName [32]byte
	rand.Read(fileName[:])

	assetPath := filepath.Join(cfg.assetsRoot, base64.RawURLEncoding.EncodeToString(fileName[:]))
	assetPath = assetPath + "." + extension
	rawFile, err := os.Create(assetPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to store local file", err)
		return
	}

	//old below to clean
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Could not find video", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	io.Copy(rawFile, file)
	assetPath = fmt.Sprintf("http://localhost:%s/%s", cfg.port, assetPath)
	video.ThumbnailURL = &assetPath
	log.Printf("Assetpath is %s", assetPath)
	if err = cfg.db.UpdateVideo(video); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating database", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
