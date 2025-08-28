package main

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
        video.ThumbnailURL, err = cfg.uploadImageToAssets(videoIDString, cfg.assetsRoot, contentType, data)
        err = cfg.db.UpdateVideo(video)
        if err != nil {
                respondWithError(w, http.StatusInternalServerError, " Failed to update video data.", err)
                return
        }
        fmt.Println("New thumbnail URL:", *video.ThumbnailURL)
	respondWithJSON(w, http.StatusOK, video)
}

func (cfg *apiConfig) uploadImageToAssets (videoId string, assetsRoot string, mediaType string, thumbnail multipart.File) (*string, error) {
        /*
        Saves image in assets with specific <videoId>.<file_extension> format 

        input:
                videoId         string          stringified uuid 
                mediaType       string          contains information on the type of picture
                imageByte       []byte          byte map of the image data 

        output:
                *string         pointer to the created data url 
                error           error object containing information on what failed during the function call
        */

        extension := getFileExtention(mediaType)
        thumbnailFile := strings.Join([]string{videoId, extension}, ".")
        thumbnailPath := filepath.Join(assetsRoot, thumbnailFile)
        file, err := os.Create(thumbnailPath)
        if err != nil {
                return nil, err
        }
        defer file.Close()
        io.Copy(file, thumbnail)
        thumbnailUrl := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, thumbnailFile)
        return &thumbnailUrl, nil
}

func getFileExtention(mediaType string) string {
        parts := strings.Split(mediaType, "/")
        return parts[len(parts)-1]
}
