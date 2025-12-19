package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

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

// DeleteFiles is a placeholder for the actual implementation.
func (c *APIClient) DeleteFiles(apiKey, locTag string) error {
	// TODO: Implement actual API call to delete files
	fmt.Printf("Deleting files for APIKey: %s, LocTag: %s\n", apiKey, locTag)
	return nil
}

// GeneratePresignedUploadURL sends a request to the API to get a presigned URL for file upload.
func (c *APIClient) GeneratePresignedUploadURL(req *models.PresignUploadRequest) (*models.PresignedUploadResponse, error) {
	url := fmt.Sprintf("%s/api/v1/uploader/files", c.BaseURL)
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal presign upload request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", c.APIKey)

	resp, err := c.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send presign upload request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get presigned upload URL: status %d, response: %s", resp.StatusCode, respBody)
	}

	var presignedResp models.PresignedUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&presignedResp); err != nil {
		return nil, fmt.Errorf("failed to decode presigned upload response: %w", err)
	}

	return &presignedResp, nil
}

// UploadFileWithPresignedURL uploads a file using the provided presigned URL and fields.
func (c *APIClient) UploadFileWithPresignedURL(presignedResp *models.PresignedUploadResponse, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Optional but nice: set Content-Length
	fi, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// For presigned PUT URLs, just send the raw file as the body.
	req, err := http.NewRequest(http.MethodPut, presignedResp.UploadUrl, file)
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}

	req.ContentLength = fi.Size()

	// If you set a Content-Type when generating the presigned URL,
	// you MUST set the same header here. Otherwise, you can omit it
	// or set a generic one like below:
	// req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to upload file to S3: status %d, response: %s", resp.StatusCode, respBody)
	}

	return nil
}
