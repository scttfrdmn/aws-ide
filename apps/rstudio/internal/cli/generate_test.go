package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCleanPackageName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"tidyverse==1.3.0", "tidyverse"},
		{"ggplot2>=3.3.0", "ggplot2"},
		{"dplyr<=1.0.0", "dplyr"},
		{"shiny~=1.7.0", "shiny"},
		{"data-table!=1.14.0", "data-table"},
		{"simple-package", "simple-package"},
		{"package_with_underscore", "package_with_underscore"},
		{"invalid-line-with-special@chars", "invalid-line-with-special"},
		{"", ""},
	}

	for _, test := range tests {
		result := cleanPackageName(test.input)
		if result != test.expected {
			t.Errorf("cleanPackageName(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestIsCommonScientificPackage(t *testing.T) {
	// Note: Implementation currently checks for Python packages
	// These should be updated when RStudio-specific logic is implemented
	tests := []struct {
		pkg      string
		expected bool
	}{
		{"numpy", true},
		{"pandas", true},
		{"matplotlib", true},
		{"scipy", true},
		{"scikit-learn", true},
		{"torch", true},
		{"tensorflow", true},
		{"keras", true},
		{"unknown-package", false},
		{"", false},
	}

	for _, test := range tests {
		result := isCommonScientificPackage(test.pkg)
		if result != test.expected {
			t.Errorf("isCommonScientificPackage(%q) = %t, expected %t", test.pkg, result, test.expected)
		}
	}
}

func TestIsSystemPackage(t *testing.T) {
	// Note: Implementation currently checks for Python system packages
	// These should be updated when RStudio-specific logic is implemented
	tests := []struct {
		pkg      string
		expected bool
	}{
		{"pip", true},
		{"setuptools", true},
		{"wheel", true},
		{"pkg-resources", true},
		{"numpy", false},
		{"pandas", false},
		{"", false},
	}

	for _, test := range tests {
		result := isSystemPackage(test.pkg)
		if result != test.expected {
			t.Errorf("isSystemPackage(%q) = %t, expected %t", test.pkg, result, test.expected)
		}
	}
}

func TestIsKnownPackage(t *testing.T) {
	// Note: Implementation currently checks for Python packages
	// These should be updated when RStudio-specific logic is implemented
	tests := []struct {
		pkg      string
		expected bool
	}{
		{"pandas", true},
		{"numpy", true},
		{"torch", true},
		{"tensorflow", true},
		{"requests", true},
		{"flask", true},
		{"unknown-package", false},
		{"", false},
	}

	for _, test := range tests {
		result := isKnownPackage(test.pkg)
		if result != test.expected {
			t.Errorf("isKnownPackage(%q) = %t, expected %t", test.pkg, result, test.expected)
		}
	}
}

func TestSuggestInstanceType(t *testing.T) {
	// Note: Implementation currently checks for Python packages
	// These should be updated when RStudio-specific logic is implemented
	tests := []struct {
		packages []string
		expected string
	}{
		{[]string{"pandas", "numpy"}, "m7g.medium"},
		{[]string{"torch", "transformers"}, "m7g.large"},
		{[]string{"scipy", "scikit-learn"}, "c7g.large"},
		{[]string{"tensorflow", "datasets"}, "m7g.large"},
		{[]string{"opencv-python", "scipy"}, "c7g.large"},
		{[]string{}, "m7g.medium"},
	}

	for _, test := range tests {
		result := suggestInstanceType(test.packages)
		if result != test.expected {
			t.Errorf("suggestInstanceType(%v) = %q, expected %q", test.packages, result, test.expected)
		}
	}
}

func TestSuggestEBSSize(t *testing.T) {
	// Note: Implementation currently checks for Python packages
	// These should be updated when RStudio-specific logic is implemented
	tests := []struct {
		packages []string
		expected int
	}{
		{[]string{"pandas", "numpy"}, 15},
		{[]string{"torch", "transformers"}, 40},
		{[]string{"tensorflow"}, 40},
		{[]string{"scipy", "scikit-learn"}, 25},
		{[]string{"opencv-python"}, 25},
		{[]string{}, 15},
	}

	for _, test := range tests {
		result := suggestEBSSize(test.packages)
		if result != test.expected {
			t.Errorf("suggestEBSSize(%v) = %d, expected %d", test.packages, result, test.expected)
		}
	}
}

func TestParseRequirementsTxt(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Create requirements.txt file
	reqFile := filepath.Join(tmpDir, "requirements.txt")
	reqContent := `# This is a comment
tidyverse==1.3.0
ggplot2>=3.3.0
dplyr

# Another comment
shiny~=1.7.0
`
	err := os.WriteFile(reqFile, []byte(reqContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create requirements.txt: %v", err)
	}

	// Test parsing
	packages, err := parseRequirementsTxt(reqFile)
	if err != nil {
		t.Fatalf("parseRequirementsTxt failed: %v", err)
	}

	expected := []string{"tidyverse", "ggplot2", "dplyr", "shiny"}
	if len(packages) != len(expected) {
		t.Errorf("Expected %d packages, got %d", len(expected), len(packages))
	}

	for _, exp := range expected {
		found := false
		for _, pkg := range packages {
			if pkg == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected package %q not found", exp)
		}
	}
}

func TestParseRequirementsTxt_Directory(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Create requirements.txt file in directory
	reqFile := filepath.Join(tmpDir, "requirements.txt")
	reqContent := `tidyverse==1.3.0
ggplot2>=3.3.0
`
	err := os.WriteFile(reqFile, []byte(reqContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create requirements.txt: %v", err)
	}

	// Test parsing directory
	packages, err := parseRequirementsTxt(tmpDir)
	if err != nil {
		t.Fatalf("parseRequirementsTxt failed: %v", err)
	}

	expected := []string{"tidyverse", "ggplot2"}
	if len(packages) != len(expected) {
		t.Errorf("Expected %d packages, got %d", len(expected), len(packages))
	}
}

func TestParseRequirementsTxt_NotFound(t *testing.T) {
	// Test with non-existent file
	_, err := parseRequirementsTxt("/non/existent/file.txt")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestScanNotebooksForImports(t *testing.T) {
	// Note: RStudio implementation currently uses Python notebook format (.ipynb)
	// and Python import syntax. This test maintains the Jupyter-style format
	// until the implementation is adapted for R notebooks (.Rmd).

	// Create temp directory for test
	tmpDir := t.TempDir()

	// Create a test notebook (using .ipynb format for now)
	notebookFile := filepath.Join(tmpDir, "test.ipynb")
	notebookContent := `{
 "cells": [
  {
   "cell_type": "code",
   "execution_count": null,
   "metadata": {},
   "source": [
    "import pandas as pd\n",
    "import numpy as np\n",
    "from sklearn import datasets\n",
    "import matplotlib.pyplot as plt\n"
   ]
  },
  {
   "cell_type": "markdown",
   "metadata": {},
   "source": [
    "This is a markdown cell"
   ]
  },
  {
   "cell_type": "code",
   "execution_count": null,
   "metadata": {},
   "source": [
    "import torch\n",
    "from PIL import Image\n"
   ]
  }
 ],
 "metadata": {
  "kernelspec": {
   "display_name": "Python 3",
   "language": "python",
   "name": "python3"
  }
 },
 "nbformat": 4,
 "nbformat_minor": 4
}`

	err := os.WriteFile(notebookFile, []byte(notebookContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create notebook file: %v", err)
	}

	// Test scanning
	packages, err := scanNotebooksForImports(tmpDir)
	if err != nil {
		t.Fatalf("scanNotebooksForImports failed: %v", err)
	}

	// Expected packages based on import mapping
	expectedPackages := map[string]bool{
		"pandas":       true,
		"numpy":        true,
		"scikit-learn": true,
		"matplotlib":   true,
		"torch":        true,
		"Pillow":       true,
	}

	if len(packages) == 0 {
		t.Error("Expected packages to be found, got empty list")
	}

	for _, pkg := range packages {
		if !expectedPackages[pkg] {
			t.Errorf("Unexpected package found: %s", pkg)
		}
	}
}

func TestScanNotebooksForImports_NoNotebooks(t *testing.T) {
	// Create temp directory with no notebooks
	tmpDir := t.TempDir()

	// Create a regular Python file (should be ignored)
	pyFile := filepath.Join(tmpDir, "test.py")
	err := os.WriteFile(pyFile, []byte("import pandas\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create Python file: %v", err)
	}

	// Test scanning
	packages, err := scanNotebooksForImports(tmpDir)
	if err != nil {
		t.Fatalf("scanNotebooksForImports failed: %v", err)
	}

	if len(packages) != 0 {
		t.Errorf("Expected no packages for directory without notebooks, got: %d", len(packages))
	}
}

func TestScanNotebooksForImports_InvalidJSON(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Create invalid notebook file
	notebookFile := filepath.Join(tmpDir, "invalid.ipynb")
	err := os.WriteFile(notebookFile, []byte("invalid json"), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid notebook file: %v", err)
	}

	// Test scanning (should handle invalid files gracefully)
	packages, err := scanNotebooksForImports(tmpDir)
	if err != nil {
		t.Fatalf("scanNotebooksForImports failed: %v", err)
	}

	// Should return empty list for invalid notebooks
	if len(packages) != 0 {
		t.Errorf("Expected no packages for invalid notebook, got: %d", len(packages))
	}
}

func TestRunGenerate_Integration(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Create requirements.txt
	reqFile := filepath.Join(tmpDir, "requirements.txt")
	reqContent := `tidyverse==1.3.0
ggplot2>=3.3.0
dplyr
caret
`
	err := os.WriteFile(reqFile, []byte(reqContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create requirements.txt: %v", err)
	}

	// Create output file path
	outputFile := filepath.Join(tmpDir, "test-env.yaml")

	// Run generate
	err = runGenerate(tmpDir, outputFile, "test-env", "m7g.medium", false)
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
	if !strings.Contains(contentStr, "name: test-env") {
		t.Error("Output should contain environment name")
	}

	if !strings.Contains(contentStr, "instance_type: c7g.large") {
		t.Error("Output should suggest c7g.large for ML packages")
	}

	if !strings.Contains(contentStr, "tidyverse") {
		t.Error("Output should contain tidyverse package")
	}

	if !strings.Contains(contentStr, "caret") {
		t.Error("Output should contain caret package")
	}
}
