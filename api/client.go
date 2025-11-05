package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"reflect"

	"shreshtasmg.in/sh_backups/config"
	"shreshtasmg.in/sh_backups/logger"
	"shreshtasmg.in/sh_backups/models"
)

type APIClient struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
}

func NewAPIClient(baseURL, apiKey string) *APIClient {
	return &APIClient{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Client:  &http.Client{},
	}
}

const (
	locTag = "TallyBackups"
)

func (c *APIClient) DeleteFiles(apiKey, folderPrefix string) error {
	url := fmt.Sprintf("%s/api/companies/delete/files", c.BaseURL)
	deleteReq := &models.FileDeleteRequest{
		LocTag: locTag,
	}
	body, err := json.Marshal(deleteReq)
	if err != nil {
		logger.Error("Failed to marshal file metadata", err)
		return err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		logger.Error("Failed to create new HTTP request", err)
		return err
	}
	req.Header.Set("X-Company-Api-Key", apiKey)
	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		err1 := config.ParseErrorBody(resp.Status, respBody)
		logger.ErrorFn(err1)
		err := fmt.Errorf("unexpected status: %d", resp.StatusCode)
		logger.Error("Unexpected status when deleting files", err)
		return err
	}
	logger.Info("Deleted files " + folderPrefix + " successfully")
	return nil
}

func (c *APIClient) GetFolderSize(apiKey, folderPrefix string) (*models.FolderInfoResponse, error) {
	url := fmt.Sprintf("%s/api/filemeta/folder/size", c.BaseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Error("Failed to create new HTTP request", err)
		return nil, err
	}
	req.Header.Set("X-Company-Api-Key", apiKey)
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		err1 := config.ParseErrorBody(resp.Status, respBody)
		logger.ErrorFn(err1)
		err := fmt.Errorf("unexpected status: %d", resp.StatusCode)
		logger.Error("Unexpected status when getting folder size", err)
		return nil, err
	}
	var folderSize *models.FolderInfoResponse
	err = json.NewDecoder(resp.Body).Decode(&folderSize)
	if err != nil {
		logger.Error("Failed to decode folder size response", err)
		return nil, err
	}
	logger.Info("Got folder size " + folderPrefix + " successfully")
	return folderSize, nil
}

func (c *APIClient) GeneratePresignURL(apiKey, path string) (*models.PresignedUploadResponse, error) {
	// Write Request Presign
	url := fmt.Sprintf("%s/api/companies/generate/presigned/url/upload", c.BaseURL)
	filePathWithExt := filepath.Base(path)
	fileInfo, err := os.Stat(path)
	if err != nil {
		logger.Error("Error getting file info", err)
		return nil, err
	}
	fileSize := fileInfo.Size()
	presignReq := &models.PresignUploadRequest{
		FileName:    filePathWithExt,
		ContentSize: fileSize,
		LocTag:      locTag,
	}
	body, err := json.Marshal(presignReq)
	if err != nil {
		logger.Error("Failed to marshal file metadata", err)
		return nil, err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		logger.Error("Failed to create new HTTP request", err)
		return nil, err
	}
	req.Header.Set("X-Company-Api-Key", c.APIKey)
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		err1 := config.ParseErrorBody(resp.Status, respBody)
		logger.ErrorFn(err1)
		err := fmt.Errorf("unexpected status: %d", resp.StatusCode)
		logger.Error("Unexpected status when fetching presign upload url", err)
		return nil, err
	}
	// Create an instance of our struct.
	var presignedResponse *models.PresignedUploadResponse

	// Unmarshal the JSON string into the struct.
	err = json.NewDecoder(resp.Body).Decode(&presignedResponse)
	if err != nil {
		logger.Error("Error unmarshalling JSON", err)
	}
	return presignedResponse, nil
}

func (c *APIClient) UploadFile(apiKey string, filePath string) error {
	requestPresignUpload, err := c.GeneratePresignURL(apiKey, filePath)
	if err != nil {
		return err
	}
	uploadRequest := &models.UploadRequest{
		Key:            requestPresignUpload.Fields["key"],
		XAmzAlgorithm:  requestPresignUpload.Fields["x-amz-algorithm"],
		XAmzCredential: requestPresignUpload.Fields["x-amz-credential"],
		XAmzDate:       requestPresignUpload.Fields["x-amz-date"],
		Policy:         requestPresignUpload.Fields["policy"],
		XAmzSignature:  requestPresignUpload.Fields["x-amz-signature"],
		ContentType:    requestPresignUpload.Fields["Content-Type"],
	}
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	val := reflect.ValueOf(uploadRequest).Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		formTag := field.Tag.Get("form")

		// Skip the file field, as it needs to be handled separately and added last.
		if formTag == "file" {
			continue
		}

		// Get the value of the field and add it to the form writer.
		fieldValue := val.Field(i).String()
		if formTag != "" && fieldValue != "" {
			if err := writer.WriteField(formTag, fieldValue); err != nil {
				return fmt.Errorf("failed to write field %s: %w", formTag, err)
			}
		}
	}
	fileField, found := typ.FieldByName("FileToUpload")
	if !found {
		return fmt.Errorf("UploadRequest struct missing FileToUpload field")
	}
	fileFormFieldName := fileField.Tag.Get("form")
	// 3. Create a new form file field and write the file content.
	part, err := writer.CreateFormFile(fileFormFieldName, filePath)
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err = io.Copy(part, file); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// 4. Close the multipart writer to finalize the request body.
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}
	req, err := http.NewRequest("POST", requestPresignUpload.URL, body)
	if err != nil {
		return fmt.Errorf("failed to create new HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Company-Api-Key", c.APIKey)
	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		err1 := config.ParseErrorBody(resp.Status, respBody)
		logger.ErrorFn(err1)
		err := fmt.Errorf("unexpected status: %d", resp.StatusCode)
		logger.Error("Unexpected status when uploading file", err)
		return err
	}

	return nil
}

// FindCompanyByAPIKey
func (c *APIClient) FindCompanyByAPIKey(apiKey string) (*models.Company, error) {
	url := fmt.Sprintf("%s/api/companies/by-api-key", c.BaseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Error("Failed to create new HTTP request", err)
		return nil, err
	}
	req.Header.Set("X-Company-Api-Key", c.APIKey)
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("unexpected status: %d", resp.StatusCode)
		logger.Error("Unexpected status when fetching company by API key", err)
		return nil, err
	}
	var company models.Company
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &company); err != nil {
		err1 := config.ParseErrorBody(resp.Status, respBody)
		logger.ErrorFn(err1)
		logger.Error("Failed to decode company response", err)
		return nil, err
	}
	return &company, nil
}

// InsertFileMetadata
func (c *APIClient) InsertFileMetadata(meta *models.FileMetadata) error {
	url := fmt.Sprintf("%s/api/filemeta", c.BaseURL)
	body, err := json.Marshal(meta)
	if err != nil {
		logger.Error("Failed to marshal file metadata", err)
		return err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		logger.Error("Failed to create new HTTP request for inserting file metadata", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Company-Api-Key", c.APIKey)
	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		err1 := config.ParseErrorBody(resp.Status, respBody)
		logger.ErrorFn(err1)
		err := fmt.Errorf("unexpected status: %d", resp.StatusCode)
		logger.Error("Unexpected status when inserting file metadata", err)
		return err
	}
	return nil
}

// UpdateCompanyQuota
func (c *APIClient) UpdateCompanyQuota(usageQuota *models.UpdateUsageQuota) error {
	url := fmt.Sprintf("%s/api/companies/quota", c.BaseURL)
	body, err := json.Marshal(usageQuota)
	if err != nil {
		logger.Error("Failed to marshal usage quota data", err)
		return err
	}
	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(body))
	if err != nil {
		logger.Error("Failed to create new HTTP request for updating company quota", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Company-Api-Key", c.APIKey)
	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		err1 := config.ParseErrorBody(resp.Status, respBody)
		logger.ErrorFn(err1)
		err := fmt.Errorf("unexpected status: %d", resp.StatusCode)
		logger.Error("Unexpected status when updating company quota", err)
		return err
	}
	return nil
}
