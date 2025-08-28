package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	/*
        Function to handle thumbnail upload. Receives data via https request. Uses the video id 
        to upload a dataUrl into ThumbnailURL comlumn of the db. dataUrl contains base64 encoded
        string of the image byte map and content-type of the image (data:<media-type>;base64,<data>).
        Uses automated json responses on success or failure. See json.go file for more information.

        input:
                w               http.ResponseWriter
                r               http.Request

        output:

        */
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
        const maxMemory int64 = 10*1024*1024
        err = r.ParseMultipartForm(maxMemory)
        if err != nil {
                respondWithError(w, http.StatusInternalServerError, "Failed to parse multipart form.", err)
                return
        }
        data, header, err := r.FormFile("thumbnail")
        if err != nil {
                respondWithError(w, http.StatusInternalServerError, "Failed to retrieve file data", err)
                return
        }
        defer data.Close()
        contentType := header.Header.Get("Content-Type")
        imageByte, err := io.ReadAll(data)
        video, err := cfg.db.GetVideo(videoID)
        if err != nil {
                respondWithError(w, http.StatusInternalServerError, " Failed to get video from database.", err)
                return
        }
        if video.UserID != userID {
                respondWithError(w, http.StatusUnauthorized, "You are not the owner of the requested video.", err)
                return
        }
        // using func below write image data and media type into single dataUrl  
        video.ThumbnailURL, err = writeDataToThumbnailUrl(contentType, imageByte)
        err = cfg.db.UpdateVideo(video)
        if err != nil {
                respondWithError(w, http.StatusInternalServerError, " Failed to update video data.", err)
                return
        }
	respondWithJSON(w, http.StatusOK, video)
}

func writeDataToThumbnailUrl (mediaType string, imageByte []byte) (*string, error) {
        /*
        Function to create data url 

        input:
                mediaType       string          contains information on the type of picture
                imageByte       []byte          byte map of the image data 

        output:
                *string         pointer to the created data url 
                error           error object containing information on what failed during the function call
        */

        dataString := base64.StdEncoding.EncodeToString(imageByte)
        if dataString == "" {
                return nil, fmt.Errorf("Failed to encode image data.")
        }
        dataUrl := fmt.Sprintf("data:%s;base64,%s", mediaType, dataString)
        return &dataUrl, nil
}
