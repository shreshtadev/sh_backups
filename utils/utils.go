package utils

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func PtrInt16(v int16) *int16 {
	return &v
}

func PtrInt64(v int64) *int64 {
	return &v
}

var nonAlphanumeric = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func Slugify(input string) string {
	// Lowercase the string
	slug := strings.ToLower(input)

	// Replace non-alphanumeric characters with hyphens
	slug = nonAlphanumeric.ReplaceAllString(slug, "-")

	// Trim leading/trailing hyphens
	slug = strings.Trim(slug, "-")

	return slug
}

func FindZipFileWithPatternAndLatestDate(folder string) (string, int64, error) {
	var match string
	var size int64
	var latestDate time.Time

	// Compile regex for Tallybackupason files
	regex := regexp.MustCompile(`^Tallybackupason(\d{2})(\d{2})(\d{4})\.zip$`)

	err := filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		name := info.Name()
		if strings.HasSuffix(name, ".zip") {
			// Check for Tallybackupason pattern
			matches := regex.FindStringSubmatch(name)
			if len(matches) > 3 {
				dateStr := matches[1] + matches[2] + matches[3]
				fileDate, err := time.Parse("02012006", dateStr)
				if err == nil && fileDate.After(latestDate) {
					match = path
					size = info.Size()
					latestDate = fileDate
				}
			} else if match == "" {
				// If not Tallybackupason, just pick the first match
				match = path
				size = info.Size()
			}
		}
		return nil
	})

	if err != nil {
		return "", 0, err
	}
	if match == "" {
		return "", 0, os.ErrNotExist
	}
	return match, size, nil
}
