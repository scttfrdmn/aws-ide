package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/scttfrdmn/aws-jupyter/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// NewGenerateCmd creates the generate command for creating environment configs from local Python projects
func NewGenerateCmd() *cobra.Command {
	var (
		source        string
		output        string
		name          string
		instanceType  string
		scanNotebooks bool
	)

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate environment config from local setup",
		Long: `Generate an environment configuration file by analyzing your local Python environment,
requirements files, conda environment, or Jupyter notebooks.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerate(source, output, name, instanceType, scanNotebooks)
		},
	}

	cmd.Flags().StringVarP(&source, "source", "s", ".", "Source directory or file to analyze")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (default: <name>.yaml)")
	cmd.Flags().StringVarP(&name, "name", "n", "generated", "Environment name")
	cmd.Flags().StringVarP(&instanceType, "instance-type", "t", "m7g.medium", "Default instance type")
	cmd.Flags().BoolVar(&scanNotebooks, "scan-notebooks", true, "Scan .ipynb files for imports")

	return cmd
}

func runGenerate(source, output, name, instanceType string, scanNotebooks bool) error {
	if output == "" {
		output = fmt.Sprintf("%s.yaml", name)
	}

	fmt.Printf("Analyzing local environment from: %s\n", source)

	env := createBaseEnvironment(name, instanceType)
	packages := collectAllPackages(source, scanNotebooks)
	finalPackages := packageMapToSortedSlice(packages)

	env.PipPackages = append(env.PipPackages, finalPackages...)
	optimizeEnvironmentSettings(env, instanceType, finalPackages)

	return writeEnvironmentFile(env, output, finalPackages)
}

func createBaseEnvironment(name, instanceType string) *config.Environment {
	return &config.Environment{
		Name:              name,
		InstanceType:      instanceType,
		AMIBase:           "ubuntu22-arm64",
		EBSVolumeSize:     20,
		Packages:          []string{"python3-pip", "python3-dev", "jupyter", "git", "htop", "awscli"},
		PipPackages:       []string{"jupyterlab", "notebook", "ipywidgets"},
		JupyterExtensions: []string{"jupyterlab"},
		EnvironmentVars:   map[string]string{"PYTHONPATH": "/home/ubuntu/notebooks"},
	}
}

func collectAllPackages(source string, scanNotebooks bool) map[string]bool {
	packages := make(map[string]bool)

	// 1. Check for requirements.txt
	if reqPackages, err := parseRequirementsTxt(source); err == nil {
		fmt.Printf("Found requirements.txt with %d packages\n", len(reqPackages))
		addPackagesToMap(packages, reqPackages)
	}

	// 2. Check for conda environment
	if condaPackages, err := analyzeCondaEnvironment(); err == nil {
		fmt.Printf("Found conda environment with %d pip packages\n", len(condaPackages))
		addPackagesToMap(packages, condaPackages)
	}

	// 3. Analyze current pip environment
	if pipPackages, err := analyzePipEnvironment(); err == nil {
		fmt.Printf("Found current pip environment with %d packages\n", len(pipPackages))
		addPackagesToMap(packages, pipPackages)
	}

	// 4. Scan notebooks for imports
	if scanNotebooks {
		if nbPackages, err := scanNotebooksForImports(source); err == nil {
			fmt.Printf("Found %d unique imports from notebooks\n", len(nbPackages))
			addPackagesToMap(packages, nbPackages)
		}
	}

	return packages
}

func addPackagesToMap(packages map[string]bool, newPackages []string) {
	for _, pkg := range newPackages {
		packages[pkg] = true
	}
}

func packageMapToSortedSlice(packages map[string]bool) []string {
	var finalPackages []string
	for pkg := range packages {
		finalPackages = append(finalPackages, pkg)
	}
	sort.Strings(finalPackages)
	return finalPackages
}

func optimizeEnvironmentSettings(env *config.Environment, originalInstanceType string, packages []string) {
	// Detect and suggest instance type based on packages
	if originalInstanceType == "m7g.medium" {
		env.InstanceType = suggestInstanceType(packages)
		if env.InstanceType != originalInstanceType {
			fmt.Printf("Suggested instance type: %s (based on detected packages)\n", env.InstanceType)
		}
	}

	// Suggest EBS size based on packages
	env.EBSVolumeSize = suggestEBSSize(packages)
}

func writeEnvironmentFile(env *config.Environment, output string, packages []string) error {
	data, err := yaml.Marshal(env)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	if err := os.WriteFile(output, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("Generated environment config: %s\n", output)
	fmt.Printf("Total packages: %d\n", len(packages))
	fmt.Printf("Instance type: %s\n", env.InstanceType)
	fmt.Printf("EBS volume: %dGB\n", env.EBSVolumeSize)

	return nil
}

func parseRequirementsTxt(source string) ([]string, error) {
	var paths []string

	// Check for requirements.txt in source directory
	if stat, err := os.Stat(source); err == nil {
		if stat.IsDir() {
			paths = append(paths, filepath.Join(source, "requirements.txt"))
		} else {
			paths = append(paths, source)
		}
	}

	var packages []string
	for _, path := range paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}

		file, err := os.Open(path)
		if err != nil {
			continue
		}
		defer func() {
			if closeErr := file.Close(); closeErr != nil {
				// Log error but don't fail the entire operation
				fmt.Printf("Warning: failed to close file %s: %v\n", path, closeErr)
			}
		}()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			// Remove version specifiers and clean up
			pkg := cleanPackageName(line)
			if pkg != "" {
				packages = append(packages, pkg)
			}
		}
	}

	if len(packages) == 0 {
		return nil, fmt.Errorf("no requirements found")
	}

	return packages, nil
}

func analyzeCondaEnvironment() ([]string, error) {
	cmd := exec.Command("conda", "list", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var condaPackages []struct {
		Name    string `json:"name"`
		Channel string `json:"channel"`
	}

	if err := json.Unmarshal(output, &condaPackages); err != nil {
		return nil, err
	}

	var packages []string
	for _, pkg := range condaPackages {
		// Only include pip-installed packages or common scientific packages
		if pkg.Channel == "pypi" || isCommonScientificPackage(pkg.Name) {
			packages = append(packages, pkg.Name)
		}
	}

	return packages, nil
}

func analyzePipEnvironment() ([]string, error) {
	cmd := exec.Command("pip", "freeze")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var packages []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pkg := cleanPackageName(line)
		if pkg != "" && !isSystemPackage(pkg) {
			packages = append(packages, pkg)
		}
	}

	return packages, nil
}

func scanNotebooksForImports(source string) ([]string, error) {
	notebooks, err := findNotebookFiles(source)
	if err != nil {
		return nil, err
	}

	imports := extractImportsFromNotebooks(notebooks)
	packages := mapImportsToPackages(imports)

	return packages, nil
}

func findNotebookFiles(source string) ([]string, error) {
	var notebooks []string
	err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".ipynb") {
			notebooks = append(notebooks, path)
		}
		return nil
	})
	return notebooks, err
}

func extractImportsFromNotebooks(notebooks []string) map[string]bool {
	imports := make(map[string]bool)
	importRegex := regexp.MustCompile(`(?m)^(?:from\s+(\w+)|import\s+(\w+))`)

	for _, notebook := range notebooks {
		extractImportsFromNotebook(notebook, imports, importRegex)
	}

	return imports
}

func extractImportsFromNotebook(notebookPath string, imports map[string]bool, importRegex *regexp.Regexp) {
	content, err := os.ReadFile(notebookPath)
	if err != nil {
		return
	}

	var nb struct {
		Cells []struct {
			CellType string   `json:"cell_type"`
			Source   []string `json:"source"`
		} `json:"cells"`
	}

	if err := json.Unmarshal(content, &nb); err != nil {
		return
	}

	for _, cell := range nb.Cells {
		if cell.CellType == "code" {
			extractImportsFromCell(cell.Source, imports, importRegex)
		}
	}
}

func extractImportsFromCell(source []string, imports map[string]bool, importRegex *regexp.Regexp) {
	for _, line := range source {
		matches := importRegex.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if match[1] != "" {
				imports[match[1]] = true
			}
			if match[2] != "" {
				imports[match[2]] = true
			}
		}
	}
}

func mapImportsToPackages(imports map[string]bool) []string {
	// Map common import names to package names
	packageMap := map[string]string{
		"cv2":     "opencv-python",
		"sklearn": "scikit-learn",
		"PIL":     "Pillow",
		"torch":   "torch",
		"tf":      "tensorflow",
		"pd":      "pandas",
		"np":      "numpy",
		"plt":     "matplotlib",
		"sns":     "seaborn",
	}

	var packages []string
	for imp := range imports {
		if pkg, exists := packageMap[imp]; exists {
			packages = append(packages, pkg)
		} else if isKnownPackage(imp) {
			packages = append(packages, imp)
		}
	}

	return packages
}

func cleanPackageName(line string) string {
	// Remove version specifiers (==, >=, <=, >, <, ~=, !=)
	re := regexp.MustCompile(`^([a-zA-Z0-9_-]+)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func isCommonScientificPackage(pkg string) bool {
	common := map[string]bool{
		"numpy": true, "pandas": true, "matplotlib": true, "scipy": true,
		"scikit-learn": true, "seaborn": true, "plotly": true, "bokeh": true,
		"jupyter": true, "jupyterlab": true, "notebook": true, "ipywidgets": true,
		"torch": true, "tensorflow": true, "keras": true, "transformers": true,
		"requests": true, "beautifulsoup4": true, "lxml": true, "openpyxl": true,
	}
	return common[pkg]
}

func isSystemPackage(pkg string) bool {
	system := map[string]bool{
		"pip": true, "setuptools": true, "wheel": true, "pkg-resources": true,
	}
	return system[pkg]
}

func isKnownPackage(pkg string) bool {
	known := map[string]bool{
		"pandas": true, "numpy": true, "matplotlib": true, "seaborn": true,
		"plotly": true, "bokeh": true, "scipy": true, "scikit-learn": true,
		"requests": true, "beautifulsoup4": true, "lxml": true, "openpyxl": true,
		"boto3": true, "botocore": true, "sqlalchemy": true, "psycopg2": true,
		"torch": true, "tensorflow": true, "transformers": true, "datasets": true,
		"accelerate": true, "evaluate": true, "wandb": true, "tensorboard": true,
		"opencv": true, "Pillow": true, "imageio": true, "tqdm": true,
		"click": true, "flask": true, "fastapi": true, "streamlit": true,
	}
	return known[pkg]
}

func suggestInstanceType(packages []string) string {
	hasML := false
	hasHeavyCompute := false

	for _, pkg := range packages {
		switch pkg {
		case "torch", "tensorflow", "transformers", "datasets":
			hasML = true
		case "scipy", "scikit-learn", "opencv-python":
			hasHeavyCompute = true
		}
	}

	if hasML {
		return "m7g.large" // More memory for ML workloads
	} else if hasHeavyCompute {
		return "c7g.large" // Compute optimized
	}

	return "m7g.medium" // Default
}

func suggestEBSSize(packages []string) int {
	baseSize := 15

	hasML := false
	hasLargeData := false

	for _, pkg := range packages {
		switch pkg {
		case "torch", "tensorflow", "transformers":
			hasML = true
		case "opencv-python", "scipy", "scikit-learn":
			hasLargeData = true
		}
	}

	if hasML {
		return 40 // ML models can be large
	} else if hasLargeData {
		return 25 // Scientific computing needs space
	}

	return baseSize
}
