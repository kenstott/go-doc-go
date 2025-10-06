package contentsource

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// S3ContentSource implements ContentSource for Amazon S3
type S3ContentSource struct {
	name               string
	bucketName         string
	prefix             string
	regionName         string
	awsAccessKeyID     string
	awsSecretAccessKey string
	awsSessionToken    string
	assumeRoleARN      string
	endpointURL        string
	includeExtensions  []string
	excludeExtensions  []string
	includePrefixes    []string
	excludePrefixes    []string
	includePatterns    []*regexp.Regexp
	excludePatterns    []*regexp.Regexp
	recursive          bool
	maxDepth           int
	detectMimeType     bool
	tempDir            string
	deleteAfterProcess bool
	maxLinkDepth       int
	localLinkMode      string
	client             *s3.Client
	contentCache       map[string]*DocumentContent
}

// S3Config holds configuration for S3ContentSource
type S3Config struct {
	Name               string   `json:"name"`
	BucketName         string   `json:"bucket_name"`
	Prefix             string   `json:"prefix"`
	RegionName         string   `json:"region_name"`
	AWSAccessKeyID     string   `json:"aws_access_key_id"`
	AWSSecretAccessKey string   `json:"aws_secret_access_key"`
	AWSSessionToken    string   `json:"aws_session_token"`
	AssumeRoleARN      string   `json:"assume_role_arn"`
	EndpointURL        string   `json:"endpoint_url"`
	IncludeExtensions  []string `json:"include_extensions"`
	ExcludeExtensions  []string `json:"exclude_extensions"`
	IncludePrefixes    []string `json:"include_prefixes"`
	ExcludePrefixes    []string `json:"exclude_prefixes"`
	IncludePatterns    []string `json:"include_patterns"`
	ExcludePatterns    []string `json:"exclude_patterns"`
	Recursive          bool     `json:"recursive"`
	MaxDepth           int      `json:"max_depth"`
	DetectMimeType     bool     `json:"detect_mimetype"`
	TempDir            string   `json:"temp_dir"`
	DeleteAfterProcess bool     `json:"delete_after_processing"`
	MaxLinkDepth       int      `json:"max_link_depth"`
	LocalLinkMode      string   `json:"local_link_mode"`
}

// NewS3ContentSource creates a new S3 content source
func NewS3ContentSource(cfg map[string]interface{}) (*S3ContentSource, error) {
	// Extract configuration
	name, _ := cfg["name"].(string)
	if name == "" {
		name = "unnamed-s3-source"
	}

	bucketName, _ := cfg["bucket_name"].(string)
	if bucketName == "" {
		return nil, fmt.Errorf("bucket_name is required")
	}

	prefix, _ := cfg["prefix"].(string)
	regionName, _ := cfg["region_name"].(string)
	awsAccessKeyID, _ := cfg["aws_access_key_id"].(string)
	awsSecretAccessKey, _ := cfg["aws_secret_access_key"].(string)
	awsSessionToken, _ := cfg["aws_session_token"].(string)
	assumeRoleARN, _ := cfg["assume_role_arn"].(string)
	endpointURL, _ := cfg["endpoint_url"].(string)

	// Extract extension lists
	var includeExts, excludeExts, includePrefixes, excludePrefixes []string
	if val, ok := cfg["include_extensions"].([]interface{}); ok {
		for _, ext := range val {
			if s, ok := ext.(string); ok {
				includeExts = append(includeExts, s)
			}
		}
	}
	if val, ok := cfg["exclude_extensions"].([]interface{}); ok {
		for _, ext := range val {
			if s, ok := ext.(string); ok {
				excludeExts = append(excludeExts, s)
			}
		}
	}
	if val, ok := cfg["include_prefixes"].([]interface{}); ok {
		for _, p := range val {
			if s, ok := p.(string); ok {
				includePrefixes = append(includePrefixes, s)
			}
		}
	}
	if val, ok := cfg["exclude_prefixes"].([]interface{}); ok {
		for _, p := range val {
			if s, ok := p.(string); ok {
				excludePrefixes = append(excludePrefixes, s)
			}
		}
	}

	// Extract patterns
	var includePatterns, excludePatterns []*regexp.Regexp
	if val, ok := cfg["include_patterns"].([]interface{}); ok {
		for _, p := range val {
			if s, ok := p.(string); ok {
				if re, err := regexp.Compile(s); err == nil {
					includePatterns = append(includePatterns, re)
				}
			}
		}
	}
	if val, ok := cfg["exclude_patterns"].([]interface{}); ok {
		for _, p := range val {
			if s, ok := p.(string); ok {
				if re, err := regexp.Compile(s); err == nil {
					excludePatterns = append(excludePatterns, re)
				}
			}
		}
	}

	recursive := true
	if val, ok := cfg["recursive"].(bool); ok {
		recursive = val
	}

	maxDepth := 10
	if val, ok := cfg["max_depth"].(float64); ok {
		maxDepth = int(val)
	} else if val, ok := cfg["max_depth"].(int); ok {
		maxDepth = val
	}

	detectMimeType := true
	if val, ok := cfg["detect_mimetype"].(bool); ok {
		detectMimeType = val
	}

	tempDir, _ := cfg["temp_dir"].(string)
	if tempDir == "" {
		tempDir = os.TempDir()
	}

	deleteAfterProcess := true
	if val, ok := cfg["delete_after_processing"].(bool); ok {
		deleteAfterProcess = val
	}

	maxLinkDepth := 3
	if val, ok := cfg["max_link_depth"].(float64); ok {
		maxLinkDepth = int(val)
	} else if val, ok := cfg["max_link_depth"].(int); ok {
		maxLinkDepth = val
	}

	localLinkMode := "relative"
	if val, ok := cfg["local_link_mode"].(string); ok {
		localLinkMode = val
	}

	// Initialize AWS SDK
	client, err := initializeS3Client(
		regionName,
		awsAccessKeyID,
		awsSecretAccessKey,
		awsSessionToken,
		assumeRoleARN,
		endpointURL,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize S3 client: %w", err)
	}

	return &S3ContentSource{
		name:               name,
		bucketName:         bucketName,
		prefix:             prefix,
		regionName:         regionName,
		awsAccessKeyID:     awsAccessKeyID,
		awsSecretAccessKey: awsSecretAccessKey,
		awsSessionToken:    awsSessionToken,
		assumeRoleARN:      assumeRoleARN,
		endpointURL:        endpointURL,
		includeExtensions:  includeExts,
		excludeExtensions:  excludeExts,
		includePrefixes:    includePrefixes,
		excludePrefixes:    excludePrefixes,
		includePatterns:    includePatterns,
		excludePatterns:    excludePatterns,
		recursive:          recursive,
		maxDepth:           maxDepth,
		detectMimeType:     detectMimeType,
		tempDir:            tempDir,
		deleteAfterProcess: deleteAfterProcess,
		maxLinkDepth:       maxLinkDepth,
		localLinkMode:      localLinkMode,
		client:             client,
		contentCache:       make(map[string]*DocumentContent),
	}, nil
}

// FetchDocument fetches a document from S3
func (s *S3ContentSource) FetchDocument(sourceID string) (*DocumentContent, error) {
	// Extract bucket and key from sourceID (s3://bucket/key or just key)
	bucket, key := extractBucketAndKey(sourceID)
	if bucket == "" {
		bucket = s.bucketName
		key = sourceID
	}

	// Normalize key
	key = strings.TrimPrefix(key, "/")

	// Create fully qualified source
	qualifiedSource := fmt.Sprintf("s3://%s/%s", bucket, key)

	// Check cache
	cacheKey := fmt.Sprintf("%s/%s", bucket, key)
	if cached, ok := s.contentCache[cacheKey]; ok {
		return cached, nil
	}

	ctx := context.Background()

	// Get object metadata
	headOutput, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("object not found: %s: %w", qualifiedSource, err)
	}

	contentType := ""
	if headOutput.ContentType != nil {
		contentType = *headOutput.ContentType
	}

	var lastModified float64
	if headOutput.LastModified != nil {
		lastModified = float64(headOutput.LastModified.Unix())
	}

	size := headOutput.ContentLength
	etag := ""
	if headOutput.ETag != nil {
		etag = strings.Trim(*headOutput.ETag, "\"")
	}

	// Get object content
	getOutput, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("error downloading %s: %w", qualifiedSource, err)
	}
	defer getOutput.Body.Close()

	// Read content
	data, err := io.ReadAll(getOutput.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading content: %w", err)
	}

	// Try to decode as UTF-8
	content := string(data)
	isBinary := false
	if !isValidUTF8(data) {
		isBinary = true
	}

	// Save to temp file if binary
	var tempFilePath string
	if isBinary {
		fileName := filepath.Base(key)
		tempFilePath = filepath.Join(s.tempDir, fmt.Sprintf("s3_%s_%s", etag, fileName))
		if err := os.WriteFile(tempFilePath, data, 0644); err != nil {
			return nil, fmt.Errorf("error writing temp file: %w", err)
		}
	}

	// Detect document type
	docType := detectDocType(key, contentType, isBinary)

	// Create metadata
	metadata := map[string]interface{}{
		"bucket":        bucket,
		"key":           key,
		"content_type":  contentType,
		"last_modified": lastModified,
		"size":          size,
		"etag":          etag,
		"is_binary":     isBinary,
		"temp_file_path": tempFilePath,
		"filename":      filepath.Base(key),
		"extension":     strings.TrimPrefix(filepath.Ext(key), "."),
		"url":           qualifiedSource,
	}

	// Generate content hash
	var contentHash string
	if !isBinary {
		contentHash = GetContentHash(content)
	} else {
		contentHash = etag
	}

	doc := &DocumentContent{
		ID:          qualifiedSource,
		Content:     content,
		BinaryPath:  tempFilePath,
		Metadata:    metadata,
		ContentHash: contentHash,
		DocType:     docType,
	}

	// Cache the result
	s.contentCache[cacheKey] = doc

	return doc, nil
}

// ListDocuments lists available documents in S3
func (s *S3ContentSource) ListDocuments() ([]DocumentInfo, error) {
	ctx := context.Background()
	var results []DocumentInfo

	// Create paginator
	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket:  aws.String(s.bucketName),
		Prefix:  aws.String(s.prefix),
		MaxKeys: aws.Int32(1000),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("error listing objects: %w", err)
		}

		for _, obj := range page.Contents {
			if obj.Key == nil {
				continue
			}

			key := *obj.Key

			// Skip directory markers
			if strings.HasSuffix(key, "/") {
				continue
			}

			// Apply filters
			if !s.shouldIncludeObject(key) {
				continue
			}

			// Create fully qualified source
			qualifiedSource := fmt.Sprintf("s3://%s/%s", s.bucketName, key)

			var lastModified float64
			if obj.LastModified != nil {
				lastModified = float64(obj.LastModified.Unix())
			}

			size := obj.Size
			etag := ""
			if obj.ETag != nil {
				etag = strings.Trim(*obj.ETag, "\"")
			}

			extension := strings.TrimPrefix(filepath.Ext(key), ".")

			metadata := map[string]interface{}{
				"bucket":        s.bucketName,
				"key":           key,
				"last_modified": lastModified,
				"size":          size,
				"etag":          etag,
				"filename":      filepath.Base(key),
				"extension":     extension,
				"url":           qualifiedSource,
			}

			// Guess document type based on extension
			docType := guessDocType(extension)

			results = append(results, DocumentInfo{
				ID:       qualifiedSource,
				Metadata: metadata,
				DocType:  docType,
			})
		}
	}

	return results, nil
}

// HasChanged checks if an S3 object has changed
func (s *S3ContentSource) HasChanged(sourceID string, lastModified interface{}) (bool, error) {
	// Extract bucket and key
	bucket, key := extractBucketAndKey(sourceID)
	if bucket == "" {
		bucket = s.bucketName
		key = sourceID
	}

	key = strings.TrimPrefix(key, "/")

	// Check cache first
	cacheKey := fmt.Sprintf("%s/%s", bucket, key)
	if cached, ok := s.contentCache[cacheKey]; ok {
		if lastMod, ok := cached.Metadata["last_modified"].(float64); ok {
			if lastModTime, ok := lastModified.(float64); ok {
				return lastMod > lastModTime, nil
			}
		}
	}

	ctx := context.Background()

	// Get object metadata
	headOutput, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return true, nil // Assume changed if we can't check
	}

	if headOutput.LastModified == nil {
		return true, nil
	}

	currentModified := float64(headOutput.LastModified.Unix())

	if lastModified == nil {
		return true, nil
	}

	var lastModTime float64
	switch v := lastModified.(type) {
	case float64:
		lastModTime = v
	case int64:
		lastModTime = float64(v)
	case int:
		lastModTime = float64(v)
	default:
		return true, nil
	}

	return currentModified > lastModTime, nil
}

// shouldIncludeObject checks if an S3 object should be included based on filters
func (s *S3ContentSource) shouldIncludeObject(key string) bool {
	// Get file extension
	extension := strings.TrimPrefix(filepath.Ext(key), ".")

	// Check exclude extensions
	if len(s.excludeExtensions) > 0 {
		for _, ext := range s.excludeExtensions {
			if extension == ext {
				return false
			}
		}
	}

	// Check include extensions
	if len(s.includeExtensions) > 0 {
		found := false
		for _, ext := range s.includeExtensions {
			if extension == ext {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check exclude prefixes
	for _, prefix := range s.excludePrefixes {
		if strings.HasPrefix(key, prefix) {
			return false
		}
	}

	// Check include prefixes
	if len(s.includePrefixes) > 0 {
		found := false
		for _, prefix := range s.includePrefixes {
			if strings.HasPrefix(key, prefix) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check exclude patterns
	for _, pattern := range s.excludePatterns {
		if pattern.MatchString(key) {
			return false
		}
	}

	// Check include patterns
	if len(s.includePatterns) > 0 {
		found := false
		for _, pattern := range s.includePatterns {
			if pattern.MatchString(key) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// initializeS3Client creates an S3 client with the given configuration
func initializeS3Client(regionName, accessKeyID, secretAccessKey, sessionToken, assumeRoleARN, endpointURL string) (*s3.Client, error) {
	ctx := context.Background()

	var cfg aws.Config
	var err error

	// Load configuration
	if accessKeyID != "" && secretAccessKey != "" {
		// Use provided credentials
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(regionName),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				accessKeyID,
				secretAccessKey,
				sessionToken,
			)),
		)
	} else {
		// Use default credential chain
		cfg, err = config.LoadDefaultConfig(ctx, config.WithRegion(regionName))
	}
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	// Assume role if specified
	if assumeRoleARN != "" {
		stsClient := sts.NewFromConfig(cfg)
		result, err := stsClient.AssumeRole(ctx, &sts.AssumeRoleInput{
			RoleArn:         aws.String(assumeRoleARN),
			RoleSessionName: aws.String("DocumentPointerSession"),
		})
		if err != nil {
			return nil, fmt.Errorf("unable to assume role: %w", err)
		}

		cfg.Credentials = credentials.NewStaticCredentialsProvider(
			*result.Credentials.AccessKeyId,
			*result.Credentials.SecretAccessKey,
			*result.Credentials.SessionToken,
		)
	}

	// Create S3 client
	var client *s3.Client
	if endpointURL != "" {
		client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpointURL)
		})
	} else {
		client = s3.NewFromConfig(cfg)
	}

	return client, nil
}

// extractBucketAndKey extracts bucket and key from S3 URI
func extractBucketAndKey(s3URI string) (string, string) {
	if strings.HasPrefix(s3URI, "s3://") {
		parts := strings.SplitN(strings.TrimPrefix(s3URI, "s3://"), "/", 2)
		if len(parts) == 2 {
			return parts[0], parts[1]
		}
		return parts[0], ""
	}
	return "", s3URI
}

// isValidUTF8 checks if data is valid UTF-8
func isValidUTF8(data []byte) bool {
	for _, b := range data {
		// Check for binary content (null bytes)
		if b == 0 {
			return false
		}
	}
	return true
}

// detectDocType detects document type from key and content type
func detectDocType(key, contentType string, isBinary bool) string {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(key), "."))

	if isBinary {
		switch ext {
		case "md", "markdown":
			return "markdown"
		case "html", "htm":
			return "html"
		case "pdf":
			return "pdf"
		case "docx":
			return "docx"
		case "pptx":
			return "pptx"
		case "xlsx":
			return "xlsx"
		default:
			return "binary"
		}
	}

	// Text-based detection
	switch ext {
	case "json":
		return "json"
	case "csv":
		return "csv"
	case "xml":
		return "xml"
	case "md", "markdown":
		return "markdown"
	case "html", "htm":
		return "html"
	case "txt", "text":
		return "text"
	default:
		return "text"
	}
}

// guessDocType guesses document type from extension
func guessDocType(extension string) string {
	switch extension {
	case "md", "markdown":
		return "markdown"
	case "html", "htm":
		return "html"
	case "txt":
		return "text"
	default:
		return ""
	}
}
