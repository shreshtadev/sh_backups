package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
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

	// Step 3: Get company by API key using API client
	company, err := apiClient.FindCompanyByAPIKey(cfg.APIKey)
	if err != nil {
		logger.Error("Failed to fetch company", err)
		os.Exit(1)
	}

	if company.TotalUsageQuota == company.UsedQuota {
		logger.Error("Company has reached its usage quota", nil)
		os.Exit(1)
	}

	currentTime := func() string {
		return time.Now().Format(time.RFC3339)
	}

	// Check for --register flag
	for _, arg := range os.Args {
		if arg == "--upload" || arg == "-U" || arg == "" {
			logger.Info(fmt.Sprintf("Uploading Operation Started at %s...", currentTime()))
			handleFileUpload(apiClient, company, cfg.LocalFolderPath)
			logger.Info(fmt.Sprintf("Uploading Operation Completed at %s...", currentTime()))
			os.Exit(0)
		}

		if arg == "--delete" || arg == "-D" {
			logger.Info(fmt.Sprintf("Deletion Operation Started %s...", currentTime()))
			handleFileDelete(apiClient, company, true)
			logger.Info(fmt.Sprintf("Deletion Operation Completed at %s...", currentTime()))
			os.Exit(0)
		}

		if arg == "--force-delete" || arg == "-FD" {
			logger.Info(fmt.Sprintf("Force Deletion Operation Started at %s...", currentTime()))
			handleFileDelete(apiClient, company, false)
			logger.Info(fmt.Sprintf("Force Deletion Operation Completed at %s...", currentTime()))
			os.Exit(0)
		}
	}
}

func handleFileUpload(apiClient *api.APIClient, company *models.Company, localFolder string) error {
	// Assume pattern is "Tally" and extension is ".zip"

	localZipPath, fileSize, err := utils.FindZipFileWithPatternAndLatestDate(localFolder)
	if err != nil || fileSize == 0 {
		logger.Error("Failed to find latest Tally file or filesize is 0", err)
		return nil
	}

	uploadKey := filepath.Base(localZipPath)
	// Step 5: Upload .zip file from local folder
	err = apiClient.UploadFile(company.CompanyApiKey, localZipPath)
	if err != nil {
		logger.Error("Failed to upload file to S3", err)
		return err
	}
	logger.Info(fmt.Sprintf("Uploaded file to S3: %s", uploadKey))

	// Get file size
	info, err := os.Stat(localZipPath)
	if err != nil {
		logger.Error("Failed to stat uploaded file", err)
		return err
	}
	size := info.Size()

	// Insert upload metadata using API client
	meta := &models.FileMetadata{
		Id:          uuid.NewString(),
		CreatedAt:   time.Now().Format(time.RFC3339),
		FileName:    filepath.Base(localZipPath),
		FileSize:    &size,
		FileKey:     uploadKey,
		CompanyId:   company.Id,
		FileTxnType: utils.PtrInt16(1), // 1 = upload
		FileTxnMeta: "Uploaded to S3",
	}
	if err := apiClient.InsertFileMetadata(meta); err != nil {
		logger.Error("Failed to insert upload metadata", err)
	}

	// Update company quota using API client
	updateQuota := &models.UpdateUsageQuota{
		UsedQuota:   size,
		FileTxnType: 1, // 1 = upload
	}
	err = apiClient.UpdateCompanyQuota(updateQuota)
	if err != nil {
		logger.Error("Failed to update company quota", err)
	}
	return nil
}

func handleFileDelete(apiClient *api.APIClient, company *models.Company, applyCondition bool) {
	companyFolder := company.CompanyName
	folderInfo, _ := apiClient.GetFolderSize(company.CompanyApiKey, locTag)
	contentSize := folderInfo.TotalSize
	var appliedCondition bool
	if applyCondition {
		appliedCondition = contentSize >= *company.TotalUsageQuota
	} else {
		appliedCondition = true
	}

	if appliedCondition {
		dErr := apiClient.DeleteFiles(company.CompanyApiKey, locTag)
		if dErr != nil {
			logger.Error("Cannot delete files", dErr)
			return
		}
		meta := &models.FileMetadata{
			Id:          uuid.NewString(),
			CreatedAt:   time.Now().Format(time.RFC3339),
			FileName:    companyFolder,
			FileSize:    &contentSize,
			FileKey:     companyFolder + "/" + locTag + "/",
			CompanyId:   company.Id,
			FileTxnType: utils.PtrInt16(2), // 2 = delete
			FileTxnMeta: "Deleted files in S3",
		}
		if err := apiClient.InsertFileMetadata(meta); err != nil {
			logger.Error("Failed to insert upload metadata", err)
		}

		updateQuota := &models.UpdateUsageQuota{
			UsedQuota:   int64(0),
			FileTxnType: 2, // 2 = delete
		}
		qErr := apiClient.UpdateCompanyQuota(updateQuota)
		if qErr != nil {
			logger.Error("Failed to update company quota", qErr)
		}
		logger.Info(fmt.Sprintf("Deleted Folder Contents with Tally backups...%dMB", (contentSize / 1024 / 1024)))
	} else {
		logger.Info(fmt.Sprintf("Under valid quota usage...%d MB", (contentSize / 1024 / 1024)))
	}
}
