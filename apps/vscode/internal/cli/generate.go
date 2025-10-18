package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// NewGenerateCmd creates the generate command for analyzing Node.js projects
func NewGenerateCmd() *cobra.Command {
	var outputFile string
	var projectDir string

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate environment config from local Node.js project",
		Long: `Analyze your local Node.js/JavaScript/TypeScript project and generate an optimized
environment configuration file.

This command will:
• Scan package.json for dependencies
• Detect frameworks (React, Vue, Next.js, Express, etc.)
• Analyze package-lock.json, yarn.lock, or pnpm-lock.yaml
• Suggest appropriate instance types and storage
• Generate a ready-to-use environment YAML file

The generated config can be used with:
  aws-vscode launch --env-file generated-env.yml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerate(projectDir, outputFile)
		},
	}

	cmd.Flags().StringVarP(&outputFile, "output", "o", "vscode-env.yml", "Output file for generated environment config")
	cmd.Flags().StringVarP(&projectDir, "dir", "d", ".", "Directory containing Node.js project to analyze")

	return cmd
}

// PackageJSON represents the structure of package.json
type PackageJSON struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	Scripts         map[string]string `json:"scripts"`
}

// ProjectAnalysis contains detected project characteristics
type ProjectAnalysis struct {
	HasPackageJSON   bool
	HasLockFile      string // "npm", "yarn", "pnpm", or ""
	Packages         []string
	DevPackages      []string
	Scripts          map[string]string
	DetectedType     string // "frontend", "backend", "fullstack", "web"
	Frameworks       []string
	RequiresDatabase bool
	IsTypeScript     bool
	IsMonorepo       bool
}

// EnvironmentConfig represents the YAML structure for environments
type EnvironmentConfig struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	AMIBase     string            `yaml:"ami_base"`
	InstanceType string           `yaml:"instance_type"`
	EBSSize     int               `yaml:"ebs_size"`
	Packages    []string          `yaml:"packages"`
	Extensions  []string          `yaml:"extensions"`
	Settings    map[string]string `yaml:"settings,omitempty"`
}

func runGenerate(projectDir string, outputFile string) error {
	fmt.Println("🔍 Analyzing Node.js project...")

	// Convert to absolute path
	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return fmt.Errorf("failed to resolve project directory: %w", err)
	}

	// Analyze the project
	analysis, err := analyzeProject(absDir)
	if err != nil {
		return fmt.Errorf("failed to analyze project: %w", err)
	}

	// Display analysis results
	displayAnalysis(analysis)

	// Generate environment config
	envConfig := generateEnvironmentConfig(analysis)

	// Write to file
	if err := writeEnvironmentConfig(envConfig, outputFile); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("\n✓ Generated environment config: %s\n", outputFile)
	fmt.Printf("\nTo launch an instance with this configuration:\n")
	fmt.Printf("  aws-vscode launch --env-file %s\n", outputFile)

	return nil
}

func analyzeProject(projectDir string) (*ProjectAnalysis, error) {
	analysis := &ProjectAnalysis{
		Scripts:      make(map[string]string),
		Frameworks:   []string{},
		Packages:     []string{},
		DevPackages:  []string{},
	}

	// Check for package.json
	packageJSONPath := filepath.Join(projectDir, "package.json")
	if _, err := os.Stat(packageJSONPath); err == nil {
		analysis.HasPackageJSON = true

		// Read and parse package.json
		data, err := os.ReadFile(packageJSONPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read package.json: %w", err)
		}

		var pkgJSON PackageJSON
		if err := json.Unmarshal(data, &pkgJSON); err != nil {
			return nil, fmt.Errorf("failed to parse package.json: %w", err)
		}

		// Extract packages
		for pkg := range pkgJSON.Dependencies {
			analysis.Packages = append(analysis.Packages, pkg)
		}
		for pkg := range pkgJSON.DevDependencies {
			analysis.DevPackages = append(analysis.DevPackages, pkg)
		}

		analysis.Scripts = pkgJSON.Scripts

		// Detect TypeScript
		if _, hasTS := pkgJSON.Dependencies["typescript"]; hasTS {
			analysis.IsTypeScript = true
		}
		if _, hasTS := pkgJSON.DevDependencies["typescript"]; hasTS {
			analysis.IsTypeScript = true
		}

		// Detect frameworks and project type
		detectFrameworks(analysis, pkgJSON)
	}

	// Check for lock files
	if _, err := os.Stat(filepath.Join(projectDir, "package-lock.json")); err == nil {
		analysis.HasLockFile = "npm"
	} else if _, err := os.Stat(filepath.Join(projectDir, "yarn.lock")); err == nil {
		analysis.HasLockFile = "yarn"
	} else if _, err := os.Stat(filepath.Join(projectDir, "pnpm-lock.yaml")); err == nil {
		analysis.HasLockFile = "pnpm"
	}

	// Check for monorepo indicators
	if _, err := os.Stat(filepath.Join(projectDir, "lerna.json")); err == nil {
		analysis.IsMonorepo = true
	}
	if _, err := os.Stat(filepath.Join(projectDir, "nx.json")); err == nil {
		analysis.IsMonorepo = true
	}
	if _, err := os.Stat(filepath.Join(projectDir, "pnpm-workspace.yaml")); err == nil {
		analysis.IsMonorepo = true
	}

	// Determine project type based on detected packages
	determineProjectType(analysis)

	return analysis, nil
}

func detectFrameworks(analysis *ProjectAnalysis, pkgJSON PackageJSON) {
	allPackages := make(map[string]bool)
	for pkg := range pkgJSON.Dependencies {
		allPackages[pkg] = true
	}
	for pkg := range pkgJSON.DevDependencies {
		allPackages[pkg] = true
	}

	// Frontend frameworks
	if allPackages["react"] || allPackages["react-dom"] {
		analysis.Frameworks = append(analysis.Frameworks, "React")
	}
	if allPackages["next"] {
		analysis.Frameworks = append(analysis.Frameworks, "Next.js")
	}
	if allPackages["vue"] {
		analysis.Frameworks = append(analysis.Frameworks, "Vue.js")
	}
	if allPackages["@angular/core"] {
		analysis.Frameworks = append(analysis.Frameworks, "Angular")
	}
	if allPackages["svelte"] {
		analysis.Frameworks = append(analysis.Frameworks, "Svelte")
	}

	// Backend frameworks
	if allPackages["express"] {
		analysis.Frameworks = append(analysis.Frameworks, "Express")
	}
	if allPackages["@nestjs/core"] {
		analysis.Frameworks = append(analysis.Frameworks, "NestJS")
	}
	if allPackages["fastify"] {
		analysis.Frameworks = append(analysis.Frameworks, "Fastify")
	}
	if allPackages["koa"] {
		analysis.Frameworks = append(analysis.Frameworks, "Koa")
	}

	// Databases
	if allPackages["mongoose"] || allPackages["mongodb"] {
		analysis.RequiresDatabase = true
	}
	if allPackages["pg"] || allPackages["mysql2"] || allPackages["mysql"] {
		analysis.RequiresDatabase = true
	}
	if allPackages["prisma"] || allPackages["@prisma/client"] {
		analysis.RequiresDatabase = true
	}
	if allPackages["typeorm"] || allPackages["sequelize"] {
		analysis.RequiresDatabase = true
	}
}

func determineProjectType(analysis *ProjectAnalysis) {
	hasFrontend := false
	hasBackend := false

	frontendFrameworks := []string{"React", "Vue.js", "Angular", "Svelte", "Next.js"}
	backendFrameworks := []string{"Express", "NestJS", "Fastify", "Koa"}

	for _, fw := range analysis.Frameworks {
		for _, frontend := range frontendFrameworks {
			if fw == frontend {
				hasFrontend = true
			}
		}
		for _, backend := range backendFrameworks {
			if fw == backend {
				hasBackend = true
			}
		}
	}

	// Next.js is fullstack by nature
	for _, fw := range analysis.Frameworks {
		if fw == "Next.js" {
			analysis.DetectedType = "fullstack"
			return
		}
	}

	if hasFrontend && hasBackend {
		analysis.DetectedType = "fullstack"
	} else if hasFrontend {
		analysis.DetectedType = "frontend"
	} else if hasBackend {
		analysis.DetectedType = "backend"
	} else {
		analysis.DetectedType = "web"
	}
}

func displayAnalysis(analysis *ProjectAnalysis) {
	fmt.Println("\n📊 Project Analysis:")
	fmt.Printf("  Project Type: %s\n", analysis.DetectedType)

	if len(analysis.Frameworks) > 0 {
		fmt.Printf("  Frameworks: %s\n", strings.Join(analysis.Frameworks, ", "))
	}

	if analysis.IsTypeScript {
		fmt.Println("  Language: TypeScript")
	} else {
		fmt.Println("  Language: JavaScript")
	}

	if analysis.HasLockFile != "" {
		fmt.Printf("  Package Manager: %s\n", analysis.HasLockFile)
	}

	if analysis.IsMonorepo {
		fmt.Println("  Structure: Monorepo")
	}

	if analysis.RequiresDatabase {
		fmt.Println("  Database: Detected")
	}

	fmt.Printf("  Dependencies: %d production, %d dev\n", len(analysis.Packages), len(analysis.DevPackages))
}

func generateEnvironmentConfig(analysis *ProjectAnalysis) *EnvironmentConfig {
	config := &EnvironmentConfig{
		Name:         fmt.Sprintf("%s-env", analysis.DetectedType),
		Description:  fmt.Sprintf("Generated %s environment", analysis.DetectedType),
		AMIBase:      "ubuntu-22.04-arm64",
		InstanceType: "t4g.medium",
		EBSSize:      20,
		Packages:     []string{},
		Extensions:   []string{},
		Settings:     make(map[string]string),
	}

	// Adjust instance type based on project characteristics
	if analysis.IsMonorepo || len(analysis.Packages) > 100 {
		config.InstanceType = "t4g.large"
		config.EBSSize = 30
	}

	if analysis.RequiresDatabase {
		config.InstanceType = "t4g.large"
		config.EBSSize = 25
	}

	// Add system packages
	config.Packages = append(config.Packages,
		"curl",
		"wget",
		"git",
		"build-essential",
	)

	// Add Node.js (using NodeSource repository for latest LTS)
	config.Packages = append(config.Packages,
		"nodejs",
		"npm",
	)

	// Add package manager if detected
	switch analysis.HasLockFile {
	case "yarn":
		config.Packages = append(config.Packages, "yarn")
	case "pnpm":
		// pnpm is installed via npm, not apt
	}

	// Add database packages if needed
	if analysis.RequiresDatabase {
		// Check for specific database packages
		hasPostgres := contains(analysis.Packages, "pg") || contains(analysis.Packages, "postgres")
		hasMongo := contains(analysis.Packages, "mongodb") || contains(analysis.Packages, "mongoose")
		hasMysql := contains(analysis.Packages, "mysql") || contains(analysis.Packages, "mysql2")

		if hasPostgres {
			config.Packages = append(config.Packages, "postgresql", "postgresql-contrib")
		}
		if hasMongo {
			config.Packages = append(config.Packages, "mongodb")
		}
		if hasMysql {
			config.Packages = append(config.Packages, "mysql-server")
		}
	}

	// Add VSCode extensions based on detected technologies
	config.Extensions = append(config.Extensions, "dbaeumer.vscode-eslint")
	config.Extensions = append(config.Extensions, "esbenp.prettier-vscode")

	if analysis.IsTypeScript {
		config.Extensions = append(config.Extensions, "ms-vscode.vscode-typescript-next")
	}

	for _, fw := range analysis.Frameworks {
		switch fw {
		case "React":
			config.Extensions = append(config.Extensions, "dsznajder.es7-react-js-snippets")
		case "Vue.js":
			config.Extensions = append(config.Extensions, "vue.volar")
		case "Angular":
			config.Extensions = append(config.Extensions, "angular.ng-template")
		case "Svelte":
			config.Extensions = append(config.Extensions, "svelte.svelte-vscode")
		}
	}

	if analysis.RequiresDatabase {
		config.Extensions = append(config.Extensions, "mtxr.sqltools")
	}

	// Add Docker extension if docker-related scripts are present
	for scriptName := range analysis.Scripts {
		if regexp.MustCompile(`docker`).MatchString(scriptName) {
			config.Extensions = append(config.Extensions, "ms-azuretools.vscode-docker")
			config.Packages = append(config.Packages, "docker.io", "docker-compose")
			break
		}
	}

	return config
}

func writeEnvironmentConfig(config *EnvironmentConfig, outputFile string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	// Add header comment
	header := fmt.Sprintf("# Generated VSCode environment configuration\n# Created by aws-vscode generate\n# Project type: %s\n\n", config.Description)
	fullContent := header + string(data)

	if err := os.WriteFile(outputFile, []byte(fullContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
