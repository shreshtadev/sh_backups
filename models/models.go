package models

import (
	"bytes"
	"time"
)

const CustomTimeFormat = "2006-01-02T15:04:05"

type CustomTime struct {
	time.Time
}

func (ct *CustomTime) UnmarshalJSON(b []byte) (err error) {
	s := string(b)
	if s == "null" {
		return nil
	}
	// Remove quotes
	s = s[1 : len(s)-1]
	// Handle empty string case
	if s == "" {
		ct.Time = time.Time{}
		return nil
	}
	ct.Time, err = time.Parse(CustomTimeFormat, s)
	return
}

type Company struct {
	Id              string      `json:"id"`
	CreatedAt       *CustomTime `json:"created_at"`
	CompanyName     string      `json:"company_name"`
	CompanySlug     string      `json:"company_slug"`
	CompanyApiKey   string      `json:"company_api_key"`
	StartDate       *CustomTime `json:"start_date"`
	EndDate         *CustomTime `json:"end_date"`
	TotalUsageQuota *int64      `json:"total_usage_quota"`
	UsedQuota       *int64      `json:"used_quota"`
	BaseURL         string      `json:"api_base_url"`
}

type FileMetadata struct {
	Id          string `json:"id"`
	CreatedAt   string `json:"created_at"`
	FileName    string `json:"file_name"`
	FileSize    *int64 `json:"file_size"`
	FileKey     string `json:"file_key"`
	CompanyId   string `json:"company_id"`
	FileTxnType *int16 `json:"file_txn_type"`
	FileTxnMeta string `json:"file_txn_meta"`
}

type UpdateUsageQuota struct {
	UsedQuota   int64 `json:"used_quota"`
	FileTxnType int16 `json:"file_txn_type"`
}

type PresignUploadRequest struct {
	FileName    string `json:"file_name"`
	ContentSize int64  `json:"content_size"`
	LocTag      string `json:"loc_tag"`
}

type PresignedUploadResponse struct {
	// URL is the endpoint to which the multipart form data should be posted.
	URL string `json:"url"`
	// Fields is a map containing the necessary form fields, including the signed policy.
	Fields map[string]string `json:"fields"`
}

type UploadRequest struct {
	Key            string        `form:"key"`
	XAmzAlgorithm  string        `form:"x-amz-algorithm"`
	XAmzCredential string        `form:"x-amz-credential"`
	XAmzDate       string        `form:"x-amz-date"`
	Policy         string        `form:"policy"`
	XAmzSignature  string        `form:"x-amz-signature"`
	ContentType    string        `form:"Content-Type"`
	FileToUpload   *bytes.Buffer `form:"file"`
}

type FileDeleteRequest struct {
	LocTag string `json:"loc_tag"`
}

type FolderInfoResponse struct {
	FolderPath        string `json:"folder_path"`
	TotalSize         int64  `json:"total_size"`
	TotalSizeReadable string `json:"total_size_readable"`
}
