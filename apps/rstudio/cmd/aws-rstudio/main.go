package main

import (
	"fmt"
	"os"

	"github.com/scttfrdmn/aws-ide/apps/rstudio/internal/cli"
	"github.com/scttfrdmn/aws-ide/pkg"
	"github.com/scttfrdmn/aws-ide/pkg/config"
	"github.com/spf13/cobra"
)

var (
	version = "0.6.0" // App version
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	if err := config.EnsureConfigDir(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create config directory: %v\n", err)
		os.Exit(1)
	}

	rootCmd := &cobra.Command{
		Use:   "aws-rstudio",
		Short: "Secure RStudio Server on AWS Graviton with Session Manager & advanced networking",
		Long: `aws-rstudio is a powerful CLI tool for launching secure RStudio Server instances
on AWS EC2 Graviton processors with professional-grade security and networking.

Features:
• Session Manager & SSH connection methods
• Private subnet support with NAT Gateway
• Smart security groups and key management
• Built-in environments for data science, statistical computing, and research
• Cost-aware infrastructure with reuse strategies`,
		Version: fmt.Sprintf("v%s (platform: v%s, commit: %s, date: %s)", version, pkg.Version, commit, date),
	}

	// Add subcommands
	rootCmd.AddCommand(cli.NewLaunchCmd())
	rootCmd.AddCommand(cli.NewListCmd())
	rootCmd.AddCommand(cli.NewConnectCmd())
	rootCmd.AddCommand(cli.NewStopCmd())
	rootCmd.AddCommand(cli.NewStartCmd())
	rootCmd.AddCommand(cli.NewTerminateCmd())
	rootCmd.AddCommand(cli.NewCreateAMICmd())
	rootCmd.AddCommand(cli.NewListAMIsCmd())
	rootCmd.AddCommand(cli.NewDeleteAMICmd())
	rootCmd.AddCommand(cli.NewEnvCmd())
	rootCmd.AddCommand(cli.NewStatusCmd())
	rootCmd.AddCommand(cli.NewGenerateCmd())
	rootCmd.AddCommand(cli.NewKeyCmd())
	rootCmd.AddCommand(cli.NewExportConfigCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
