package cli

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/scttfrdmn/aws-ide/pkg/config"
	"github.com/spf13/cobra"
)

// NewExportConfigCmd creates the export-config command
func NewExportConfigCmd() *cobra.Command {
	var (
		configName string
		output     string
	)

	cmd := &cobra.Command{
		Use:   "export-config",
		Short: "Export RStudio configuration from local machine or instance",
		Long: `Export RStudio configuration including settings, packages, and preferences.

This creates a tar.gz archive that can be imported to new instances.

Examples:
  # Export using default config
  aws-rstudio export-config --output rstudio-config.tar.gz

  # Export using custom config definition
  aws-rstudio export-config --config my-custom --output config.tar.gz`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExportConfig(configName, output)
		},
	}

	cmd.Flags().StringVar(&configName, "config", "rstudio-default", "Export config definition to use")
	cmd.Flags().StringVarP(&output, "output", "o", "rstudio-config.tar.gz", "Output file path")

	return cmd
}

func runExportConfig(configName, output string) error {
	// Load export config
	exportCfg, err := config.LoadExportConfig(configName)
	if err != nil {
		return fmt.Errorf("failed to load export config: %w", err)
	}

	fmt.Printf("Exporting %s configuration...\n", exportCfg.Name)
	fmt.Printf("Description: %s\n\n", exportCfg.Description)

	// Create temporary directory for staging files
	tempDir, err := os.MkdirTemp("", "rstudio-export-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Process each export path
	var exportedPaths []string
	for _, exportPath := range exportCfg.ExportPaths {
		fmt.Printf("Processing: %s\n", exportPath.Description)

		// Run generate command if specified
		if exportPath.GenerateCommand != "" {
			fmt.Printf("  Running: %s\n", exportPath.GenerateCommand)
			cmd := exec.Command("bash", "-c", exportPath.GenerateCommand)
			cmd.Dir = homeDir
			if output, err := cmd.CombinedOutput(); err != nil {
				if !exportPath.Optional {
					return fmt.Errorf("failed to run generate command: %w\nOutput: %s", err, string(output))
				}
				fmt.Printf("  Warning: generate command failed (optional): %v\n", err)
				continue
			}
		}

		// Expand path
		sourcePath := filepath.Join(homeDir, exportPath.Path)

		// Check if path exists
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			if !exportPath.Optional {
				return fmt.Errorf("required path does not exist: %s", sourcePath)
			}
			fmt.Printf("  Skipping (not found, optional)\n")
			continue
		}

		// Copy to temp directory
		destPath := filepath.Join(tempDir, exportPath.Path)
		if err := copyPath(sourcePath, destPath, exportPath.Exclude); err != nil {
			return fmt.Errorf("failed to copy %s: %w", sourcePath, err)
		}

		exportedPaths = append(exportedPaths, exportPath.Path)
		fmt.Printf("  ✓ Exported\n")
	}

	if len(exportedPaths) == 0 {
		return fmt.Errorf("no paths were exported")
	}

	// Create tar.gz archive
	fmt.Printf("\nCreating archive: %s\n", output)
	if err := createTarGz(tempDir, output); err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}

	// Get file size
	fileInfo, _ := os.Stat(output)
	sizeKB := fileInfo.Size() / 1024

	fmt.Printf("\n✓ Export complete!\n")
	fmt.Printf("  Output: %s (%.1f KB)\n", output, float64(sizeKB))
	fmt.Printf("  Exported %d paths\n", len(exportedPaths))
	fmt.Printf("\nTo import this configuration on an instance:\n")
	fmt.Printf("  aws-rstudio import-config --input %s --instance-id i-xxx\n", output)

	return nil
}

// copyPath recursively copies a file or directory, excluding specified patterns
func copyPath(src, dst string, exclude []string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if srcInfo.IsDir() {
		return copyDir(src, dst, exclude)
	}
	return copyFile(src, dst)
}

// copyDir recursively copies a directory
func copyDir(src, dst string, exclude []string) error {
	// Create destination directory
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		// Check if path should be excluded
		skip := false
		for _, pattern := range exclude {
			if matched, _ := filepath.Match(pattern, srcPath); matched {
				skip = true
				break
			}
			// Also check if the base name matches
			if matched, _ := filepath.Match(pattern, entry.Name()); matched {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath, exclude); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy contents
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// Copy permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

// createTarGz creates a tar.gz archive from a directory
func createTarGz(srcDir, destFile string) error {
	// Create output file
	outFile, err := os.Create(destFile)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Create gzip writer
	gzWriter := gzip.NewWriter(outFile)
	defer gzWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Walk through source directory
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		// Set relative path in archive
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		header.Name = relPath

		// Skip root directory
		if relPath == "." {
			return nil
		}

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// Write file contents (if not a directory)
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(tarWriter, file); err != nil {
				return err
			}
		}

		return nil
	})
}
