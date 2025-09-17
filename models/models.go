package models

import "time"

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
	CompanyApiKey   string      `json:"company_api_key"`
	LocalFolder     string      `json:"local_folder_path"`
	StartDate       *CustomTime `json:"start_date"`
	EndDate         *CustomTime `json:"end_date"`
	TotalUsageQuota *int64      `json:"total_usage_quota"`
	UsedQuota       *int64      `json:"used_quota"`
	BucketName      string      `json:"aws_bucket_name"`
	Region          string      `json:"aws_bucket_region"`
	AccessKey       string      `json:"aws_access_key"`
	SecretKey       string      `json:"aws_secret_key"`
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

type RegisterCompany struct {
	CompanyName string `json:"company_name"`
	LocalFolder string `json:"local_folder_path"`
	BucketName  string `json:"aws_bucket_name"`
	Region      string `json:"aws_bucket_region"`
	AccessKey   string `json:"aws_access_key"`
	SecretKey   string `json:"aws_secret_key"`
}

type UpdateUsageQuota struct {
	UsedQuota   int64 `json:"used_quota"`
	FileTxnType int16 `json:"file_txn_type"`
}
