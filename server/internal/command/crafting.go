package command

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/lawnchairsociety/opentowermud/server/internal/crafting"
	"github.com/lawnchairsociety/opentowermud/server/internal/npc"
)

// executeCraft handles the craft command
// Usage:
//
//	craft              - show available recipes at current station
//	craft <recipe>     - attempt to craft a recipe
//	craft info <recipe> - show recipe details
func executeCraft(c *Command, p PlayerInterface) string {
	room := p.GetCurrentRoom().(RoomInterface)
	args := strings.TrimSpace(strings.Join(c.Args, " "))

	// Find what station is in this room
	station := getStationInRoom(room)
	if station == "" {
		return "There is no crafting station here. Look for a forge, workbench, alchemy lab, or enchanting table."
	}

	// Get the recipe registry from the server
	serverIface := p.GetServer()
	server, ok := serverIface.(ServerInterface)
	if !ok {
		return "Error: Unable to access server."
	}

	recipes := server.GetRecipeRegistry()
	if recipes == nil {
		return "Error: Crafting system not initialized."
	}

	stationName := crafting.StationName(station)

	// If no args, show available recipes at this station
	if args == "" {
		return showStationRecipes(p, recipes, station, stationName)
	}

	// Check for 'info' subcommand
	if strings.HasPrefix(strings.ToLower(args), "info ") {
		recipeName := strings.TrimSpace(args[5:])
		return showRecipeInfo(p, recipes, recipeName)
	}

	// Otherwise, try to craft the named recipe
	return attemptCraft(p, recipes, args, station, stationName, server)
}

// getStationInRoom returns the crafting station type in the room, or empty string if none
func getStationInRoom(room RoomInterface) string {
	features := room.GetFeatures()
	for _, f := range features {
		switch f {
		case crafting.StationForge, crafting.StationWorkbench, crafting.StationAlchemyLab, crafting.StationEnchantingTable:
			return f
		}
	}
	return ""
}

// showStationRecipes shows all known recipes for the current station
func showStationRecipes(p PlayerInterface, recipes *crafting.RecipeRegistry, station, stationName string) string {
	stationRecipes := recipes.GetRecipesByStation(station)
	if len(stationRecipes) == 0 {
		return fmt.Sprintf("No recipes are available at the %s.", stationName)
	}

	// Get the skill for this station
	skill, _ := crafting.GetSkillForStation(station)
	skillLevel := p.GetCraftingSkill(skill)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n=== %s Recipes ===\n", stationName))
	sb.WriteString(fmt.Sprintf("Your %s skill: %d\n\n", skill.String(), skillLevel))

	knownRecipes := 0
	for _, recipe := range stationRecipes {
		if !p.KnowsRecipe(recipe.ID) {
			continue
		}
		knownRecipes++

		// Show if player can craft it (level/skill requirements)
		canCraft, reason := canCraftRecipe(p, recipe)
		status := "[OK]"
		if !canCraft {
			status = fmt.Sprintf("[%s]", reason)
		}

		sb.WriteString(fmt.Sprintf("  %s - %s (Difficulty: %d) %s\n",
			recipe.ID, recipe.Name, recipe.Difficulty, status))
	}

	if knownRecipes == 0 {
		sb.WriteString("  You haven't learned any recipes for this station yet.\n")
		sb.WriteString("  Find a crafting trainer to learn new recipes!\n")
	}

	sb.WriteString("\nUse 'craft <recipe>' to craft an item, or 'craft info <recipe>' for details.")
	return sb.String()
}

// showRecipeInfo shows detailed information about a recipe
func showRecipeInfo(p PlayerInterface, recipes *crafting.RecipeRegistry, recipeName string) string {
	recipe := recipes.FindRecipeByName(recipeName)
	if recipe == nil {
		return fmt.Sprintf("Unknown recipe: %s", recipeName)
	}

	if !p.KnowsRecipe(recipe.ID) {
		return fmt.Sprintf("You haven't learned the recipe for %s yet.", recipe.Name)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n=== %s ===\n", recipe.Name))
	if recipe.Description != "" {
		sb.WriteString(fmt.Sprintf("%s\n", recipe.Description))
	}
	sb.WriteString(fmt.Sprintf("\nSkill: %s (requires %d)\n", recipe.Skill.String(), recipe.SkillRequired))
	sb.WriteString(fmt.Sprintf("Difficulty: %d\n", recipe.Difficulty))
	sb.WriteString(fmt.Sprintf("Station: %s\n", crafting.StationName(recipe.Station)))
	sb.WriteString(fmt.Sprintf("Player Level Required: %d\n", recipe.LevelRequired))
	sb.WriteString(fmt.Sprintf("Skill Points on Success: +%d\n", recipe.SkillGain))

	sb.WriteString("\nIngredients:\n")
	for _, ing := range recipe.Ingredients {
		sb.WriteString(fmt.Sprintf("  - %s x%d\n", ing.ItemID, ing.Quantity))
	}

	sb.WriteString(fmt.Sprintf("\nProduces: %s", recipe.OutputItem))
	if recipe.OutputCount > 1 {
		sb.WriteString(fmt.Sprintf(" x%d", recipe.OutputCount))
	}
	sb.WriteString("\n")

	// Show if player can craft it
	canCraft, reason := canCraftRecipe(p, recipe)
	if canCraft {
		sb.WriteString("\n[You meet all requirements to craft this]")
	} else {
		sb.WriteString(fmt.Sprintf("\n[Cannot craft: %s]", reason))
	}

	return sb.String()
}

// canCraftRecipe checks if a player can attempt to craft a recipe
func canCraftRecipe(p PlayerInterface, recipe *crafting.Recipe) (bool, string) {
	// Check player level
	if p.GetLevel() < recipe.LevelRequired {
		return false, fmt.Sprintf("requires level %d", recipe.LevelRequired)
	}

	// Check skill level
	skillLevel := p.GetCraftingSkill(recipe.Skill)
	if skillLevel < recipe.SkillRequired {
		return false, fmt.Sprintf("requires %d %s", recipe.SkillRequired, recipe.Skill.String())
	}

	return true, ""
}

// hasIngredients checks if the player has all required ingredients
func hasIngredients(p PlayerInterface, recipe *crafting.Recipe) (bool, string) {
	for _, ing := range recipe.Ingredients {
		count := p.CountItemsByID(ing.ItemID)
		if count < ing.Quantity {
			if count == 0 {
				return false, fmt.Sprintf("You don't have any %s.", ing.ItemID)
			}
			return false, fmt.Sprintf("You need %d %s but only have %d.", ing.Quantity, ing.ItemID, count)
		}
	}
	return true, ""
}

// attemptCraft tries to craft a recipe
func attemptCraft(p PlayerInterface, recipes *crafting.RecipeRegistry, recipeName, station, stationName string, server ServerInterface) string {
	recipe := recipes.FindRecipeByName(recipeName)
	if recipe == nil {
		return fmt.Sprintf("Unknown recipe: %s. Use 'craft' to see available recipes.", recipeName)
	}

	if !p.KnowsRecipe(recipe.ID) {
		return fmt.Sprintf("You haven't learned the recipe for %s. Find a crafting trainer to learn it.", recipe.Name)
	}

	// Check station type matches
	if recipe.Station != station {
		return fmt.Sprintf("%s requires a %s, not a %s.", recipe.Name, crafting.StationName(recipe.Station), stationName)
	}

	// Check requirements
	canCraft, reason := canCraftRecipe(p, recipe)
	if !canCraft {
		return fmt.Sprintf("You cannot craft %s: %s", recipe.Name, reason)
	}

	// Check ingredients
	hasIng, ingMsg := hasIngredients(p, recipe)
	if !hasIng {
		return ingMsg
	}

	// Roll the crafting check
	skillLevel := p.GetCraftingSkill(recipe.Skill)
	intMod := (p.GetIntelligence() - 10) / 2
	roll, success := rollCraftingCheck(skillLevel, intMod, recipe.Difficulty)

	// Consume ingredients
	for _, ing := range recipe.Ingredients {
		for i := 0; i < ing.Quantity; i++ {
			p.RemoveItemByID(ing.ItemID)
		}
	}

	if !success {
		// Failure - materials returned
		for _, ing := range recipe.Ingredients {
			for i := 0; i < ing.Quantity; i++ {
				item := server.CreateItem(ing.ItemID)
				if item != nil {
					p.AddItem(item)
				}
			}
		}
		return fmt.Sprintf("You attempt to craft %s but fail! (Roll: %d vs DC %d)\nYour materials are returned to you.",
			recipe.Name, roll, recipe.Difficulty)
	}

	// Success! Create the output item(s)
	var createdItems []string
	for i := 0; i < recipe.OutputCount; i++ {
		item := server.CreateItem(recipe.OutputItem)
		if item != nil {
			p.AddItem(item)
			createdItems = append(createdItems, item.Name)
		}
	}

	// Award skill points
	oldLevel := skillLevel
	newLevel := p.AddCraftingSkillPoints(recipe.Skill, recipe.SkillGain)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Success! You craft %s. (Roll: %d vs DC %d)\n",
		recipe.Name, roll, recipe.Difficulty))

	if len(createdItems) > 0 {
		sb.WriteString(fmt.Sprintf("You created: %s\n", strings.Join(createdItems, ", ")))
	}

	if newLevel > oldLevel {
		sb.WriteString(fmt.Sprintf("Your %s skill increased to %d! (+%d)",
			recipe.Skill.String(), newLevel, recipe.SkillGain))
	} else if newLevel < crafting.MaxSkillLevel {
		sb.WriteString(fmt.Sprintf("You gained %d %s skill point(s). (%d/%d)",
			recipe.SkillGain, recipe.Skill.String(), newLevel, crafting.MaxSkillLevel))
	}

	return sb.String()
}

// rollCraftingCheck performs a crafting skill check
// Formula: d20 + (Skill / 5) + (INT modifier / 2) >= Difficulty
func rollCraftingCheck(skillLevel, intMod, difficulty int) (int, bool) {
	dieRoll := rand.Intn(20) + 1 // 1-20
	skillBonus := skillLevel / 5
	intBonus := intMod / 2

	totalRoll := dieRoll + skillBonus + intBonus
	return totalRoll, totalRoll >= difficulty
}

// executeSkills shows the player's crafting skill levels
func executeSkills(c *Command, p PlayerInterface) string {
	var sb strings.Builder
	sb.WriteString("\n=== Crafting Skills ===\n")

	skills := p.GetAllCraftingSkills()
	hasAny := false

	for _, skill := range crafting.AllSkills() {
		level := skills[skill]
		station := skill.Station()
		stationName := crafting.StationName(station)

		bar := createSkillBar(level, crafting.MaxSkillLevel)
		sb.WriteString(fmt.Sprintf("  %s: %d/%d %s (Use at: %s)\n",
			skill.String(), level, crafting.MaxSkillLevel, bar, stationName))

		if level > 0 {
			hasAny = true
		}
	}

	if !hasAny {
		sb.WriteString("\nYou haven't developed any crafting skills yet.\n")
		sb.WriteString("Find crafting trainers to learn recipes and start crafting!\n")
	}

	return sb.String()
}

// createSkillBar creates a visual skill bar
func createSkillBar(current, max int) string {
	barLength := 20
	filled := (current * barLength) / max
	if current > 0 && filled == 0 {
		filled = 1
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barLength-filled)
	return "[" + bar + "]"
}

// executeLearn handles learning recipes from trainers
func executeLearn(c *Command, p PlayerInterface) string {
	room := p.GetCurrentRoom().(RoomInterface)
	args := strings.TrimSpace(strings.Join(c.Args, " "))

	// Find a crafting trainer in this room
	npcs := room.GetNPCs()
	var trainer *npc.NPC
	var trainerSkill string
	var teachableRecipes []string

	for _, n := range npcs {
		skill := n.GetCraftingTrainer()
		recipes := n.GetTeachesRecipes()
		if skill != "" && len(recipes) > 0 {
			trainer = n
			trainerSkill = skill
			teachableRecipes = recipes
			break
		}
	}

	if trainer == nil {
		return "There is no crafting trainer here. Find a trainer to learn new recipes."
	}

	// Get the recipe registry
	serverIface := p.GetServer()
	server, ok := serverIface.(ServerInterface)
	if !ok {
		return "Error: Unable to access server."
	}

	recipes := server.GetRecipeRegistry()
	if recipes == nil {
		return "Error: Crafting system not initialized."
	}

	// If no args, show what can be learned
	if args == "" {
		return showLearnableRecipes(p, trainer, trainerSkill, teachableRecipes, recipes)
	}

	// Try to learn the specified recipe
	return learnRecipe(p, trainer, trainerSkill, teachableRecipes, recipes, args)
}

// showLearnableRecipes shows what recipes a trainer can teach
func showLearnableRecipes(p PlayerInterface, trainer *npc.NPC, trainerSkill string, teachableRecipes []string, recipes *crafting.RecipeRegistry) string {
	skill, _ := crafting.ParseSkill(trainerSkill)
	skillLevel := p.GetCraftingSkill(skill)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n=== %s's Recipes ===\n", trainer.GetName()))
	sb.WriteString(fmt.Sprintf("Teaching: %s (Your level: %d)\n\n", skill.String(), skillLevel))

	learned := 0
	available := 0
	tooAdvanced := 0

	for _, recipeID := range teachableRecipes {
		recipe := recipes.GetRecipe(recipeID)
		if recipe == nil {
			continue
		}

		if p.KnowsRecipe(recipeID) {
			sb.WriteString(fmt.Sprintf("  [KNOWN] %s\n", recipe.Name))
			learned++
		} else if skillLevel >= recipe.SkillRequired && p.GetLevel() >= recipe.LevelRequired {
			sb.WriteString(fmt.Sprintf("  %s (Skill req: %d, Level req: %d)\n",
				recipe.Name, recipe.SkillRequired, recipe.LevelRequired))
			available++
		} else {
			reason := ""
			if p.GetLevel() < recipe.LevelRequired {
				reason = fmt.Sprintf("Level %d required", recipe.LevelRequired)
			} else {
				reason = fmt.Sprintf("%d %s required", recipe.SkillRequired, skill.String())
			}
			sb.WriteString(fmt.Sprintf("  [LOCKED] %s (%s)\n", recipe.Name, reason))
			tooAdvanced++
		}
	}

	sb.WriteString(fmt.Sprintf("\nLearned: %d | Available: %d | Locked: %d\n", learned, available, tooAdvanced))
	if available > 0 {
		sb.WriteString("Use 'learn <recipe name>' to learn a recipe.")
	}

	return sb.String()
}

// learnRecipe attempts to learn a recipe from a trainer
func learnRecipe(p PlayerInterface, trainer *npc.NPC, trainerSkill string, teachableRecipes []string, allRecipes *crafting.RecipeRegistry, recipeName string) string {
	// Find the recipe by name
	var recipe *crafting.Recipe
	for _, recipeID := range teachableRecipes {
		r := allRecipes.GetRecipe(recipeID)
		if r == nil {
			continue
		}
		if strings.EqualFold(r.ID, recipeName) || strings.EqualFold(r.Name, recipeName) ||
			strings.Contains(strings.ToLower(r.Name), strings.ToLower(recipeName)) {
			recipe = r
			break
		}
	}

	if recipe == nil {
		return fmt.Sprintf("%s doesn't teach that recipe. Use 'learn' to see available recipes.", trainer.GetName())
	}

	// Check if already known
	if p.KnowsRecipe(recipe.ID) {
		return fmt.Sprintf("You already know how to craft %s.", recipe.Name)
	}

	// Check level requirement
	if p.GetLevel() < recipe.LevelRequired {
		return fmt.Sprintf("You need to be level %d to learn %s.", recipe.LevelRequired, recipe.Name)
	}

	// Check skill requirement
	skill, _ := crafting.ParseSkill(trainerSkill)
	skillLevel := p.GetCraftingSkill(skill)
	if skillLevel < recipe.SkillRequired {
		return fmt.Sprintf("You need %d %s skill to learn %s. (You have %d)",
			recipe.SkillRequired, skill.String(), recipe.Name, skillLevel)
	}

	// Learn the recipe!
	p.LearnRecipe(recipe.ID)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s teaches you the recipe for %s!\n", trainer.GetName(), recipe.Name))
	sb.WriteString(fmt.Sprintf("You can now craft this at a %s.", crafting.StationName(recipe.Station)))

	return sb.String()
}
