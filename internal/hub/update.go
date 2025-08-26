package hub

import (
	"beszel"
	"beszel/internal/ghupdate"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// Update updates beszel to the latest version
func Update(_ *cobra.Command, _ []string) {
	config := ghupdate.Config{
		Repo:    "svenvg93/lightspeed", // Update this to your repository
		Current: beszel.Version,
		Filters: []string{"beszel_"},
	}

	ghupdate.PrintUpdateInfo("beszel", beszel.Version, "")

	release, hasUpdate, err := ghupdate.CheckForUpdate(config)
	if err != nil {
		fmt.Printf("Error checking for updates: %v\n", err)
		os.Exit(1)
	}

	if !hasUpdate {
		fmt.Println("You are up to date")
		return
	}

	latestVersion, err := release.Version()
	if err != nil {
		fmt.Printf("Invalid release version: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Latest version: %s\n", latestVersion.String())

	// Find appropriate asset for current platform
	asset := release.FindAsset(config.Filters)
	if asset == nil {
		fmt.Println("No compatible release found for your platform")
		os.Exit(1)
	}

	ghupdate.PrintUpdateProgress(beszel.Version, latestVersion.String())

	// Get current executable path
	binaryPath, err := os.Executable()
	if err != nil {
		fmt.Printf("Error getting binary path: %v\n", err)
		os.Exit(1)
	}

	// Perform the update
	err = ghupdate.UpdateBinary(asset, binaryPath)
	if err != nil {
		fmt.Printf("Please try rerunning with sudo. Error: %v\n", err)
		os.Exit(1)
	}

	// Set ownership to beszel:beszel if possible (similar to original beszel implementation)
	if chownPath, err := exec.LookPath("chown"); err == nil {
		exec.Command(chownPath, "beszel:beszel", binaryPath).Run()
	}

	ghupdate.PrintUpdateSuccess(latestVersion.String(), release.Body)
}
