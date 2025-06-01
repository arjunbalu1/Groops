package services

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

type ImageService struct {
	cld *cloudinary.Cloudinary
}

func NewImageService() (*ImageService, error) {
	// Get Cloudinary configuration from environment
	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")

	if cloudName == "" || apiKey == "" || apiSecret == "" {
		return nil, fmt.Errorf("missing Cloudinary configuration")
	}

	cld, err := cloudinary.NewFromParams(cloudName, apiKey, apiSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Cloudinary: %w", err)
	}

	return &ImageService{cld: cld}, nil
}

// UploadAvatar uploads an avatar image to Cloudinary
func (s *ImageService) UploadAvatar(file multipart.File, filename string, userID string) (string, error) {
	// Validate file type
	allowedTypes := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if !allowedTypes[ext] {
		return "", fmt.Errorf("invalid file type: %s. Allowed types: jpg, jpeg, png, gif, webp", ext)
	}

	// Create a unique public ID for the avatar
	publicID := fmt.Sprintf("avatars/user_%s", userID)

	// Upload parameters
	uploadParams := uploader.UploadParams{
		PublicID:       publicID,
		Folder:         "groops/avatars",
		Overwrite:      &[]bool{true}[0],
		ResourceType:   "image",
		Transformation: "c_fill,g_face,h_300,w_300/q_auto,f_auto", // Auto-crop to face, 300x300, optimize quality and format
	}

	// Upload to Cloudinary
	result, err := s.cld.Upload.Upload(context.Background(), file, uploadParams)
	if err != nil {
		return "", fmt.Errorf("failed to upload image: %w", err)
	}

	return result.SecureURL, nil
}

// DeleteAvatar deletes an avatar from Cloudinary
func (s *ImageService) DeleteAvatar(publicID string) error {
	_, err := s.cld.Upload.Destroy(context.Background(), uploader.DestroyParams{
		PublicID: publicID,
	})
	return err
}

// ValidateImageFile validates if the uploaded file is a valid image
func (s *ImageService) ValidateImageFile(file multipart.File, maxSize int64) error {
	// Reset file pointer
	file.Seek(0, 0)

	// Check file size
	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	if int64(len(data)) > maxSize {
		return fmt.Errorf("file too large: %d bytes (max %d bytes)", len(data), maxSize)
	}

	// Reset file pointer for later use
	file.Seek(0, 0)

	return nil
}
