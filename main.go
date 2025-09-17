package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"shreshtasmg.in/sh_backups/api"
	"shreshtasmg.in/sh_backups/config"
	"shreshtasmg.in/sh_backups/logger"
	"shreshtasmg.in/sh_backups/models"
	"shreshtasmg.in/sh_backups/s3"
	"shreshtasmg.in/sh_backups/utils"
)

func handleRegisterCompany(apiClient *api.APIClient) {
	reader := bufio.NewReader(os.Stdin)
	var company models.RegisterCompany

	fmt.Print("Enter Company Name: ")
	company.CompanyName, _ = reader.ReadString('\n')
	company.CompanyName = strings.TrimSpace(company.CompanyName)

	fmt.Print("Enter Local Folder Path: ")
	company.LocalFolder, _ = reader.ReadString('\n')
	company.LocalFolder = strings.TrimSpace(company.LocalFolder)

	logger.Info(fmt.Sprintf("Registering company: %+v", company))

	registeredCompany, err := apiClient.RegisterCompany(company)
	if err != nil {
		logger.Error("Failed to register company", err)
		return
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Error("Cannot find home dir", err)
	}
	licPath := filepath.Join(homeDir, "apikey.lic")
	file, err := os.Create(licPath)
	if err != nil {
		logger.Error("Failed to create apikey.lic file", err)
		return
	}
	defer file.Close()
	_, err = fmt.Fprintf(file, "API_KEY=%s\nAPI_BASE_URL=%s", registeredCompany.CompanyApiKey, registeredCompany.BaseURL)
	if err != nil {
		logger.Error("Failed to write API Key to apikey.lic", err)
		return
	}
	logger.Info("Company registered successfully!")
}

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

	// Step 4: Delete .zip file from S3 matching pattern
	s3Client, err := s3.New(company.Region, company.AccessKey, company.SecretKey, company.BucketName)
	if err != nil {
		logger.Error("Failed to initialize S3 client", err)
		os.Exit(1)
	}

	currentTime := func() string {
		return time.Now().Format(time.RFC3339)
	}

	// Check for --register flag
	for _, arg := range os.Args {
		if arg == "--register" || arg == "-R" {
			logger.Info(fmt.Sprintf("Registration Operation Started at %s...", currentTime()))
			handleRegisterCompany(apiClient)
			logger.Info(fmt.Sprintf("Registration Operation Completed at %s...", currentTime()))
			os.Exit(0)
		}

		if arg == "--upload" || arg == "-U" || arg == "" {
			logger.Info(fmt.Sprintf("Uploading Operation Started at %s...", currentTime()))
			handleFileUpload(apiClient, s3Client, company)
			logger.Info(fmt.Sprintf("Uploading Operation Completed at %s...", currentTime()))
			os.Exit(0)
		}

		if arg == "--delete" || arg == "-D" {
			logger.Info(fmt.Sprintf("Deletion Operation Started %s...", currentTime()))
			handleFileDelete(apiClient, s3Client, company, true)
			logger.Info(fmt.Sprintf("Deletion Operation Completed at %s...", currentTime()))
			os.Exit(0)
		}

		if arg == "--force-delete" || arg == "-FD" {
			logger.Info(fmt.Sprintf("Force Deletion Operation Started at %s...", currentTime()))
			handleFileDelete(apiClient, s3Client, company, false)
			logger.Info(fmt.Sprintf("Force Deletion Operation Completed at %s...", currentTime()))
			os.Exit(0)
		}
	}
}

func handleFileUpload(apiClient *api.APIClient, s3Client *s3.S3Client, company *models.Company) error {
	// Assume pattern is "Tally" and extension is ".zip"

	localZipPath, fileSize, err := utils.FindZipFileWithPatternAndLatestDate(company.LocalFolder)
	if err != nil || fileSize == 0 {
		logger.Error("Failed to find latest Tally file or filesize is 0", err)
		return nil
	}

	uploadKey := filepath.Base(localZipPath)
	companyFolder := utils.Slugify(company.CompanyName)
	// Check if file with the same name exists in S3
	fileToCheckIfExists := strings.Join([]string{companyFolder, uploadKey}, "/")
	_, metaerr := s3Client.GetFileMetadata(&fileToCheckIfExists)
	if metaerr != nil {
		logger.Info(fmt.Sprintf("Unable to find file with name %s", uploadKey))
	} else {
		logger.Info(fmt.Sprintf("Found file find file with name %s", uploadKey))
		return nil
	}
	// Step 5: Upload .zip file from local folder
	err = s3Client.UploadFile(uploadKey, localZipPath, company.CompanyName)
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

func handleFileDelete(apiClient *api.APIClient, s3Client *s3.S3Client, company *models.Company, applyCondition bool) {
	companyFolder := company.CompanyName
	contentSize, sErr := s3Client.GetTotalContentLength(companyFolder)
	if sErr != nil {
		logger.Error("Error finding any files", sErr)
		return
	}
	if contentSize == 0 {
		logger.Info("Cannot find any files")
		return
	}

	var appliedCondition bool
	if applyCondition {
		appliedCondition = contentSize >= *company.TotalUsageQuota
	} else {
		appliedCondition = true
	}

	if appliedCondition {
		dErr := s3Client.DeleteAllZipFilesWithPattern("Tally", companyFolder)
		if dErr != nil {
			logger.Error("Cannot delete files", dErr)
			return
		}
		meta := &models.FileMetadata{
			Id:          uuid.NewString(),
			CreatedAt:   time.Now().Format(time.RFC3339),
			FileName:    companyFolder,
			FileSize:    &contentSize,
			FileKey:     companyFolder + "/",
			CompanyId:   company.Id,
			FileTxnType: utils.PtrInt16(2), // 2 = delete
			FileTxnMeta: "Deleted files in S3",
		}
		if err := apiClient.InsertFileMetadata(meta); err != nil {
			logger.Error("Failed to insert upload metadata", err)
		}

		updateQuota := &models.UpdateUsageQuota{
			UsedQuota:   contentSize,
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
