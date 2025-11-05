package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"shreshtasmg.in/sh_backups/logger"
)

type AppConfig struct {
	APIKey          string
	APIBaseUrl      string
	LocalFolderPath string
}

func Load() AppConfig {
	homeDir, err := os.UserHomeDir()
	var licPath string
	if err != nil {
		logger.Error("Could not get user home directory: %v", err)
	}
	licPath = filepath.Join(homeDir, "apikey.lic")
	_, err = os.Stat(licPath)
	if os.IsNotExist(err) {
		licPath = "apikey.lic"
	}

	_ = godotenv.Load(licPath)

	return AppConfig{
		APIKey:          must("API_KEY"),
		APIBaseUrl:      must("API_BASE_URL"),
		LocalFolderPath: must("LOCAL_FOLDER_PATH"),
	}
}

func must(key string) string {
	val := os.Getenv(key)
	if val == "" {
		logger.Error(fmt.Sprintf("Missing env var: %s", key), nil)
		os.Exit(1)
	}
	return val
}
