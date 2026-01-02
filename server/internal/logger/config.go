package logger

import (
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config holds logging configuration
type Config struct {
	Level           string `yaml:"level"`
	ConsoleEnabled  bool   `yaml:"console_enabled"`
	ConsoleFormat   string `yaml:"console_format"`
	FileEnabled     bool   `yaml:"file_enabled"`
	FilePath        string `yaml:"file_path"`
	FileFormat      string `yaml:"file_format"`
	FileMaxSizeMB   int    `yaml:"file_max_size_mb"`
	FileMaxBackups  int    `yaml:"file_max_backups"`
	FileMaxAgeDays  int    `yaml:"file_max_age_days"`
	AxiomEnabled    bool   `yaml:"axiom_enabled"`
	AxiomDataset    string `yaml:"axiom_dataset"`
	AxiomToken      string `yaml:"axiom_token"`
	AxiomService    string `yaml:"axiom_service"`
}

// LoggingConfig wraps the Config for YAML parsing
type LoggingConfig struct {
	Logging Config `yaml:"logging"`
}

// LoadConfig loads logging configuration from a YAML file
// and applies environment variable overrides
func LoadConfig(configPath string) (Config, error) {
	// Default configuration
	config := Config{
		Level:           "INFO",
		ConsoleEnabled:  true,
		ConsoleFormat:   "text",
		FileEnabled:     false,
		FilePath:        "logs/server.log",
		FileFormat:      "text",
		FileMaxSizeMB:   10,
		FileMaxBackups:  5,
		FileMaxAgeDays:  30,
	}

	// Try to load from file if it exists
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err == nil {
			var loggingConfig LoggingConfig
			if err := yaml.Unmarshal(data, &loggingConfig); err == nil {
				// Merge loaded config with defaults
				if loggingConfig.Logging.Level != "" {
					config.Level = loggingConfig.Logging.Level
				}
				// Only override bool if explicitly set in YAML
				config.ConsoleEnabled = loggingConfig.Logging.ConsoleEnabled
				if loggingConfig.Logging.ConsoleFormat != "" {
					config.ConsoleFormat = loggingConfig.Logging.ConsoleFormat
				}
				config.FileEnabled = loggingConfig.Logging.FileEnabled
				if loggingConfig.Logging.FilePath != "" {
					config.FilePath = loggingConfig.Logging.FilePath
				}
				if loggingConfig.Logging.FileFormat != "" {
					config.FileFormat = loggingConfig.Logging.FileFormat
				}
				if loggingConfig.Logging.FileMaxSizeMB > 0 {
					config.FileMaxSizeMB = loggingConfig.Logging.FileMaxSizeMB
				}
				if loggingConfig.Logging.FileMaxBackups > 0 {
					config.FileMaxBackups = loggingConfig.Logging.FileMaxBackups
				}
				if loggingConfig.Logging.FileMaxAgeDays > 0 {
					config.FileMaxAgeDays = loggingConfig.Logging.FileMaxAgeDays
				}
				config.AxiomEnabled = loggingConfig.Logging.AxiomEnabled
				if loggingConfig.Logging.AxiomDataset != "" {
					config.AxiomDataset = loggingConfig.Logging.AxiomDataset
				}
				if loggingConfig.Logging.AxiomToken != "" {
					config.AxiomToken = loggingConfig.Logging.AxiomToken
				}
				if loggingConfig.Logging.AxiomService != "" {
					config.AxiomService = loggingConfig.Logging.AxiomService
				}
			}
		}
		// Silently use defaults if file doesn't exist or can't be parsed
	}

	// Apply environment variable overrides
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		config.Level = logLevel
	}

	if consoleFormat := os.Getenv("LOG_CONSOLE_FORMAT"); consoleFormat != "" {
		config.ConsoleFormat = consoleFormat
	}

	if fileEnabled := os.Getenv("LOG_FILE_ENABLED"); fileEnabled != "" {
		if enabled, err := strconv.ParseBool(fileEnabled); err == nil {
			config.FileEnabled = enabled
		}
	}

	if filePath := os.Getenv("LOG_FILE_PATH"); filePath != "" {
		config.FilePath = filePath
	}

	if axiomEnabled := os.Getenv("AXIOM_ENABLED"); axiomEnabled != "" {
		if enabled, err := strconv.ParseBool(axiomEnabled); err == nil {
			config.AxiomEnabled = enabled
		}
	}

	if axiomDataset := os.Getenv("AXIOM_DATASET"); axiomDataset != "" {
		config.AxiomDataset = axiomDataset
	}

	if axiomToken := os.Getenv("AXIOM_TOKEN"); axiomToken != "" {
		config.AxiomToken = axiomToken
	}

	if axiomService := os.Getenv("AXIOM_SERVICE"); axiomService != "" {
		config.AxiomService = axiomService
	}

	return config, nil
}
