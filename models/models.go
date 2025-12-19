package models

import (
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

type PresignUploadRequest struct {
	FileName    string  `json:"file_name"`
	FileSize    int64   `json:"file_size"`
	LocTag      string  `json:"loc_tag"`
	FileTxnType int16   `json:"file_txn_type"`
	FileTxnMeta *string `json:"file_txn_meta,omitempty"`
}

type PresignedUploadResponse struct {
	FileId    string `json:"file_id"`
	FileKey   string `json:"file_key"`
	UploadUrl string `json:"upload_url"`
}

type FileDeleteRequest struct {
	LocTag string `json:"loc_tag"`
}
