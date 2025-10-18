package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetermineProjectType(t *testing.T) {
	tests := []struct {
		name         string
		frameworks   []string
		expectedType string
	}{
		{"React only", []string{"React"}, "frontend"},
		{"Vue only", []string{"Vue.js"}, "frontend"},
		{"Express only", []string{"Express"}, "backend"},
		{"NestJS only", []string{"NestJS"}, "backend"},
		{"React and Express", []string{"React", "Express"}, "fullstack"},
		{"Next.js (fullstack by nature)", []string{"Next.js"}, "fullstack"},
		{"No frameworks", []string{}, "web"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := &ProjectAnalysis{
				Frameworks: tt.frameworks,
			}
			determineProjectType(analysis)
			if analysis.DetectedType != tt.expectedType {
				t.Errorf("Expected project type %q, got %q", tt.expectedType, analysis.DetectedType)
			}
		})
	}
}

func TestDetectFrameworks(t *testing.T) {
	tests := []struct {
		name               string
		dependencies       map[string]string
		expectedFrameworks []string
	}{
		{
			name: "React project",
			dependencies: map[string]string{
				"react":     "^18.0.0",
				"react-dom": "^18.0.0",
			},
			expectedFrameworks: []string{"React"},
		},
		{
			name: "Next.js project",
			dependencies: map[string]string{
				"next":  "^13.0.0",
				"react": "^18.0.0",
			},
			expectedFrameworks: []string{"React", "Next.js"},
		},
		{
			name: "Express backend",
			dependencies: map[string]string{
				"express": "^4.18.0",
			},
			expectedFrameworks: []string{"Express"},
		},
		{
			name: "Fullstack with Vue and Express",
			dependencies: map[string]string{
				"vue":     "^3.0.0",
				"express": "^4.18.0",
			},
			expectedFrameworks: []string{"Vue.js", "Express"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkgJSON := PackageJSON{
				Dependencies: tt.dependencies,
			}
			analysis := &ProjectAnalysis{
				Frameworks: []string{},
			}
			detectFrameworks(analysis, pkgJSON)

			if len(analysis.Frameworks) != len(tt.expectedFrameworks) {
				t.Errorf("Expected %d frameworks, got %d", len(tt.expectedFrameworks), len(analysis.Frameworks))
			}

			for _, expected := range tt.expectedFrameworks {
				found := false
				for _, fw := range analysis.Frameworks {
					if fw == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected framework %q not found", expected)
				}
			}
		})
	}
}

func TestAnalyzeProject_WithPackageJSON(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Create package.json
	pkgJSON := PackageJSON{
		Name:    "test-project",
		Version: "1.0.0",
		Dependencies: map[string]string{
			"react":     "^18.0.0",
			"react-dom": "^18.0.0",
			"express":   "^4.18.0",
		},
		DevDependencies: map[string]string{
			"typescript": "^5.0.0",
			"eslint":     "^8.0.0",
		},
		Scripts: map[string]string{
			"dev":   "next dev",
			"build": "next build",
			"test":  "jest",
		},
	}

	pkgJSONBytes, err := json.MarshalIndent(pkgJSON, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal package.json: %v", err)
	}

	err = os.WriteFile(filepath.Join(tmpDir, "package.json"), pkgJSONBytes, 0644)
	if err != nil {
		t.Fatalf("Failed to write package.json: %v", err)
	}

	// Analyze the project
	analysis, err := analyzeProject(tmpDir)
	if err != nil {
		t.Fatalf("analyzeProject failed: %v", err)
	}

	// Verify results
	if !analysis.HasPackageJSON {
		t.Error("Expected HasPackageJSON to be true")
	}

	if !analysis.IsTypeScript {
		t.Error("Expected IsTypeScript to be true")
	}

	if len(analysis.Packages) != 3 {
		t.Errorf("Expected 3 dependencies, got %d", len(analysis.Packages))
	}

	if len(analysis.DevPackages) != 2 {
		t.Errorf("Expected 2 dev dependencies, got %d", len(analysis.DevPackages))
	}

	if len(analysis.Frameworks) == 0 {
		t.Error("Expected frameworks to be detected")
	}

	if analysis.DetectedType != "fullstack" {
		t.Errorf("Expected project type 'fullstack', got %q", analysis.DetectedType)
	}
}

func TestAnalyzeProject_WithLockFiles(t *testing.T) {
	tests := []struct {
		name         string
		lockFileName string
		expected     string
	}{
		{"npm lock file", "package-lock.json", "npm"},
		{"yarn lock file", "yarn.lock", "yarn"},
		{"pnpm lock file", "pnpm-lock.yaml", "pnpm"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create minimal package.json
			pkgJSON := PackageJSON{
				Name:    "test",
				Version: "1.0.0",
			}
			pkgJSONBytes, _ := json.Marshal(pkgJSON)
			_ = os.WriteFile(filepath.Join(tmpDir, "package.json"), pkgJSONBytes, 0644)

			// Create lock file
			_ = os.WriteFile(filepath.Join(tmpDir, tt.lockFileName), []byte{}, 0644)

			// Analyze
			analysis, err := analyzeProject(tmpDir)
			if err != nil {
				t.Fatalf("analyzeProject failed: %v", err)
			}

			if analysis.HasLockFile != tt.expected {
				t.Errorf("Expected lock file type %q, got %q", tt.expected, analysis.HasLockFile)
			}
		})
	}
}

func TestGenerateEnvironmentConfig(t *testing.T) {
	analysis := &ProjectAnalysis{
		DetectedType:     "fullstack",
		Frameworks:       []string{"React", "Express"},
		IsTypeScript:     true,
		HasLockFile:      "npm",
		Packages:         []string{"react", "react-dom", "express"},
		DevPackages:      []string{"typescript"},
		RequiresDatabase: false,
		IsMonorepo:       false,
	}

	config := generateEnvironmentConfig(analysis)

	// Verify basic structure
	if config.Name != "fullstack-env" {
		t.Errorf("Expected name 'fullstack-env', got %q", config.Name)
	}

	if config.InstanceType != "t4g.medium" {
		t.Errorf("Expected instance type 't4g.medium', got %q", config.InstanceType)
	}

	if config.EBSSize != 20 {
		t.Errorf("Expected EBS size 20, got %d", config.EBSSize)
	}

	if config.AMIBase != "ubuntu-22.04-arm64" {
		t.Errorf("Expected AMI base 'ubuntu-22.04-arm64', got %q", config.AMIBase)
	}

	// Verify packages include Node.js
	foundNode := false
	for _, pkg := range config.Packages {
		if pkg == "nodejs" {
			foundNode = true
			break
		}
	}
	if !foundNode {
		t.Error("Expected 'nodejs' in packages")
	}

	// Verify TypeScript extension is included
	foundTSExtension := false
	for _, ext := range config.Extensions {
		if ext == "ms-vscode.vscode-typescript-next" {
			foundTSExtension = true
			break
		}
	}
	if !foundTSExtension {
		t.Error("Expected TypeScript extension in extensions list")
	}
}

func TestGenerateEnvironmentConfig_WithDatabase(t *testing.T) {
	analysis := &ProjectAnalysis{
		DetectedType:     "backend",
		Frameworks:       []string{"Express"},
		Packages:         []string{"express", "pg"},
		RequiresDatabase: true,
	}

	config := generateEnvironmentConfig(analysis)

	// Should have larger instance and storage for database
	if config.InstanceType != "t4g.large" {
		t.Errorf("Expected instance type 't4g.large' for database projects, got %q", config.InstanceType)
	}

	if config.EBSSize < 25 {
		t.Errorf("Expected EBS size >= 25 for database projects, got %d", config.EBSSize)
	}

	// Should include PostgreSQL packages
	foundPostgres := false
	for _, pkg := range config.Packages {
		if pkg == "postgresql" {
			foundPostgres = true
			break
		}
	}
	if !foundPostgres {
		t.Error("Expected 'postgresql' in packages for project with pg dependency")
	}

	// Should include SQL tools extension
	foundSQLTools := false
	for _, ext := range config.Extensions {
		if ext == "mtxr.sqltools" {
			foundSQLTools = true
			break
		}
	}
	if !foundSQLTools {
		t.Error("Expected SQL tools extension for database projects")
	}
}

func TestRunGenerate_Integration(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Create package.json
	pkgJSON := PackageJSON{
		Name:    "test-app",
		Version: "1.0.0",
		Dependencies: map[string]string{
			"react":     "^18.0.0",
			"react-dom": "^18.0.0",
		},
		DevDependencies: map[string]string{
			"typescript": "^5.0.0",
		},
	}

	pkgJSONBytes, err := json.MarshalIndent(pkgJSON, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal package.json: %v", err)
	}

	err = os.WriteFile(filepath.Join(tmpDir, "package.json"), pkgJSONBytes, 0644)
	if err != nil {
		t.Fatalf("Failed to write package.json: %v", err)
	}

	// Create output file path
	outputFile := filepath.Join(tmpDir, "test-env.yml")

	// Run generate
	err = runGenerate(tmpDir, outputFile)
	if err != nil {
		t.Fatalf("runGenerate failed: %v", err)
	}

	// Verify output file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("Output file was not created")
	}

	// Read and verify content
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	contentStr := string(content)

	// Check for expected content
	if !strings.Contains(contentStr, "name:") {
		t.Error("Output should contain environment name")
	}

	if !strings.Contains(contentStr, "instance_type:") {
		t.Error("Output should contain instance type")
	}

	if !strings.Contains(contentStr, "nodejs") {
		t.Error("Output should contain nodejs package")
	}
}

func TestRunGenerate_NoPackageJSON(t *testing.T) {
	// Create temp directory without package.json
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "test-env.yml")

	// Should still work, just with default config
	err := runGenerate(tmpDir, outputFile)
	if err != nil {
		t.Fatalf("runGenerate should handle missing package.json gracefully, got error: %v", err)
	}

	// Verify output file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("Output file should be created even without package.json")
	}
}
