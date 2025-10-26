package config

import (
	"fmt"

	"github.com/scttfrdmn/lens/pkg/dcv"
)

// QGISEnvironment represents a QGIS environment configuration
type QGISEnvironment struct {
	Name          string
	Description   string
	InstanceType  string
	RequiresGPU   bool
	EBSVolumeSize int
	Packages      []string // System packages to install
	QGISPlugins   []string // QGIS plugins to install
	DCVConfig     *dcv.Config
}

// GetDefaultEnvironments returns all built-in QGIS environments
func GetDefaultEnvironments() map[string]*QGISEnvironment {
	return map[string]*QGISEnvironment{
		"basic-gis": {
			Name:          "basic-gis",
			Description:   "QGIS with essential plugins for general GIS analysis",
			InstanceType:  "t3.xlarge",
			RequiresGPU:   false,
			EBSVolumeSize: 50,
			Packages: []string{
				"qgis",
				"qgis-plugin-grass",
				"python3-qgis",
				"firefox",
				"git",
			},
			QGISPlugins: []string{
				"QuickMapServices",
				"QuickOSM",
			},
			DCVConfig: dcv.DefaultConfig(),
		},
		"advanced-gis": {
			Name:          "advanced-gis",
			Description:   "QGIS + GRASS GIS + SAGA GIS + PostGIS for advanced spatial analysis",
			InstanceType:  "t3.xlarge",
			RequiresGPU:   false,
			EBSVolumeSize: 50,
			Packages: []string{
				"qgis",
				"qgis-plugin-grass",
				"python3-qgis",
				"grass",
				"saga",
				"postgresql",
				"postgis",
				"gdal-bin",
				"python3-gdal",
				"firefox",
				"git",
			},
			QGISPlugins: []string{
				"QuickMapServices",
				"QuickOSM",
				"GRASS",
				"Processing",
			},
			DCVConfig: dcv.DefaultConfig(),
		},
		"remote-sensing": {
			Name:          "remote-sensing",
			Description:   "QGIS + Orfeo Toolbox + SNAP with GPU acceleration for satellite imagery",
			InstanceType:  "g4dn.xlarge",
			RequiresGPU:   true,
			EBSVolumeSize: 100,
			Packages: []string{
				"qgis",
				"qgis-plugin-grass",
				"python3-qgis",
				"otb-bin",
				"python3-otb",
				"gdal-bin",
				"python3-gdal",
				"firefox",
				"git",
			},
			QGISPlugins: []string{
				"QuickMapServices",
				"QuickOSM",
				"Semi-Automatic Classification Plugin",
				"Orfeo Toolbox",
			},
			DCVConfig: dcv.DefaultGPUConfig(),
		},
	}
}

// Get returns a QGIS environment by name
func Get(name string) (*QGISEnvironment, error) {
	envs := GetDefaultEnvironments()
	env, ok := envs[name]
	if !ok {
		return nil, fmt.Errorf("QGIS environment %q not found", name)
	}
	return env, nil
}

// List returns all available QGIS environment names
func List() []string {
	envs := GetDefaultEnvironments()
	names := make([]string, 0, len(envs))
	for name := range envs {
		names = append(names, name)
	}
	return names
}

// GetRecommendedInstanceTypes returns recommended instance types for QGIS
func GetRecommendedInstanceTypes(requiresGPU bool) []string {
	if requiresGPU {
		return []string{
			"g4dn.xlarge",  // 4 vCPU, 16GB, NVIDIA T4 - $0.526/hr
			"g4dn.2xlarge", // 8 vCPU, 32GB, NVIDIA T4 - $0.752/hr
			"g5.xlarge",    // 4 vCPU, 16GB, NVIDIA A10G - $1.006/hr
		}
	}
	return []string{
		"t3.xlarge",  // 4 vCPU, 16GB - $0.166/hr
		"t3.2xlarge", // 8 vCPU, 32GB - $0.333/hr
		"m5.xlarge",  // 4 vCPU, 16GB - $0.192/hr
		"m5.2xlarge", // 8 vCPU, 32GB - $0.384/hr
	}
}
