package class

import (
	"encoding/json"
	"sort"
)

// MaxPrimaryLevel is the maximum level for a primary class
const MaxPrimaryLevel = 50

// MaxSecondaryLevel is the maximum level for secondary classes
const MaxSecondaryLevel = 25

// MinLevelForMulticlass is the minimum primary class level to unlock multiclassing
const MinLevelForMulticlass = 10

// ClassLevels tracks levels in each class for a character
type ClassLevels struct {
	levels       map[Class]int
	primaryClass Class
}

// NewClassLevels creates a new ClassLevels with a starting class at level 1
func NewClassLevels(startingClass Class) *ClassLevels {
	return &ClassLevels{
		levels:       map[Class]int{startingClass: 1},
		primaryClass: startingClass,
	}
}

// ParseClassLevels parses a JSON string into ClassLevels
func ParseClassLevels(jsonStr string, primaryClass Class) (*ClassLevels, error) {
	cl := &ClassLevels{
		levels:       make(map[Class]int),
		primaryClass: primaryClass,
	}

	if jsonStr == "" {
		cl.levels[primaryClass] = 1
		return cl, nil
	}

	// Parse JSON into map[string]int first
	var rawLevels map[string]int
	if err := json.Unmarshal([]byte(jsonStr), &rawLevels); err != nil {
		return nil, err
	}

	// Convert string keys to Class type
	for classStr, level := range rawLevels {
		c, err := ParseClass(classStr)
		if err != nil {
			continue // Skip invalid class names
		}
		cl.levels[c] = level
	}

	// Ensure primary class exists
	if _, exists := cl.levels[primaryClass]; !exists {
		cl.levels[primaryClass] = 1
	}

	return cl, nil
}

// ToJSON converts ClassLevels to a JSON string for persistence
func (cl *ClassLevels) ToJSON() string {
	// Convert to map[string]int for JSON
	rawLevels := make(map[string]int)
	for c, level := range cl.levels {
		rawLevels[string(c)] = level
	}

	data, err := json.Marshal(rawLevels)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// GetLevel returns the level in a specific class
func (cl *ClassLevels) GetLevel(c Class) int {
	return cl.levels[c]
}

// GetPrimaryClass returns the primary class
func (cl *ClassLevels) GetPrimaryClass() Class {
	return cl.primaryClass
}

// SetPrimaryClass sets the primary class (used when loading from DB)
func (cl *ClassLevels) SetPrimaryClass(c Class) {
	cl.primaryClass = c
}

// GetAllLevels returns a copy of all class levels
func (cl *ClassLevels) GetAllLevels() map[Class]int {
	result := make(map[Class]int)
	for c, level := range cl.levels {
		result[c] = level
	}
	return result
}

// GetTotalLevel returns the sum of all class levels
func (cl *ClassLevels) GetTotalLevel() int {
	total := 0
	for _, level := range cl.levels {
		total += level
	}
	return total
}

// GetEffectiveLevel returns the highest class level (used for scaling)
func (cl *ClassLevels) GetEffectiveLevel() int {
	highest := 0
	for _, level := range cl.levels {
		if level > highest {
			highest = level
		}
	}
	return highest
}

// HasClass returns true if the character has at least 1 level in a class
func (cl *ClassLevels) HasClass(c Class) bool {
	return cl.levels[c] > 0
}

// GetClasses returns all classes the character has levels in, sorted by level descending
func (cl *ClassLevels) GetClasses() []Class {
	var classes []Class
	for c, level := range cl.levels {
		if level > 0 {
			classes = append(classes, c)
		}
	}
	// Sort by level descending
	sort.Slice(classes, func(i, j int) bool {
		return cl.levels[classes[i]] > cl.levels[classes[j]]
	})
	return classes
}

// CanGainLevel checks if a class can gain another level
func (cl *ClassLevels) CanGainLevel(c Class) bool {
	currentLevel := cl.levels[c]

	if c == cl.primaryClass {
		return currentLevel < MaxPrimaryLevel
	}
	return currentLevel < MaxSecondaryLevel
}

// GainLevel adds a level to a class
// Returns the new level, or 0 if at cap
func (cl *ClassLevels) GainLevel(c Class) int {
	if !cl.CanGainLevel(c) {
		return 0
	}
	cl.levels[c]++
	return cl.levels[c]
}

// SetLevel sets the level for a class directly (used when loading from DB)
func (cl *ClassLevels) SetLevel(c Class, level int) {
	cl.levels[c] = level
}

// CanMulticlass returns true if the character can start multiclassing
func (cl *ClassLevels) CanMulticlass() bool {
	return cl.levels[cl.primaryClass] >= MinLevelForMulticlass
}

// AddClass adds a new class at level 1 (for multiclassing)
func (cl *ClassLevels) AddClass(c Class) bool {
	if cl.HasClass(c) {
		return false // Already has this class
	}
	cl.levels[c] = 1
	return true
}
