package cli

import (
	"fmt"

	"github.com/scttfrdmn/aws-ide/pkg/config"
	"github.com/spf13/cobra"
)

// NewEnvCmd creates the env command for managing environment configurations
func NewEnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Environment management commands",
	}

	cmd.AddCommand(NewEnvListCmd())
	cmd.AddCommand(NewEnvValidateCmd())
	return cmd
}

// NewEnvListCmd creates the list subcommand for viewing available environments
func NewEnvListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available VSCode Server environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEnvList()
		},
	}
}

// NewEnvValidateCmd creates the validate subcommand for checking environment configuration validity
func NewEnvValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate ENV_NAME",
		Short: "Validate a VSCode Server environment configuration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEnvValidate(args[0])
		},
	}
}

func runEnvList() error {
	envs, err := config.ListEnvironments()
	if err != nil {
		return fmt.Errorf("failed to list environments: %w", err)
	}

	fmt.Println("Available VSCode Server environments:")
	for _, env := range envs {
		fmt.Printf("  %s\n", env)
	}

	return nil
}

func runEnvValidate(envName string) error {
	env, err := config.LoadEnvironment(envName)
	if err != nil {
		return fmt.Errorf("failed to load environment: %w", err)
	}

	fmt.Printf("Environment %s is valid:\n", envName)
	fmt.Printf("  Name: %s\n", env.Name)
	fmt.Printf("  Instance Type: %s\n", env.InstanceType)
	fmt.Printf("  EBS Volume: %dGB\n", env.EBSVolumeSize)
	fmt.Printf("  Packages: %d\n", len(env.Packages))

	if nodeVer, ok := env.EnvironmentVars["NODEJS_VERSION"]; ok {
		fmt.Printf("  Node.js: %s\n", nodeVer)
	}
	if pythonVer, ok := env.EnvironmentVars["PYTHON_VERSION"]; ok && pythonVer != "" {
		fmt.Printf("  Python: %s\n", pythonVer)
		fmt.Printf("  Pip Packages: %d\n", len(env.PipPackages))
	}
	if goVer, ok := env.EnvironmentVars["GO_VERSION"]; ok {
		fmt.Printf("  Go: %s\n", goVer)
	}

	return nil
}
