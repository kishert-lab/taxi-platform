package handler

import (
	"io"

	"github.com/gin-gonic/gin"

	"github.com/kishert-lab/taxi-platform/internal/dto"
)

const maxProfilePhotoBytes int64 = 5 * 1024 * 1024

var allowedProfilePhotoTypes = map[string]struct{}{
	"image/jpeg": {},
	"image/png":  {},
	"image/webp": {},
}

func profilePhotoUploadRequest(context *gin.Context) (dto.ProfilePhotoUploadRequest, func(), bool) {
	fileHeader, err := context.FormFile("photo")
	if err != nil {
		failValidation(context, "Profile photo is required")
		return dto.ProfilePhotoUploadRequest{}, func() {}, false
	}
	if fileHeader.Size <= 0 || fileHeader.Size > maxProfilePhotoBytes {
		failValidation(context, "Profile photo size must be between 1 byte and 5 MB")
		return dto.ProfilePhotoUploadRequest{}, func() {}, false
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if _, allowed := allowedProfilePhotoTypes[contentType]; !allowed {
		failValidation(context, "Profile photo content type must be image/jpeg, image/png, or image/webp")
		return dto.ProfilePhotoUploadRequest{}, func() {}, false
	}

	file, err := fileHeader.Open()
	if err != nil {
		failValidation(context, "Profile photo cannot be opened")
		return dto.ProfilePhotoUploadRequest{}, func() {}, false
	}

	closeFile := func() {
		_ = file.Close()
	}
	request := dto.ProfilePhotoUploadRequest{
		FileName:    fileHeader.Filename,
		ContentType: contentType,
		Size:        fileHeader.Size,
		Body:        io.Reader(file),
	}

	return request, closeFile, true
}
