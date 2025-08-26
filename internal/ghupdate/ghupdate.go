package ghupdate

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/fatih/color"
)

type Release struct {
	TagName    string  `json:"tag_name"`
	Name       string  `json:"name"`
	Body       string  `json:"body"`
	Assets     []Asset `json:"assets"`
	PreRelease bool    `json:"prerelease"`
	Draft      bool    `json:"draft"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type Config struct {
	Repo    string
	Current string
	Filters []string
}

func (r *Release) Version() (semver.Version, error) {
	tagName := strings.TrimPrefix(r.TagName, "v")
	return semver.Parse(tagName)
}

func (r *Release) FindAsset(filters []string) *Asset {
	osName := runtime.GOOS
	archName := runtime.GOARCH
	
	// Convert Go arch names to common release naming conventions
	if archName == "amd64" {
		archName = "x86_64"
	} else if archName == "arm64" {
		archName = "aarch64"
	}

	for _, asset := range r.Assets {
		name := strings.ToLower(asset.Name)
		
		// Skip if it doesn't match any filter
		matchesFilter := len(filters) == 0
		for _, filter := range filters {
			if strings.Contains(name, strings.ToLower(filter)) {
				matchesFilter = true
				break
			}
		}
		if !matchesFilter {
			continue
		}
		
		// Check if asset matches current OS and architecture
		if strings.Contains(name, osName) && strings.Contains(name, archName) {
			return &asset
		}
	}
	
	return nil
}

func CheckForUpdate(config Config) (*Release, bool, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", config.Repo)
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, false, fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}
	
	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, false, fmt.Errorf("failed to decode release info: %w", err)
	}
	
	// Skip drafts and pre-releases
	if release.Draft || release.PreRelease {
		return nil, false, nil
	}
	
	currentVersion, err := semver.Parse(config.Current)
	if err != nil {
		return nil, false, fmt.Errorf("invalid current version: %w", err)
	}
	
	latestVersion, err := release.Version()
	if err != nil {
		return nil, false, fmt.Errorf("invalid release version: %w", err)
	}
	
	if latestVersion.GT(currentVersion) {
		return &release, true, nil
	}
	
	return &release, false, nil
}

func UpdateBinary(asset *Asset, targetPath string) error {
	// Create temporary file for download
	tempFile, err := os.CreateTemp("", "update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()
	
	// Download the asset
	if err := downloadFile(asset.BrowserDownloadURL, tempFile.Name()); err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	
	// Extract binary from archive if needed
	binaryPath := tempFile.Name()
	if strings.HasSuffix(asset.Name, ".tar.gz") {
		extractedPath, err := extractTarGz(tempFile.Name(), filepath.Base(targetPath))
		if err != nil {
			return fmt.Errorf("failed to extract tar.gz: %w", err)
		}
		defer os.Remove(extractedPath)
		binaryPath = extractedPath
	} else if strings.HasSuffix(asset.Name, ".zip") {
		extractedPath, err := extractZip(tempFile.Name(), filepath.Base(targetPath))
		if err != nil {
			return fmt.Errorf("failed to extract zip: %w", err)
		}
		defer os.Remove(extractedPath)
		binaryPath = extractedPath
	}
	
	// Make executable
	if err := os.Chmod(binaryPath, 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}
	
	// Replace the current binary
	if err := os.Rename(binaryPath, targetPath); err != nil {
		return fmt.Errorf("failed to replace binary: %w", err)
	}
	
	// Make sure target is executable
	if err := os.Chmod(targetPath, 0755); err != nil {
		return fmt.Errorf("failed to make target executable: %w", err)
	}
	
	return nil
}

func downloadFile(url, filepath string) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}
	
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()
	
	_, err = io.Copy(out, resp.Body)
	return err
}

func extractTarGz(archivePath, binaryName string) (string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gzr.Close()
	
	tr := tar.NewReader(gzr)
	
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		
		if header.Typeflag == tar.TypeReg && (header.Name == binaryName || filepath.Base(header.Name) == binaryName) {
			tempFile, err := os.CreateTemp("", "extracted-*")
			if err != nil {
				return "", err
			}
			defer tempFile.Close()
			
			_, err = io.Copy(tempFile, tr)
			if err != nil {
				os.Remove(tempFile.Name())
				return "", err
			}
			
			return tempFile.Name(), nil
		}
	}
	
	return "", fmt.Errorf("binary %s not found in archive", binaryName)
}

func extractZip(archivePath, binaryName string) (string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer r.Close()
	
	for _, f := range r.File {
		if f.Name == binaryName || filepath.Base(f.Name) == binaryName {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()
			
			tempFile, err := os.CreateTemp("", "extracted-*")
			if err != nil {
				return "", err
			}
			defer tempFile.Close()
			
			_, err = io.Copy(tempFile, rc)
			if err != nil {
				os.Remove(tempFile.Name())
				return "", err
			}
			
			return tempFile.Name(), nil
		}
	}
	
	return "", fmt.Errorf("binary %s not found in archive", binaryName)
}

// PrintUpdateInfo prints colored update information
func PrintUpdateInfo(appName, currentVersion, latestVersion string) {
	fmt.Printf("%s %s\n", color.CyanString(appName), color.YellowString(currentVersion))
	fmt.Println(color.BlueString("Checking for updates..."))
	fmt.Printf("Latest version: %s\n", color.GreenString(latestVersion))
}

func PrintUpdateSuccess(latestVersion, releaseNotes string) {
	fmt.Printf("%s %s\n\n%s\n", 
		color.GreenString("Successfully updated to"), 
		color.YellowString(latestVersion),
		strings.TrimSpace(releaseNotes))
}

func PrintUpdateProgress(currentVersion, latestVersion string) {
	fmt.Printf("%s %s %s %s%s\n", 
		color.BlueString("Updating from"), 
		color.YellowString(currentVersion), 
		color.BlueString("to"), 
		color.GreenString(latestVersion),
		color.BlueString("..."))
}