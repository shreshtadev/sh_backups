package s3

import (
	"context"
	"os"

	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"shreshtasmg.in/sh_backups/logger"
	"shreshtasmg.in/sh_backups/utils"
)

type S3Client struct {
	Client *s3.Client
	Bucket string
}

func New(region, accessKey, secretKey, bucket string) (*S3Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		logger.Error("Failed to load AWS config", err)
		return nil, err
	}
	return &S3Client{
		Client: s3.NewFromConfig(cfg),
		Bucket: bucket,
	}, nil
}

func (s *S3Client) ListZipKeysWithPattern(pattern, companyName string) ([]string, error) {
	var keys []string
	underFolder := strings.Join([]string{utils.Slugify(companyName), pattern}, "/")
	paginator := s3.NewListObjectsV2Paginator(s.Client, &s3.ListObjectsV2Input{
		Bucket: &s.Bucket,
		Prefix: &underFolder,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			logger.Error("Failed to get next page of S3 objects", err)
			return nil, err
		}
		for _, obj := range page.Contents {
			if strings.HasSuffix(*obj.Key, ".zip") {
				keys = append(keys, *obj.Key)
			}
		}
	}
	return keys, nil
}

func (s *S3Client) GetTotalContentLength(companyName string) (int64, error) {
	var total int64
	prefix := strings.Join([]string{utils.Slugify(companyName)}, "/")
	paginator := s3.NewListObjectsV2Paginator(s.Client, &s3.ListObjectsV2Input{
		Bucket: &s.Bucket,
		Prefix: &prefix,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			logger.Error("Failed to get next page of S3 objects", err)
			return 0, err
		}
		for _, obj := range page.Contents {
			total += *obj.Size
		}
	}
	return total, nil
}

func (s *S3Client) DeleteAllZipFilesWithPattern(pattern, companyName string) error {
	keys, err := s.ListZipKeysWithPattern(pattern, companyName)
	if err != nil {
		return err
	}
	for _, key := range keys {
		_, err := s.Client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
			Bucket: &s.Bucket,
			Key:    &key,
		})
		if err != nil {
			logger.Error("Failed to delete object from S3", err)
			return err
		}
	}
	return nil
}

func (s *S3Client) UploadFile(key, path, companyName string) error {
	file, err := os.Open(path)
	if err != nil {
		logger.Error("Failed to open file for upload", err)
		return err
	}
	defer file.Close()
	keys := strings.Join([]string{utils.Slugify(companyName), key}, "/")

	_, err = s.Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: &s.Bucket,
		Key:    &keys,
		Body:   file,
	})
	if err != nil {
		logger.Error("Failed to put object to S3", err)
	}
	return err
}

func (s *S3Client) DeleteFile(key string) error {
	_, err := s.Client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: &s.Bucket,
		Key:    &key,
	})
	if err != nil {
		logger.Error("Failed to delete folder from S3", err)
		return err
	}
	return nil
}

func (s *S3Client) GetFileMetadata(keys *string) (*s3.HeadObjectOutput, error) {
	output, err := s.Client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: &s.Bucket,
		Key:    keys,
	})
	if err != nil {
		logger.Error("Failed to get file metadata from S3", err)
		return nil, err
	}
	return output, nil
}
