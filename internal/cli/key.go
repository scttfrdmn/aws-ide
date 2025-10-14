package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/scttfrdmn/aws-jupyter/internal/aws"
	"github.com/scttfrdmn/aws-jupyter/internal/config"
	"github.com/spf13/cobra"
)

// NewKeyCmd creates the key command for managing SSH key pairs
func NewKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key",
		Short: "Manage SSH key pairs",
		Long:  "Commands for managing SSH key pairs used by aws-jupyter instances",
	}

	cmd.AddCommand(newKeyListCmd())
	cmd.AddCommand(newKeyCleanupCmd())
	cmd.AddCommand(newKeyValidateCmd())
	cmd.AddCommand(newKeyShowCmd())

	return cmd
}

func newKeyListCmd() *cobra.Command {
	var profile string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List SSH key pairs",
		Long:  "List both locally stored and AWS key pairs",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runKeyList(profile)
		},
	}

	cmd.Flags().StringVarP(&profile, "profile", "p", "default", "AWS profile to use")

	return cmd
}

func newKeyCleanupCmd() *cobra.Command {
	var (
		profile string
		dryRun  bool
	)

	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Clean up orphaned local keys",
		Long:  "Remove locally stored private keys that no longer exist in AWS",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runKeyCleanup(profile, dryRun)
		},
	}

	cmd.Flags().StringVarP(&profile, "profile", "p", "default", "AWS profile to use")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be cleaned up")

	return cmd
}

func newKeyValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate key permissions",
		Long:  "Check that stored private keys have secure permissions (600)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runKeyValidate()
		},
	}

	return cmd
}

func newKeyShowCmd() *cobra.Command {
	var profile string

	cmd := &cobra.Command{
		Use:   "show [key-name]",
		Short: "Show key pair information",
		Long:  "Display detailed information about a specific key pair",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var keyName string
			if len(args) > 0 {
				keyName = args[0]
			}
			return runKeyShow(keyName, profile)
		},
	}

	cmd.Flags().StringVarP(&profile, "profile", "p", "default", "AWS profile to use")

	return cmd
}

func runKeyList(profile string) error {
	ctx := context.Background()

	// Setup key storage
	keyStorage, err := config.DefaultKeyStorage()
	if err != nil {
		return fmt.Errorf("failed to initialize key storage: %w", err)
	}

	// List locally stored keys
	storedKeys, err := keyStorage.ListStoredKeys()
	if err != nil {
		return fmt.Errorf("failed to list stored keys: %w", err)
	}

	// Create AWS client to list remote keys
	ec2Client, err := aws.NewEC2Client(ctx, profile)
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	awsKeys, err := ec2Client.ListKeyPairs(ctx)
	if err != nil {
		return fmt.Errorf("failed to list AWS key pairs: %w", err)
	}

	// Display results
	fmt.Printf("Local SSH Keys (%d):\n", len(storedKeys))
	if len(storedKeys) == 0 {
		fmt.Println("  No keys stored locally")
	} else {
		for _, key := range storedKeys {
			keyPath := keyStorage.GetKeyPath(key)
			if _, err := os.Stat(keyPath); err == nil {
				fmt.Printf("  ✓ %s\n", key)
			} else {
				fmt.Printf("  ✗ %s (missing file)\n", key)
			}
		}
	}

	fmt.Printf("\nAWS Key Pairs in %s (%d):\n", ec2Client.GetRegion(), len(awsKeys))
	if len(awsKeys) == 0 {
		fmt.Println("  No key pairs in AWS")
	} else {
		for _, awsKey := range awsKeys {
			hasLocal := false
			for _, stored := range storedKeys {
				if stored == awsKey.Name {
					hasLocal = true
					break
				}
			}
			status := ""
			if awsKey.CreatedBy == "aws-jupyter" {
				status = " [aws-jupyter]"
			}
			if hasLocal {
				fmt.Printf("  ✓ %s (has local key)%s\n", awsKey.Name, status)
			} else {
				fmt.Printf("  ⚠ %s (no local key)%s\n", awsKey.Name, status)
			}
		}
	}

	return nil
}

func runKeyCleanup(profile string, dryRun bool) error {
	ctx := context.Background()

	keyStorage, err := config.DefaultKeyStorage()
	if err != nil {
		return fmt.Errorf("failed to initialize key storage: %w", err)
	}

	ec2Client, err := aws.NewEC2Client(ctx, profile)
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	awsKeysList, err := ec2Client.ListKeyPairs(ctx)
	if err != nil {
		return fmt.Errorf("failed to list AWS key pairs: %w", err)
	}

	// Convert to string slice for cleanup
	awsKeyNames := make([]string, len(awsKeysList))
	for i, key := range awsKeysList {
		awsKeyNames[i] = key.Name
	}

	if dryRun {
		fmt.Println("[DRY RUN] Would clean up orphaned keys:")
		err = keyStorage.CleanupOrphanedKeys(awsKeyNames)
	} else {
		fmt.Println("Cleaning up orphaned keys...")
		err = keyStorage.CleanupOrphanedKeys(awsKeyNames)
	}

	if err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	fmt.Println("Cleanup completed")
	return nil
}

func runKeyValidate() error {
	keyStorage, err := config.DefaultKeyStorage()
	if err != nil {
		return fmt.Errorf("failed to initialize key storage: %w", err)
	}

	storedKeys, err := keyStorage.ListStoredKeys()
	if err != nil {
		return fmt.Errorf("failed to list stored keys: %w", err)
	}

	if len(storedKeys) == 0 {
		fmt.Println("No keys to validate")
		return nil
	}

	fmt.Printf("Validating %d stored keys:\n", len(storedKeys))
	hasErrors := false

	for _, keyName := range storedKeys {
		if err := keyStorage.ValidateKeyPermissions(keyName); err != nil {
			fmt.Printf("  ✗ %s: %v\n", keyName, err)
			hasErrors = true
		} else {
			fmt.Printf("  ✓ %s: permissions OK\n", keyName)
		}
	}

	if hasErrors {
		fmt.Println("\nTo fix permissions, run: chmod 600 ~/.aws-jupyter/keys/*.pem")
	}

	return nil
}

func runKeyShow(keyName, profile string) error {
	ctx := context.Background()

	keyStorage, err := config.DefaultKeyStorage()
	if err != nil {
		return fmt.Errorf("failed to initialize key storage: %w", err)
	}

	ec2Client, err := aws.NewEC2Client(ctx, profile)
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	// If no key name provided, show the default key that would be used
	if keyName == "" {
		strategy := aws.DefaultKeyPairStrategy(ec2Client.GetRegion())
		keyName = strategy.GetDefaultKeyName()
		fmt.Printf("Showing default key for region %s:\n", ec2Client.GetRegion())
	}

	fmt.Printf("Key Pair: %s\n", keyName)
	fmt.Printf("Region: %s\n", ec2Client.GetRegion())

	// Check AWS
	exists, err := ec2Client.KeyPairExists(ctx, keyName)
	if err != nil {
		fmt.Printf("AWS Status: Error checking (%v)\n", err)
	} else if exists {
		fmt.Printf("AWS Status: ✓ Exists\n")
	} else {
		fmt.Printf("AWS Status: ✗ Not found\n")
	}

	// Check local storage
	if keyStorage.HasPrivateKey(keyName) {
		keyPath := keyStorage.GetKeyPath(keyName)
		fmt.Printf("Local Key: ✓ %s\n", keyPath)

		// Check permissions
		if err := keyStorage.ValidateKeyPermissions(keyName); err != nil {
			fmt.Printf("Permissions: ✗ %v\n", err)
		} else {
			fmt.Printf("Permissions: ✓ Secure (600)\n")
		}
	} else {
		fmt.Printf("Local Key: ✗ Not stored locally\n")
	}

	return nil
}
