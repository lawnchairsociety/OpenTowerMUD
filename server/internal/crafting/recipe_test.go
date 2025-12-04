package crafting

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewRecipeRegistry(t *testing.T) {
	r := NewRecipeRegistry()
	if r == nil {
		t.Fatal("NewRecipeRegistry returned nil")
	}
	if r.Count() != 0 {
		t.Errorf("New registry should have 0 recipes, got %d", r.Count())
	}
}

func TestRecipeRegistryLoadFromYAML(t *testing.T) {
	dataDir := findDataDir()
	if dataDir == "" {
		t.Skip("Could not find data directory")
	}

	r := NewRecipeRegistry()
	err := r.LoadFromYAML(filepath.Join(dataDir, "recipes.yaml"))
	if err != nil {
		t.Fatalf("LoadFromYAML failed: %v", err)
	}

	if r.Count() == 0 {
		t.Error("Registry should have recipes after loading")
	}
}

func TestRecipeRegistryLoadFromYAMLNotFound(t *testing.T) {
	r := NewRecipeRegistry()
	err := r.LoadFromYAML("/nonexistent/path/recipes.yaml")
	if err == nil {
		t.Error("LoadFromYAML should fail for non-existent file")
	}
}

func TestRecipeRegistryGetRecipe(t *testing.T) {
	dataDir := findDataDir()
	if dataDir == "" {
		t.Skip("Could not find data directory")
	}

	r := NewRecipeRegistry()
	if err := r.LoadFromYAML(filepath.Join(dataDir, "recipes.yaml")); err != nil {
		t.Fatalf("LoadFromYAML failed: %v", err)
	}

	// Test getting a valid recipe
	recipe := r.GetRecipe("iron_dagger")
	if recipe == nil {
		t.Error("GetRecipe('iron_dagger') returned nil")
	} else {
		if recipe.ID != "iron_dagger" {
			t.Errorf("Recipe ID = %q, want %q", recipe.ID, "iron_dagger")
		}
		if recipe.Skill != Blacksmithing {
			t.Errorf("Recipe skill = %v, want Blacksmithing", recipe.Skill)
		}
		if recipe.Station != StationForge {
			t.Errorf("Recipe station = %q, want %q", recipe.Station, StationForge)
		}
	}

	// Test getting a non-existent recipe
	if r.GetRecipe("nonexistent") != nil {
		t.Error("GetRecipe('nonexistent') should return nil")
	}
}

func TestRecipeRegistryGetRecipesBySkill(t *testing.T) {
	dataDir := findDataDir()
	if dataDir == "" {
		t.Skip("Could not find data directory")
	}

	r := NewRecipeRegistry()
	if err := r.LoadFromYAML(filepath.Join(dataDir, "recipes.yaml")); err != nil {
		t.Fatalf("LoadFromYAML failed: %v", err)
	}

	// Get blacksmithing recipes
	recipes := r.GetRecipesBySkill(Blacksmithing)
	if len(recipes) == 0 {
		t.Error("Expected at least one blacksmithing recipe")
	}

	// Verify all returned recipes are blacksmithing
	for _, recipe := range recipes {
		if recipe.Skill != Blacksmithing {
			t.Errorf("Recipe %q has skill %v, want Blacksmithing", recipe.ID, recipe.Skill)
		}
	}

	// Verify recipes are sorted by difficulty
	for i := 1; i < len(recipes); i++ {
		if recipes[i].Difficulty < recipes[i-1].Difficulty {
			t.Errorf("Recipes not sorted by difficulty: %q (%d) before %q (%d)",
				recipes[i-1].ID, recipes[i-1].Difficulty,
				recipes[i].ID, recipes[i].Difficulty)
		}
	}
}

func TestRecipeRegistryGetRecipesByStation(t *testing.T) {
	dataDir := findDataDir()
	if dataDir == "" {
		t.Skip("Could not find data directory")
	}

	r := NewRecipeRegistry()
	if err := r.LoadFromYAML(filepath.Join(dataDir, "recipes.yaml")); err != nil {
		t.Fatalf("LoadFromYAML failed: %v", err)
	}

	// Get forge recipes
	recipes := r.GetRecipesByStation(StationForge)
	if len(recipes) == 0 {
		t.Error("Expected at least one forge recipe")
	}

	// Verify all returned recipes use forge
	for _, recipe := range recipes {
		if recipe.Station != StationForge {
			t.Errorf("Recipe %q has station %q, want forge", recipe.ID, recipe.Station)
		}
	}
}

func TestRecipeRegistryFindRecipeByName(t *testing.T) {
	dataDir := findDataDir()
	if dataDir == "" {
		t.Skip("Could not find data directory")
	}

	r := NewRecipeRegistry()
	if err := r.LoadFromYAML(filepath.Join(dataDir, "recipes.yaml")); err != nil {
		t.Fatalf("LoadFromYAML failed: %v", err)
	}

	// Find by exact ID
	recipe := r.FindRecipeByName("iron_dagger")
	if recipe == nil {
		t.Error("FindRecipeByName('iron_dagger') returned nil")
	}

	// Find by partial name
	recipe = r.FindRecipeByName("dagger")
	if recipe == nil {
		t.Error("FindRecipeByName('dagger') returned nil")
	}

	// Find by case-insensitive match
	recipe = r.FindRecipeByName("IRON")
	if recipe == nil {
		t.Error("FindRecipeByName('IRON') returned nil")
	}

	// Find non-existent recipe
	recipe = r.FindRecipeByName("totally_fake_recipe")
	if recipe != nil {
		t.Error("FindRecipeByName('totally_fake_recipe') should return nil")
	}
}

func TestRecipeIngredients(t *testing.T) {
	dataDir := findDataDir()
	if dataDir == "" {
		t.Skip("Could not find data directory")
	}

	r := NewRecipeRegistry()
	if err := r.LoadFromYAML(filepath.Join(dataDir, "recipes.yaml")); err != nil {
		t.Fatalf("LoadFromYAML failed: %v", err)
	}

	recipe := r.GetRecipe("iron_dagger")
	if recipe == nil {
		t.Fatal("iron_dagger recipe not found")
	}

	if len(recipe.Ingredients) == 0 {
		t.Error("iron_dagger should have ingredients")
	}

	// Check that ingredients have valid data
	for _, ing := range recipe.Ingredients {
		if ing.ItemID == "" {
			t.Error("Ingredient ItemID should not be empty")
		}
		if ing.Quantity <= 0 {
			t.Errorf("Ingredient %q has invalid quantity: %d", ing.ItemID, ing.Quantity)
		}
	}
}

func TestRecipeOutputCount(t *testing.T) {
	dataDir := findDataDir()
	if dataDir == "" {
		t.Skip("Could not find data directory")
	}

	r := NewRecipeRegistry()
	if err := r.LoadFromYAML(filepath.Join(dataDir, "recipes.yaml")); err != nil {
		t.Fatalf("LoadFromYAML failed: %v", err)
	}

	// All recipes should have OutputCount >= 1 (defaults to 1)
	for _, recipe := range r.GetAllRecipes() {
		if recipe.OutputCount < 1 {
			t.Errorf("Recipe %q has OutputCount %d, should be at least 1", recipe.ID, recipe.OutputCount)
		}
	}
}

// findDataDir looks for the data directory
func findDataDir() string {
	candidates := []string{
		"../../data",
		"../../../data",
		"data",
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(candidate, "recipes.yaml")); err == nil {
			return candidate
		}
	}

	return ""
}
