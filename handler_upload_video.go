package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	http.MaxBytesReader(w, r.Body, 10<<30)
	videoIDString := r.PathValue("videoID")

	videoUUID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "can not parse to uuid", err)
		return
	}
	//Auth the user

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

	video, err := cfg.db.GetVideo(videoUUID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to find video", err)
		return
	}
	if userID != video.UserID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}
	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to get video", err)
		return
	}
	defer file.Close()
	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil || mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Wrong media file", err)
		return
	}
	tempFile, err := os.CreateTemp("", "tempUploadTubely.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to create temp file", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()
	io.Copy(tempFile, file)
	tempFile.Seek(0, io.SeekStart)

	var fileKey [32]byte
	rand.Read(fileKey[:])
	fileKeystring := base64.RawURLEncoding.EncodeToString(fileKey[:])

	//HERE add aspect
	var prefix string
	aspect, _ := getVideoAspectRatio(tempFile.Name())
	if aspect == "16:9" {
		prefix = "landscape/"
	} else if aspect == "9:16" {
		prefix = "portrait/"
	} else {
		prefix = "other/"
	}
	fileKeystring = prefix + fileKeystring
	processedFilepath, err := processVideoForFastStart(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error processing file", err)
		log.Printf("Error processing file: %s", err)
	}
	processedFile, _ := os.Open(processedFilepath)
	defer processedFile.Close()

	myParams := s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &fileKeystring,
		Body:        processedFile,
		ContentType: &mediaType,
	}
	cfg.s3Client.PutObject(context.Background(), &myParams)

	fileURL := fmt.Sprintf("%s%s", cfg.s3CfDistribution, fileKeystring)

	bucketstring := cfg.s3Bucket + "," + fileKeystring
	video.VideoURL = &bucketstring
	video.VideoURL = &fileURL
	cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error generating signedURL", err)
	}
	respondWithJSON(w, http.StatusOK, video)

}
