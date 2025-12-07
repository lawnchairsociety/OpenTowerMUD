package namefilter

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the name filter configuration
type Config struct {
	Enabled     bool     `yaml:"enabled"`
	BannedWords []string `yaml:"banned_words"`
	BannedNames []string `yaml:"banned_names"`
}

// Result contains the outcome of checking a name
type Result struct {
	Allowed bool   // Whether the name is allowed
	Reason  string // Reason for rejection (if not allowed)
}

// NameFilter handles name validation against banned words and names
type NameFilter struct {
	enabled     bool
	bannedWords []string // Lowercase banned words (partial match)
	bannedNames []string // Lowercase banned names (exact match)
}

// New creates a new NameFilter from a Config
func New(cfg *Config) *NameFilter {
	if cfg == nil {
		return &NameFilter{enabled: false}
	}

	nf := &NameFilter{
		enabled:     cfg.Enabled,
		bannedWords: make([]string, 0, len(cfg.BannedWords)),
		bannedNames: make([]string, 0, len(cfg.BannedNames)),
	}

	// Store lowercase versions for case-insensitive matching
	for _, word := range cfg.BannedWords {
		if word != "" {
			nf.bannedWords = append(nf.bannedWords, strings.ToLower(word))
		}
	}

	for _, name := range cfg.BannedNames {
		if name != "" {
			nf.bannedNames = append(nf.bannedNames, strings.ToLower(name))
		}
	}

	return nf
}

// LoadConfig loads name filter configuration from a YAML file
func LoadConfig(filepath string) (*Config, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Check validates a name against the filter rules
func (nf *NameFilter) Check(name string) Result {
	// If filter is disabled, allow all names
	if !nf.enabled {
		return Result{Allowed: true}
	}

	nameLower := strings.ToLower(name)

	// Check for exact banned names
	for _, banned := range nf.bannedNames {
		if nameLower == banned {
			return Result{
				Allowed: false,
				Reason:  "That name is not allowed.",
			}
		}
	}

	// Check for banned words (partial match)
	for _, word := range nf.bannedWords {
		if strings.Contains(nameLower, word) {
			return Result{
				Allowed: false,
				Reason:  "That name contains a word that is not allowed.",
			}
		}
	}

	return Result{Allowed: true}
}

// IsEnabled returns whether the filter is enabled
func (nf *NameFilter) IsEnabled() bool {
	return nf.enabled
}
