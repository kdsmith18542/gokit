package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/kdsmith18542/gokit/upload/storage"
	"github.com/spf13/cobra"
)

var (
	azureAccountName string
	azureAccountKey  string
	azureContainer   string
	azureBaseURL     string
)

func I18nCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "i18n",
		Short: "Manage i18n message files",
	}

	var localesDir string
	cmd.PersistentFlags().StringVar(&localesDir, "dir", "./locales", "Directory containing locale files")

	cmd.AddCommand(&cobra.Command{
		Use:   "find-missing",
		Short: "Find missing translation keys",
		Run: func(cmd *cobra.Command, args []string) {
			if err := FindMissingKeys(localesDir); err != nil {
				fmt.Printf("Error: %v\n", err)
				// Don't call os.Exit in tests
				return
			}
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "lint",
		Short: "Lint i18n files for errors",
		Run: func(cmd *cobra.Command, args []string) {
			if err := LintLocaleFiles(localesDir); err != nil {
				fmt.Printf("Error: %v\n", err)
				// Don't call os.Exit in tests
				return
			}
		},
	})
	return cmd
}

func FindMissingKeys(localesDir string) error {
	entries, err := os.ReadDir(localesDir)
	if err != nil {
		return fmt.Errorf("failed to read locales directory: %v", err)
	}
	var localeFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".toml") {
			localeFiles = append(localeFiles, entry.Name())
		}
	}
	if len(localeFiles) == 0 {
		return fmt.Errorf("no TOML locale files found in %s", localesDir)
	}
	allKeys := make(map[string]bool)
	localeKeys := make(map[string]map[string]bool)
	for _, file := range localeFiles {
		locale := strings.TrimSuffix(file, ".toml")
		filePath := filepath.Join(localesDir, file)
		keys, err := LoadLocaleKeys(filePath)
		if err != nil {
			fmt.Printf("Warning: failed to load %s: %v\n", file, err)
			continue
		}
		localeKeys[locale] = keys
		for key := range keys {
			allKeys[key] = true
		}
	}
	fmt.Printf("Analyzing %d locale files...\n", len(localeFiles))
	fmt.Println()
	hasMissing := false
	for _, locale := range GetSortedKeys(localeKeys) {
		keys := localeKeys[locale]
		var missing []string
		for key := range allKeys {
			if !keys[key] {
				missing = append(missing, key)
			}
		}
		if len(missing) > 0 {
			hasMissing = true
			fmt.Printf("Locale '%s' is missing %d keys:\n", locale, len(missing))
			for _, key := range missing {
				fmt.Printf("  - %s\n", key)
			}
			fmt.Println()
		} else {
			fmt.Printf("Locale '%s': ✓ Complete\n", locale)
		}
	}
	if !hasMissing {
		fmt.Println("✓ All locales are complete!")
	}
	return nil
}

func LintLocaleFiles(localesDir string) error {
	entries, err := os.ReadDir(localesDir)
	if err != nil {
		return fmt.Errorf("failed to read locales directory: %v", err)
	}
	var localeFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".toml") {
			localeFiles = append(localeFiles, entry.Name())
		}
	}
	if len(localeFiles) == 0 {
		return fmt.Errorf("no TOML locale files found in %s", localesDir)
	}
	fmt.Printf("Linting %d locale files...\n", len(localeFiles))
	fmt.Println()
	hasErrors := false
	for _, file := range localeFiles {
		filePath := filepath.Join(localesDir, file)
		fmt.Printf("Checking %s...\n", file)
		if err := ValidateTOMLSyntax(filePath); err != nil {
			hasErrors = true
			fmt.Printf("  ❌ TOML syntax error: %v\n", err)
		} else {
			fmt.Printf("  ✓ TOML syntax valid\n")
		}
		if issues := CheckCommonIssues(filePath); len(issues) > 0 {
			hasErrors = true
			for _, issue := range issues {
				fmt.Printf("  ⚠️  %s\n", issue)
			}
		} else {
			fmt.Printf("  ✓ No common issues found\n")
		}
		fmt.Println()
	}
	if !hasErrors {
		fmt.Println("✓ All locale files passed linting!")
	} else {
		return fmt.Errorf("linting found issues")
	}
	return nil
}

func LoadLocaleKeys(filePath string) (map[string]bool, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var messages map[string]interface{}
	if err := toml.Unmarshal(data, &messages); err != nil {
		return nil, err
	}
	keys := make(map[string]bool)
	CollectKeys(messages, "", keys)
	return keys, nil
}

func CollectKeys(messages map[string]interface{}, prefix string, keys map[string]bool) {
	for key, value := range messages {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}
		if subMap, ok := value.(map[string]interface{}); ok {
			CollectKeys(subMap, fullKey, keys)
		} else {
			keys[fullKey] = true
		}
	}
}

func ValidateTOMLSyntax(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	var messages map[string]interface{}
	return toml.Unmarshal(data, &messages)
}

func CheckCommonIssues(filePath string) []string {
	var issues []string
	data, err := os.ReadFile(filePath)
	if err != nil {
		return []string{fmt.Sprintf("Failed to read file: %v", err)}
	}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		lineNum := i + 1
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasSuffix(line, " ") || strings.HasSuffix(line, "\t") {
			issues = append(issues, fmt.Sprintf("Line %d: trailing whitespace", lineNum))
		}
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				value := strings.TrimSpace(parts[1])
				if value != "" && !strings.HasPrefix(value, `"`) && !strings.HasPrefix(value, `'`) {
					issues = append(issues, fmt.Sprintf("Line %d: value should be quoted", lineNum))
				}
			}
		}
	}
	return issues
}

func GetSortedKeys(m map[string]map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func UploadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Manage file upload backends",
	}
	var backend, bucket, region, accessKey, secretKey, endpoint string
	var forcePathStyle bool
	var credentialsFile, projectID, gcsBaseURL string
	cmd.PersistentFlags().StringVar(&backend, "backend", "s3", "Storage backend (s3, gcs, azure)")
	cmd.PersistentFlags().StringVar(&bucket, "bucket", "", "Bucket name (required)")
	cmd.PersistentFlags().StringVar(&region, "region", "", "Region (required for S3)")
	cmd.PersistentFlags().StringVar(&accessKey, "access-key", "", "Access key ID (optional, S3)")
	cmd.PersistentFlags().StringVar(&secretKey, "secret-key", "", "Secret access key (optional, S3)")
	cmd.PersistentFlags().StringVar(&endpoint, "endpoint", "", "Custom endpoint (optional, S3)")
	cmd.PersistentFlags().BoolVar(&forcePathStyle, "force-path-style", false, "Use path-style addressing (S3-compatible)")
	cmd.PersistentFlags().StringVar(&credentialsFile, "credentials-file", "", "Path to GCS service account JSON (optional, GCS)")
	cmd.PersistentFlags().StringVar(&projectID, "project-id", "", "GCP project ID (optional, GCS)")
	cmd.PersistentFlags().StringVar(&gcsBaseURL, "gcs-base-url", "", "Custom base URL for GCS public access (optional)")
	cmd.PersistentFlags().StringVar(&azureAccountName, "azure-account", os.Getenv("AZURE_STORAGE_ACCOUNT"), "Azure storage account name")
	cmd.PersistentFlags().StringVar(&azureAccountKey, "azure-key", os.Getenv("AZURE_STORAGE_KEY"), "Azure storage account key")
	cmd.PersistentFlags().StringVar(&azureContainer, "azure-container", os.Getenv("AZURE_STORAGE_CONTAINER"), "Azure blob container name")
	cmd.PersistentFlags().StringVar(&azureBaseURL, "azure-base-url", "", "Custom Azure blob base URL (optional)")
	var expiration string
	cmd.PersistentFlags().StringVar(&expiration, "expiration", "1h", "Expiration time for pre-signed URLs (e.g., 1h, 30m, 24h)")

	cmd.AddCommand(&cobra.Command{
		Use:   "verify-credentials",
		Short: "Verify credentials for a storage backend",
		Run: func(cmd *cobra.Command, args []string) {
			var stor storage.Storage
			var err error
			switch backend {
			case "s3":
				if bucket == "" || region == "" {
					fmt.Println("--bucket and --region are required for S3 backend.")
					return
				}
				s3cfg := storage.S3Config{
					Bucket:          bucket,
					Region:          region,
					AccessKeyID:     accessKey,
					SecretAccessKey: secretKey,
					Endpoint:        endpoint,
					ForcePathStyle:  forcePathStyle,
				}
				stor, err = storage.NewS3(s3cfg)
			case "gcs":
				if bucket == "" {
					fmt.Println("--bucket is required for GCS backend.")
					return
				}
				gcscfg := storage.GCSConfig{
					Bucket:          bucket,
					ProjectID:       projectID,
					CredentialsFile: credentialsFile,
					BaseURL:         gcsBaseURL,
				}
				stor, err = storage.NewGCS(gcscfg)
			case "azure":
				if azureContainer == "" {
					fmt.Println("--azure-container is required for Azure backend.")
					return
				}
				stor, err = storage.NewAzureBlob(storage.AzureConfig{
					AccountName: azureAccountName,
					AccountKey:  azureAccountKey,
					Container:   azureContainer,
					BaseURL:     azureBaseURL,
				})
			default:
				fmt.Println("Supported backends: s3, gcs, azure")
				return
			}
			if err != nil {
				fmt.Printf("Failed to create storage: %v\n", err)
				return
			}
			info, err := stor.GetBucketInfo()
			if err != nil {
				fmt.Printf("Failed to access bucket: %v\n", err)
				return
			}
			fmt.Printf("%s credentials and bucket access verified.\n", backend)
			fmt.Printf("Bucket info: %v\n", info)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "list-files",
		Short: "List files in a storage backend",
		Run: func(cmd *cobra.Command, args []string) {
			var stor storage.Storage
			var err error
			switch backend {
			case "s3":
				if bucket == "" || region == "" {
					fmt.Println("--bucket and --region are required for S3 backend.")
					return
				}
				s3cfg := storage.S3Config{
					Bucket:          bucket,
					Region:          region,
					AccessKeyID:     accessKey,
					SecretAccessKey: secretKey,
					Endpoint:        endpoint,
					ForcePathStyle:  forcePathStyle,
				}
				stor, err = storage.NewS3(s3cfg)
			case "gcs":
				if bucket == "" {
					fmt.Println("--bucket is required for GCS backend.")
					return
				}
				gcscfg := storage.GCSConfig{
					Bucket:          bucket,
					ProjectID:       projectID,
					CredentialsFile: credentialsFile,
					BaseURL:         gcsBaseURL,
				}
				stor, err = storage.NewGCS(gcscfg)
			case "azure":
				if azureContainer == "" {
					fmt.Println("--azure-container is required for Azure backend.")
					return
				}
				stor, err = storage.NewAzureBlob(storage.AzureConfig{
					AccountName: azureAccountName,
					AccountKey:  azureAccountKey,
					Container:   azureContainer,
					BaseURL:     azureBaseURL,
				})
			default:
				fmt.Println("Supported backends: s3, gcs, azure")
				return
			}
			if err != nil {
				fmt.Printf("Failed to create storage: %v\n", err)
				return
			}
			defer stor.Close()

			files, err := stor.ListFiles()
			if err != nil {
				fmt.Printf("Failed to list files: %v\n", err)
				return
			}

			if len(files) == 0 {
				fmt.Println("No files found.")
				return
			}

			fmt.Printf("Found %d files:\n", len(files))
			for _, file := range files {
				size, err := stor.GetSize(file)
				if err != nil {
					fmt.Printf("  %s (size unknown)\n", file)
				} else {
					fmt.Printf("  %s (%s)\n", file, FormatBytes(size))
				}
			}
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "upload-file [file]",
		Short: "Upload a file to storage backend",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			filePath := args[0]

			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				fmt.Printf("File not found: %s\n", filePath)
				return
			}

			var stor storage.Storage
			var err error
			switch backend {
			case "s3":
				if bucket == "" || region == "" {
					fmt.Println("--bucket and --region are required for S3 backend.")
					return
				}
				s3cfg := storage.S3Config{
					Bucket:          bucket,
					Region:          region,
					AccessKeyID:     accessKey,
					SecretAccessKey: secretKey,
					Endpoint:        endpoint,
					ForcePathStyle:  forcePathStyle,
				}
				stor, err = storage.NewS3(s3cfg)
			case "gcs":
				if bucket == "" {
					fmt.Println("--bucket is required for GCS backend.")
					return
				}
				gcscfg := storage.GCSConfig{
					Bucket:          bucket,
					ProjectID:       projectID,
					CredentialsFile: credentialsFile,
					BaseURL:         gcsBaseURL,
				}
				stor, err = storage.NewGCS(gcscfg)
			case "azure":
				if azureContainer == "" {
					fmt.Println("--azure-container is required for Azure backend.")
					return
				}
				stor, err = storage.NewAzureBlob(storage.AzureConfig{
					AccountName: azureAccountName,
					AccountKey:  azureAccountKey,
					Container:   azureContainer,
					BaseURL:     azureBaseURL,
				})
			default:
				fmt.Println("Supported backends: s3, gcs, azure")
				return
			}
			if err != nil {
				fmt.Printf("Failed to create storage: %v\n", err)
				return
			}
			defer stor.Close()

			file, err := os.Open(filePath)
			if err != nil {
				fmt.Printf("Failed to open file: %v\n", err)
				return
			}
			defer file.Close()

			fileName := filepath.Base(filePath)
			fmt.Printf("Uploading %s...\n", fileName)

			key, err := stor.Store(fileName, file)
			if err != nil {
				fmt.Printf("Failed to upload file: %v\n", err)
				return
			}

			fmt.Printf("✓ File uploaded successfully!\n")
			fmt.Printf("Key: %s\n", key)
			fmt.Printf("URL: %s\n", stor.GetURL(key))
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "generate-url [file]",
		Short: "Generate a pre-signed URL for file access",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fileName := args[0]

			expiration := 1 * time.Hour
			if expirationStr := cmd.Flag("expiration").Value.String(); expirationStr != "" {
				var err error
				expiration, err = time.ParseDuration(expirationStr)
				if err != nil {
					fmt.Printf("Invalid expiration format: %v\n", err)
					return
				}
			}

			var stor storage.Storage
			var err error
			switch backend {
			case "s3":
				if bucket == "" || region == "" {
					fmt.Println("--bucket and --region are required for S3 backend.")
					return
				}
				s3cfg := storage.S3Config{
					Bucket:          bucket,
					Region:          region,
					AccessKeyID:     accessKey,
					SecretAccessKey: secretKey,
					Endpoint:        endpoint,
					ForcePathStyle:  forcePathStyle,
				}
				stor, err = storage.NewS3(s3cfg)
			case "gcs":
				if bucket == "" {
					fmt.Println("--bucket is required for GCS backend.")
					return
				}
				gcscfg := storage.GCSConfig{
					Bucket:          bucket,
					ProjectID:       projectID,
					CredentialsFile: credentialsFile,
					BaseURL:         gcsBaseURL,
				}
				stor, err = storage.NewGCS(gcscfg)
			case "azure":
				if azureContainer == "" {
					fmt.Println("--azure-container is required for Azure backend.")
					return
				}
				stor, err = storage.NewAzureBlob(storage.AzureConfig{
					AccountName: azureAccountName,
					AccountKey:  azureAccountKey,
					Container:   azureContainer,
					BaseURL:     azureBaseURL,
				})
			default:
				fmt.Println("Supported backends: s3, gcs, azure")
				return
			}
			if err != nil {
				fmt.Printf("Failed to create storage: %v\n", err)
				return
			}
			defer stor.Close()

			if !stor.Exists(fileName) {
				fmt.Printf("File not found: %s\n", fileName)
				return
			}

			signedURL, err := stor.GetSignedURL(fileName, expiration)
			if err != nil {
				fmt.Printf("Failed to generate signed URL: %v\n", err)
				return
			}

			fmt.Printf("Pre-signed URL for %s (expires in %s):\n", fileName, expiration)
			fmt.Println(signedURL)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "delete-file [file]",
		Short: "Delete a file from storage backend",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fileName := args[0]

			var stor storage.Storage
			var err error
			switch backend {
			case "s3":
				if bucket == "" || region == "" {
					fmt.Println("--bucket and --region are required for S3 backend.")
					return
				}
				s3cfg := storage.S3Config{
					Bucket:          bucket,
					Region:          region,
					AccessKeyID:     accessKey,
					SecretAccessKey: secretKey,
					Endpoint:        endpoint,
					ForcePathStyle:  forcePathStyle,
				}
				stor, err = storage.NewS3(s3cfg)
			case "gcs":
				if bucket == "" {
					fmt.Println("--bucket is required for GCS backend.")
					return
				}
				gcscfg := storage.GCSConfig{
					Bucket:          bucket,
					ProjectID:       projectID,
					CredentialsFile: credentialsFile,
					BaseURL:         gcsBaseURL,
				}
				stor, err = storage.NewGCS(gcscfg)
			case "azure":
				if azureContainer == "" {
					fmt.Println("--azure-container is required for Azure backend.")
					return
				}
				stor, err = storage.NewAzureBlob(storage.AzureConfig{
					AccountName: azureAccountName,
					AccountKey:  azureAccountKey,
					Container:   azureContainer,
					BaseURL:     azureBaseURL,
				})
			default:
				fmt.Println("Supported backends: s3, gcs, azure")
				return
			}
			if err != nil {
				fmt.Printf("Failed to create storage: %v\n", err)
				return
			}
			defer stor.Close()

			if !stor.Exists(fileName) {
				fmt.Printf("File not found: %s\n", fileName)
				return
			}

			if err := stor.Delete(fileName); err != nil {
				fmt.Printf("Failed to delete file: %v\n", err)
				return
			}

			fmt.Printf("✓ File %s deleted successfully!\n", fileName)
		},
	})

	return cmd
}

func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
