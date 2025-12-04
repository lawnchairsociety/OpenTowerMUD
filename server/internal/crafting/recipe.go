package crafting

import (
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

// RecipeIngredient represents a required ingredient for a recipe
type RecipeIngredient struct {
	ItemID   string `yaml:"item"`
	Quantity int    `yaml:"quantity"`
}

// Recipe represents a crafting recipe
type Recipe struct {
	ID            string             `yaml:"-"`
	Name          string             `yaml:"name"`
	Description   string             `yaml:"description"`
	Skill         CraftingSkill      `yaml:"skill"`
	Difficulty    int                `yaml:"difficulty"`    // DC for skill check
	SkillGain     int                `yaml:"skill_gain"`    // Points gained on success
	Station       string             `yaml:"station"`       // Required station type
	LevelRequired int                `yaml:"level_required"` // Player level minimum
	SkillRequired int                `yaml:"skill_required"` // Skill level minimum (0-100)
	Ingredients   []RecipeIngredient `yaml:"ingredients"`
	OutputItem    string             `yaml:"output"`       // Item ID produced
	OutputCount   int                `yaml:"output_count"` // Number produced (defaults to 1)
}

// recipesFile represents the YAML structure
type recipesFile struct {
	Recipes map[string]*Recipe `yaml:"recipes"`
}

// RecipeRegistry manages all crafting recipes
type RecipeRegistry struct {
	recipes map[string]*Recipe
}

// NewRecipeRegistry creates an empty recipe registry
func NewRecipeRegistry() *RecipeRegistry {
	return &RecipeRegistry{
		recipes: make(map[string]*Recipe),
	}
}

// LoadFromYAML loads recipes from a YAML file
func (r *RecipeRegistry) LoadFromYAML(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read recipes file: %w", err)
	}

	var file recipesFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("failed to parse recipes YAML: %w", err)
	}

	for id, recipe := range file.Recipes {
		recipe.ID = id
		// Default output count to 1 if not specified
		if recipe.OutputCount == 0 {
			recipe.OutputCount = 1
		}
		r.recipes[id] = recipe
	}

	return nil
}

// GetRecipe returns a recipe by ID
func (r *RecipeRegistry) GetRecipe(id string) *Recipe {
	return r.recipes[id]
}

// GetRecipesBySkill returns all recipes for a given skill, sorted by difficulty
func (r *RecipeRegistry) GetRecipesBySkill(skill CraftingSkill) []*Recipe {
	var recipes []*Recipe
	for _, recipe := range r.recipes {
		if recipe.Skill == skill {
			recipes = append(recipes, recipe)
		}
	}
	sort.Slice(recipes, func(i, j int) bool {
		return recipes[i].Difficulty < recipes[j].Difficulty
	})
	return recipes
}

// GetRecipesByStation returns all recipes for a given station, sorted by difficulty
func (r *RecipeRegistry) GetRecipesByStation(station string) []*Recipe {
	var recipes []*Recipe
	for _, recipe := range r.recipes {
		if recipe.Station == station {
			recipes = append(recipes, recipe)
		}
	}
	sort.Slice(recipes, func(i, j int) bool {
		return recipes[i].Difficulty < recipes[j].Difficulty
	})
	return recipes
}

// GetAllRecipes returns all recipes sorted by skill then difficulty
func (r *RecipeRegistry) GetAllRecipes() []*Recipe {
	var recipes []*Recipe
	for _, recipe := range r.recipes {
		recipes = append(recipes, recipe)
	}
	sort.Slice(recipes, func(i, j int) bool {
		if recipes[i].Skill != recipes[j].Skill {
			return recipes[i].Skill < recipes[j].Skill
		}
		return recipes[i].Difficulty < recipes[j].Difficulty
	})
	return recipes
}

// Count returns the total number of recipes
func (r *RecipeRegistry) Count() int {
	return len(r.recipes)
}

// GetRecipesByIDs returns recipes matching the given IDs
func (r *RecipeRegistry) GetRecipesByIDs(ids []string) []*Recipe {
	var recipes []*Recipe
	for _, id := range ids {
		if recipe := r.recipes[id]; recipe != nil {
			recipes = append(recipes, recipe)
		}
	}
	return recipes
}

// FindRecipeByName searches for a recipe by name (case-insensitive partial match)
func (r *RecipeRegistry) FindRecipeByName(name string) *Recipe {
	// First try exact ID match
	if recipe := r.recipes[name]; recipe != nil {
		return recipe
	}

	// Then try partial name match
	for _, recipe := range r.recipes {
		if containsIgnoreCase(recipe.Name, name) || containsIgnoreCase(recipe.ID, name) {
			return recipe
		}
	}
	return nil
}

// containsIgnoreCase checks if s contains substr (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
		 len(substr) > 0 &&
		 (toLower(s) == toLower(substr) ||
		  contains(toLower(s), toLower(substr))))
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
