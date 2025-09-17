# SH Backups

This is a command-line utility for backing up local files to an AWS S3 bucket. It's designed to be run periodically to synchronize the latest version of a file (specifically a .zip file matching a certain pattern) to S3.

## Functionality

The application has two main modes of operation:

1.  **Registration (`--register`)**: This mode is used to set up a new company. It interactively prompts the user for their company name, the local folder path to monitor, and their AWS S3 credentials. Upon successful registration with the backend API, it saves a unique `apikey.lic` file in the current directory. This key is used for all subsequent operations.

2.  **Backup (Default)**: This is the primary mode. It performs the following steps:
    *   Loads configuration, including the API key from the environment.
    *   Fetches company details and usage quota from the API.
    *   Finds the most recent `.zip` file containing the name "Tally" in the configured local folder.
    *   Compares the local file with the corresponding file in the S3 bucket. If the local file is newer or doesn't exist in the bucket, it proceeds. Otherwise, it exits.
    *   If an older backup file exists in S3, it is deleted to make space for the new one.
    *   Uploads the new local file to the S3 bucket.
    *   Updates the company's usage quota and file transaction logs via the API for both deletions and uploads.

## Setup & Configuration

Before running the application, you need to configure the following environment variables. You can set them directly in your shell or create a `.env` file in the root of the project.

*   `API_KEY`: The API key for authenticating with the backend service. For the initial registration, you can use a temporary key if required by the API, but for backup operations, you must use the key generated and saved in the `apikey.lic` file.
*   `API_BASE_URL`: The base URL of the backend API service (e.g., `http://localhost:8080`).

## How to Build

To build the application, run the following command from the project root:

```sh
go build -o sh-backups .
```

This will create an executable file named `sh-backups` in the current directory.

## How to Run

### First-Time Registration

To register a new company, run the executable with the `--register` flag:

```sh
./sh-backups --register
```

You will be prompted to enter the company name, local folder path for backups, and your AWS S3 details. A `apikey.lic` file will be created upon success.

### Running a Backup

To run a backup, first copy the key from the `apikey.lic` file and set it as the `API_KEY` environment variable. Then, run the executable without any flags:

**For Linux/macOS (bash/zsh):**
```sh
export API_KEY="your-generated-api-key"
export API_BASE_URL="http://your-api-url"

./sh-backups
```

**For Windows (PowerShell):**
```powershell
$env:API_KEY="your-generated-api-key"
$env:API_BASE_URL="http://your-api-url"

./sh-backups.exe
```

