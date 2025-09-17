package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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

// FindCompanyByAPIKey
func (c *APIClient) FindCompanyByAPIKey(apiKey string) (*models.Company, error) {
	url := fmt.Sprintf("%s/companies/by-api-key", c.BaseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Error("Failed to create new HTTP request", err)
		return nil, err
	}
	req.Header.Set("company_api_key", c.APIKey)
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
		logger.Error("Failed to decode company response", err)
		return nil, err
	}
	return &company, nil
}

func (c *APIClient) RegisterCompany(company models.RegisterCompany) (*models.Company, error) {
	url := fmt.Sprintf("%s/register/company", c.BaseURL)
	body, err := json.Marshal(company)
	if err != nil {
		logger.Error("Failed to marshal company data", err)
		return nil, err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		logger.Error("Failed to create new HTTP request for company registration", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, string(respBody))
		logger.Error("Unexpected status when registering company", err)
		return nil, err
	}
	var cmp models.Company
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &cmp); err != nil {
		logger.Error("Failed to decode company registration response", err)
		return nil, err
	}

	return &cmp, nil
}

// InsertFileMetadata
func (c *APIClient) InsertFileMetadata(meta *models.FileMetadata) error {
	url := fmt.Sprintf("%s/filemeta", c.BaseURL)
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
	req.Header.Set("company_api_key", c.APIKey)
	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, string(respBody))
		logger.Error("Unexpected status when inserting file metadata", err)
		return err
	}
	return nil
}

// UpdateCompanyQuota
func (c *APIClient) UpdateCompanyQuota(usageQuota *models.UpdateUsageQuota) error {
	url := fmt.Sprintf("%s/companies/quota", c.BaseURL)
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
	req.Header.Set("company_api_key", c.APIKey)
	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, string(respBody))
		logger.Error("Unexpected status when updating company quota", err)
		return err
	}
	return nil
}
