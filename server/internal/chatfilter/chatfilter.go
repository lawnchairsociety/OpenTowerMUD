package chatfilter

import (
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// FilterMode determines how the filter handles violations
type FilterMode string

const (
	ModeReplace FilterMode = "REPLACE" // Replace banned words with asterisks
	ModeBlock   FilterMode = "BLOCK"   // Block the entire message
)

// AntispamConfig holds the anti-spam configuration
type AntispamConfig struct {
	Enabled               bool `yaml:"enabled"`
	MaxMessages           int  `yaml:"max_messages"`
	TimeWindowSeconds     int  `yaml:"time_window_seconds"`
	RepeatCooldownSeconds int  `yaml:"repeat_cooldown_seconds"`
}

// Config holds the chat filter configuration
type Config struct {
	Enabled     bool            `yaml:"enabled"`
	Mode        FilterMode      `yaml:"mode"`
	BannedWords []string        `yaml:"banned_words"`
	Antispam    *AntispamConfig `yaml:"antispam"`
}

// Result contains the outcome of filtering a message
type Result struct {
	Filtered     string   // The filtered message (with replacements if REPLACE mode)
	Violated     bool     // Whether any banned words were found
	MatchedWords []string // List of banned words that were matched
}

// ChatFilter handles word filtering for chat messages
type ChatFilter struct {
	enabled  bool
	mode     FilterMode
	patterns []*wordPattern // Pre-compiled patterns for each banned word
}

// wordPattern holds a compiled regex and the original word
type wordPattern struct {
	word    string
	pattern *regexp.Regexp
}

// New creates a new ChatFilter from a Config
func New(cfg *Config) *ChatFilter {
	if cfg == nil {
		return &ChatFilter{enabled: false}
	}

	cf := &ChatFilter{
		enabled:  cfg.Enabled,
		mode:     cfg.Mode,
		patterns: make([]*wordPattern, 0, len(cfg.BannedWords)),
	}

	// Pre-compile patterns for each banned word
	for _, word := range cfg.BannedWords {
		if word == "" {
			continue
		}
		// Create case-insensitive word boundary pattern
		// \b matches word boundaries (spaces, punctuation, start/end of string)
		pattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(word) + `\b`)
		cf.patterns = append(cf.patterns, &wordPattern{
			word:    word,
			pattern: pattern,
		})
	}

	return cf
}

// LoadConfig loads chat filter configuration from a YAML file
func LoadConfig(filepath string) (*Config, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Default to REPLACE mode if not specified
	if cfg.Mode == "" {
		cfg.Mode = ModeReplace
	}

	return &cfg, nil
}

// Check filters a message and returns the result
func (cf *ChatFilter) Check(message string) Result {
	result := Result{
		Filtered:     message,
		Violated:     false,
		MatchedWords: []string{},
	}

	// If filter is disabled, return message unchanged
	if !cf.enabled || len(cf.patterns) == 0 {
		return result
	}

	// Check each pattern against the message
	for _, wp := range cf.patterns {
		if wp.pattern.MatchString(message) {
			result.Violated = true
			result.MatchedWords = append(result.MatchedWords, wp.word)

			// In REPLACE mode, substitute the word with asterisks
			if cf.mode == ModeReplace {
				result.Filtered = wp.pattern.ReplaceAllStringFunc(result.Filtered, func(match string) string {
					return strings.Repeat("*", len(match))
				})
			}
		}
	}

	return result
}

// IsEnabled returns whether the filter is enabled
func (cf *ChatFilter) IsEnabled() bool {
	return cf.enabled
}

// Mode returns the current filter mode
func (cf *ChatFilter) Mode() FilterMode {
	return cf.mode
}

// IsBlockMode returns true if the filter is in BLOCK mode
func (cf *ChatFilter) IsBlockMode() bool {
	return cf.mode == ModeBlock
}

// IsReplaceMode returns true if the filter is in REPLACE mode
func (cf *ChatFilter) IsReplaceMode() bool {
	return cf.mode == ModeReplace
}
