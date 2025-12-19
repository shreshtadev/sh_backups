package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"shreshtasmg.in/sh_backups/api"
	"shreshtasmg.in/sh_backups/config"
	"shreshtasmg.in/sh_backups/logger"
	"shreshtasmg.in/sh_backups/models"
	"shreshtasmg.in/sh_backups/utils"
)

const (
	locTag = "TallyBackups"
)

func main() {
	// Step 1: Load config
	cfg := config.Load()

	// Step 2: Create API client
	apiBaseURL := cfg.APIBaseUrl // or from config if available
	apiClient := api.NewAPIClient(apiBaseURL, cfg.APIKey)

	currentTime := func() string {
		return time.Now().Format(time.RFC3339)
	}

	// Check for --register flag
	for _, arg := range os.Args {
		if arg == "--upload" || arg == "-U" || arg == "" {
			logger.Info(fmt.Sprintf("Uploading Operation Started at %s...", currentTime()))
			handleFileUpload(apiClient, cfg.LocalFolderPath)
			logger.Info(fmt.Sprintf("Uploading Operation Completed at %s...", currentTime()))
			os.Exit(0)
		}
	}
}

func handleFileUpload(apiClient *api.APIClient, localFolder string) error {
	// Assume pattern is "Tally" and extension is ".zip"

	localZipPath, fileSize, err := utils.FindZipFileWithPatternAndLatestDate(localFolder)
	if err != nil || fileSize == 0 {
		logger.Error("Failed to find latest Tally file or filesize is 0", err)
		return nil
	}

	uploadKey := filepath.Base(localZipPath)

	// Get file size
	info, err := os.Stat(localZipPath)
	if err != nil {
		logger.Error("Failed to stat uploaded file", err)
		return err
	}
	size := info.Size()

	// Step 5: Generate Presigned URL
	presignReq := &models.PresignUploadRequest{
		FileName:    uploadKey,
		FileSize:    size,
		LocTag:      locTag,
		FileTxnType: 1, // 1 = upload
		FileTxnMeta: utils.PtrString("Uploaded to S3"),
	}
	logger.Info(fmt.Sprintf("File to upload %s", uploadKey))
	presignedResp, err := apiClient.GeneratePresignedUploadURL(presignReq)
	if err != nil {
		logger.Error("Failed to generate presigned upload URL", err)
		return err
	}

	// Step 6: Upload .zip file using presigned URL
	err = apiClient.UploadFileWithPresignedURL(presignedResp, localZipPath)
	if err != nil {
		logger.Error("Failed to upload file to S3 using presigned URL", err)
		return err
	}
	logger.Info(fmt.Sprintf("Uploaded file to S3: %s", uploadKey))
	return nil
}
