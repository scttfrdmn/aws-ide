package main

import (
	"fmt"
	"os"

	"github.com/scttfrdmn/aws-ide/apps/vscode/internal/cli"
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
		Use:   "aws-vscode",
		Short: "VSCode Server on AWS Graviton with Session Manager & advanced networking",
		Long: `aws-vscode is a powerful CLI tool for launching VSCode Server (code-server) instances
on AWS EC2 Graviton processors with professional-grade security and networking.

Features:
• Full VSCode experience in your browser
• Session Manager & SSH connection methods
• Private subnet support with NAT Gateway
• Smart security groups and key management
• Built-in environments for web, Python, Go, and fullstack development
• Cost-aware infrastructure with reuse strategies
• Automatic extension installation`,
		Version: fmt.Sprintf("v%s (platform: v%s, commit: %s, date: %s)", version, pkg.Version, commit, date),
	}

	// Add subcommands
	rootCmd.AddCommand(cli.NewLaunchCmd())
	rootCmd.AddCommand(cli.NewListCmd())
	rootCmd.AddCommand(cli.NewConnectCmd())
	rootCmd.AddCommand(cli.NewStopCmd())
	rootCmd.AddCommand(cli.NewStartCmd())
	rootCmd.AddCommand(cli.NewTerminateCmd())
	rootCmd.AddCommand(cli.NewStatusCmd())
	rootCmd.AddCommand(cli.NewEnvCmd())
	rootCmd.AddCommand(cli.NewKeyCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
