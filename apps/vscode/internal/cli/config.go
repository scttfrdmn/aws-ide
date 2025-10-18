package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/scttfrdmn/aws-ide/pkg/config"
	"github.com/spf13/cobra"
)

// NewConfigCmd creates the config command for managing user configuration
func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage user configuration",
		Long: `Manage aws-ide user configuration settings.

The config file is stored at ~/.aws-ide/config.yaml and contains
default settings for all aws-ide tools (jupyter, rstudio, vscode).

Examples:
  # Initialize config with defaults
  aws-vscode config init

  # View current configuration
  aws-vscode config show

  # Set a configuration value
  aws-vscode config set default_region us-west-2
  aws-vscode config set default_instance_type t4g.large
  aws-vscode config set vscode.port 8080

  # Get a specific configuration value
  aws-vscode config get default_region`,
	}

	cmd.AddCommand(NewConfigInitCmd())
	cmd.AddCommand(NewConfigShowCmd())
	cmd.AddCommand(NewConfigSetCmd())
	cmd.AddCommand(NewConfigGetCmd())

	return cmd
}

// NewConfigInitCmd creates the config init subcommand
func NewConfigInitCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize config file with defaults",
		Long: `Create a new config file with default settings.

If a config file already exists, this command will fail unless --force is used.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigInit(force)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing config file")

	return cmd
}

// NewConfigShowCmd creates the config show subcommand
func NewConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Long:  `Display the current configuration settings.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigShow()
		},
	}
}

// NewConfigSetCmd creates the config set subcommand
func NewConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set KEY VALUE",
		Short: "Set a configuration value",
		Long: `Set a configuration value.

Supported keys:
  default_region              - AWS region
  default_profile             - AWS profile
  default_instance_type       - EC2 instance type
  default_ebs_size            - EBS volume size (GB)
  default_ami_base            - Base AMI name
  default_subnet_type         - Subnet type (public/private)
  prefer_ipv6                 - Prefer IPv6 (true/false)
  idle_timeout                - Idle timeout duration
  auto_terminate              - Auto-terminate on idle (true/false)
  confirm_destructive         - Confirm destructive operations (true/false)
  enable_cost_tracking        - Enable cost tracking (true/false)
  cost_alert_threshold        - Cost alert threshold ($)
  vscode.default_environment  - Default VSCode environment
  vscode.default_instance_type - Default instance type for VSCode
  vscode.default_ebs_size     - Default EBS size for VSCode
  vscode.port                 - Default port for VSCode`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigSet(args[0], args[1])
		},
	}
}

// NewConfigGetCmd creates the config get subcommand
func NewConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get KEY",
		Short: "Get a configuration value",
		Long:  `Get a configuration value by key.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigGet(args[0])
		},
	}
}

func runConfigInit(force bool) error {
	configPath := config.GetUserConfigPath()

	// Check if file exists
	if _, err := os.Stat(configPath); err == nil && !force {
		return fmt.Errorf("config file already exists at %s\nUse --force to overwrite", configPath)
	}

	if err := config.InitUserConfig(); err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	fmt.Printf("✓ Initialized config file: %s\n", configPath)
	fmt.Println("\nEdit the file to customize your settings, or use:")
	fmt.Println("  aws-vscode config set KEY VALUE")

	return nil
}

func runConfigShow() error {
	cfg, err := config.LoadUserConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println("Current Configuration:")
	fmt.Println()
	fmt.Println("AWS Settings:")
	fmt.Printf("  default_region:         %s\n", valueOrDefault(cfg.DefaultRegion, "(from AWS SDK)"))
	fmt.Printf("  default_profile:        %s\n", valueOrDefault(cfg.DefaultProfile, "(from AWS SDK)"))
	fmt.Println()
	fmt.Println("Instance Defaults:")
	fmt.Printf("  default_instance_type:  %s\n", cfg.DefaultInstanceType)
	fmt.Printf("  default_ebs_size:       %d GB\n", cfg.DefaultEBSSize)
	fmt.Printf("  default_ami_base:       %s\n", cfg.DefaultAMIBase)
	fmt.Println()
	fmt.Println("Networking:")
	fmt.Printf("  default_subnet_type:    %s\n", cfg.DefaultSubnetType)
	fmt.Printf("  prefer_ipv6:            %t\n", cfg.PreferIPv6)
	fmt.Println()
	fmt.Println("Behavior:")
	fmt.Printf("  idle_timeout:           %s\n", cfg.IdleTimeout)
	fmt.Printf("  auto_terminate:         %t\n", cfg.AutoTerminate)
	fmt.Printf("  confirm_destructive:    %t\n", cfg.ConfirmDestructive)
	fmt.Println()
	fmt.Println("Cost Tracking:")
	fmt.Printf("  enable_cost_tracking:   %t\n", cfg.EnableCostTracking)
	fmt.Printf("  cost_alert_threshold:   $%.2f/month\n", cfg.CostAlertThreshold)
	fmt.Println()
	fmt.Println("VSCode Settings:")
	if cfg.VSCode != nil {
		fmt.Printf("  default_environment:    %s\n", valueOrDefault(cfg.VSCode.DefaultEnvironment, "(not set)"))
		fmt.Printf("  default_instance_type:  %s\n", valueOrDefault(cfg.VSCode.DefaultInstanceType, "(use global)"))
		if cfg.VSCode.DefaultEBSSize > 0 {
			fmt.Printf("  default_ebs_size:       %d GB\n", cfg.VSCode.DefaultEBSSize)
		} else {
			fmt.Printf("  default_ebs_size:       (use global)\n")
		}
		fmt.Printf("  port:                   %d\n", cfg.VSCode.Port)
	}

	fmt.Printf("\nConfig file: %s\n", config.GetUserConfigPath())

	return nil
}

func runConfigSet(key, value string) error {
	cfg, err := config.LoadUserConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Handle nested keys (e.g., vscode.port)
	parts := strings.SplitN(key, ".", 2)

	if len(parts) == 2 {
		// App-specific setting
		appName := parts[0]
		appKey := parts[1]

		if appName == "vscode" {
			if cfg.VSCode == nil {
				cfg.VSCode = &config.AppConfig{}
			}
			if err := setAppConfigValue(cfg.VSCode, appKey, value); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("unsupported app: %s", appName)
		}
	} else {
		// Global setting
		if err := setGlobalConfigValue(cfg, key, value); err != nil {
			return err
		}
	}

	if err := config.SaveUserConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ Set %s = %s\n", key, value)

	return nil
}

func runConfigGet(key string) error {
	cfg, err := config.LoadUserConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Handle nested keys
	parts := strings.SplitN(key, ".", 2)

	var value string

	if len(parts) == 2 {
		// App-specific setting
		appName := parts[0]
		appKey := parts[1]

		if appName == "vscode" && cfg.VSCode != nil {
			value, err = getAppConfigValue(cfg.VSCode, appKey)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("unsupported key: %s", key)
		}
	} else {
		// Global setting
		value, err = getGlobalConfigValue(cfg, key)
		if err != nil {
			return err
		}
	}

	fmt.Println(value)

	return nil
}

func setGlobalConfigValue(cfg *config.UserConfig, key, value string) error {
	switch key {
	case "default_region":
		cfg.DefaultRegion = value
	case "default_profile":
		cfg.DefaultProfile = value
	case "default_instance_type":
		cfg.DefaultInstanceType = value
	case "default_ebs_size":
		size, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid number: %s", value)
		}
		cfg.DefaultEBSSize = size
	case "default_ami_base":
		cfg.DefaultAMIBase = value
	case "default_subnet_type":
		if value != "public" && value != "private" {
			return fmt.Errorf("invalid subnet type: %s (must be 'public' or 'private')", value)
		}
		cfg.DefaultSubnetType = value
	case "prefer_ipv6":
		val, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean: %s", value)
		}
		cfg.PreferIPv6 = val
	case "idle_timeout":
		cfg.IdleTimeout = value
	case "auto_terminate":
		val, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean: %s", value)
		}
		cfg.AutoTerminate = val
	case "confirm_destructive":
		val, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean: %s", value)
		}
		cfg.ConfirmDestructive = val
	case "enable_cost_tracking":
		val, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean: %s", value)
		}
		cfg.EnableCostTracking = val
	case "cost_alert_threshold":
		threshold, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid number: %s", value)
		}
		cfg.CostAlertThreshold = threshold
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	return nil
}

func setAppConfigValue(appCfg *config.AppConfig, key, value string) error {
	switch key {
	case "default_environment":
		appCfg.DefaultEnvironment = value
	case "default_instance_type":
		appCfg.DefaultInstanceType = value
	case "default_ebs_size":
		size, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid number: %s", value)
		}
		appCfg.DefaultEBSSize = size
	case "port":
		port, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid port number: %s", value)
		}
		appCfg.Port = port
	default:
		return fmt.Errorf("unknown app config key: %s", key)
	}

	return nil
}

func getGlobalConfigValue(cfg *config.UserConfig, key string) (string, error) {
	switch key {
	case "default_region":
		return cfg.DefaultRegion, nil
	case "default_profile":
		return cfg.DefaultProfile, nil
	case "default_instance_type":
		return cfg.DefaultInstanceType, nil
	case "default_ebs_size":
		return strconv.Itoa(cfg.DefaultEBSSize), nil
	case "default_ami_base":
		return cfg.DefaultAMIBase, nil
	case "default_subnet_type":
		return cfg.DefaultSubnetType, nil
	case "prefer_ipv6":
		return strconv.FormatBool(cfg.PreferIPv6), nil
	case "idle_timeout":
		return cfg.IdleTimeout, nil
	case "auto_terminate":
		return strconv.FormatBool(cfg.AutoTerminate), nil
	case "confirm_destructive":
		return strconv.FormatBool(cfg.ConfirmDestructive), nil
	case "enable_cost_tracking":
		return strconv.FormatBool(cfg.EnableCostTracking), nil
	case "cost_alert_threshold":
		return fmt.Sprintf("%.2f", cfg.CostAlertThreshold), nil
	default:
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}

func getAppConfigValue(appCfg *config.AppConfig, key string) (string, error) {
	switch key {
	case "default_environment":
		return appCfg.DefaultEnvironment, nil
	case "default_instance_type":
		return appCfg.DefaultInstanceType, nil
	case "default_ebs_size":
		return strconv.Itoa(appCfg.DefaultEBSSize), nil
	case "port":
		return strconv.Itoa(appCfg.Port), nil
	default:
		return "", fmt.Errorf("unknown app config key: %s", key)
	}
}

func valueOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}
