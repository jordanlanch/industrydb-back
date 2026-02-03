package backup

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Service handles database backups
type Service struct {
	s3Client       *s3.Client
	bucket         string
	databaseURL    string
	localBackupDir string
	retentionDays  int
}

// Config holds backup configuration
type Config struct {
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	AWSRegion          string
	S3Bucket           string
	DatabaseURL        string
	LocalBackupDir     string
	RetentionDays      int // Number of days to keep backups
}

// NewService creates a new backup service
func NewService(cfg Config) (*Service, error) {
	// Create AWS config
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.AWSRegion),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AWSAccessKeyID,
			cfg.AWSSecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(awsCfg)

	// Ensure local backup directory exists
	if err := os.MkdirAll(cfg.LocalBackupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	return &Service{
		s3Client:       s3Client,
		bucket:         cfg.S3Bucket,
		databaseURL:    cfg.DatabaseURL,
		localBackupDir: cfg.LocalBackupDir,
		retentionDays:  cfg.RetentionDays,
	}, nil
}

// BackupResult contains backup operation results
type BackupResult struct {
	Filename     string
	FileSize     int64
	S3Key        string
	Duration     time.Duration
	Compressed   bool
	UploadedToS3 bool
}

// CreateBackup creates a PostgreSQL backup and uploads it to S3
func (s *Service) CreateBackup(ctx context.Context) (*BackupResult, error) {
	start := time.Now()

	// Generate filename with timestamp
	timestamp := time.Now().UTC().Format("20060102-150405")
	filename := fmt.Sprintf("industrydb-backup-%s.sql.gz", timestamp)
	localPath := filepath.Join(s.localBackupDir, filename)

	// Create backup file
	file, err := os.Create(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup file: %w", err)
	}
	defer file.Close()

	// Create gzip writer
	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	// Execute pg_dump
	log.Printf("🔄 Starting database backup: %s", filename)
	cmd := exec.CommandContext(ctx, "pg_dump", s.databaseURL)
	cmd.Stdout = gzipWriter
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		os.Remove(localPath) // Clean up failed backup
		return nil, fmt.Errorf("pg_dump failed: %w", err)
	}

	// Close gzip writer to flush
	if err := gzipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// Get file size
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat backup file: %w", err)
	}

	result := &BackupResult{
		Filename:   filename,
		FileSize:   fileInfo.Size(),
		S3Key:      fmt.Sprintf("backups/%s", filename),
		Compressed: true,
		Duration:   time.Since(start),
	}

	// Upload to S3 if configured
	if s.bucket != "" {
		if err := s.uploadToS3(ctx, localPath, result.S3Key); err != nil {
			return result, fmt.Errorf("backup created locally but S3 upload failed: %w", err)
		}
		result.UploadedToS3 = true
		log.Printf("✅ Backup uploaded to S3: s3://%s/%s", s.bucket, result.S3Key)

		// Clean up old backups
		if err := s.cleanupOldBackups(ctx); err != nil {
			log.Printf("⚠️  Failed to cleanup old backups: %v", err)
		}
	}

	log.Printf("✅ Backup completed: %s (size: %d bytes, duration: %s)",
		filename, result.FileSize, result.Duration)

	return result, nil
}

// uploadToS3 uploads a file to S3
func (s *Service) uploadToS3(ctx context.Context, localPath, s3Key string) error {
	// Open file
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Upload to S3
	_, err = s.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(s.bucket),
		Key:          aws.String(s3Key),
		Body:         file,
		StorageClass: types.StorageClassStandardIa, // Infrequent Access for backups
	})
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	return nil
}

// cleanupOldBackups deletes backups older than retention period
func (s *Service) cleanupOldBackups(ctx context.Context) error {
	if s.retentionDays <= 0 {
		return nil // No retention policy
	}

	cutoffDate := time.Now().UTC().AddDate(0, 0, -s.retentionDays)

	// List objects in backups/ prefix
	result, err := s.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String("backups/"),
	})
	if err != nil {
		return fmt.Errorf("failed to list S3 objects: %w", err)
	}

	// Delete old backups
	var deleted int
	for _, obj := range result.Contents {
		if obj.LastModified.Before(cutoffDate) {
			_, err := s.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(s.bucket),
				Key:    obj.Key,
			})
			if err != nil {
				log.Printf("⚠️  Failed to delete old backup %s: %v", *obj.Key, err)
				continue
			}
			deleted++
			log.Printf("🗑️  Deleted old backup: %s (age: %d days)",
				*obj.Key, int(time.Since(*obj.LastModified).Hours()/24))
		}
	}

	if deleted > 0 {
		log.Printf("✅ Cleaned up %d old backups (retention: %d days)", deleted, s.retentionDays)
	}

	return nil
}

// ListBackups lists all backups in S3
func (s *Service) ListBackups(ctx context.Context) ([]BackupInfo, error) {
	if s.bucket == "" {
		return nil, fmt.Errorf("S3 bucket not configured")
	}

	result, err := s.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String("backups/"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list S3 objects: %w", err)
	}

	backups := make([]BackupInfo, 0, len(result.Contents))
	for _, obj := range result.Contents {
		backups = append(backups, BackupInfo{
			Key:          *obj.Key,
			Size:         *obj.Size,
			LastModified: *obj.LastModified,
			Age:          time.Since(*obj.LastModified),
		})
	}

	return backups, nil
}

// BackupInfo contains information about a backup
type BackupInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
	Age          time.Duration
}

// RestoreBackup downloads and restores a backup from S3
func (s *Service) RestoreBackup(ctx context.Context, s3Key string) error {
	// Download from S3
	log.Printf("🔄 Downloading backup from S3: %s", s3Key)
	result, err := s.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		return fmt.Errorf("failed to download from S3: %w", err)
	}
	defer result.Body.Close()

	// Read into buffer
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(result.Body); err != nil {
		return fmt.Errorf("failed to read backup data: %w", err)
	}

	// Create gzip reader
	gzipReader, err := gzip.NewReader(&buf)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	// Execute psql to restore
	log.Printf("🔄 Restoring database from backup...")
	cmd := exec.CommandContext(ctx, "psql", s.databaseURL)
	cmd.Stdin = gzipReader
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("psql restore failed: %w", err)
	}

	log.Printf("✅ Database restored successfully from: %s", s3Key)
	return nil
}
