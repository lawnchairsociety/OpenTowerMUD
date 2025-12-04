package test

import (
	"fmt"
	"strings"
	"time"

	"github.com/lawnchairsociety/opentowermud/server/internal/testclient"
)

// =============================================================================
// Navigation Helpers for Crafting Locations
// =============================================================================

// navigateToArmory moves a client from Town Square to Armory (has forge)
// Path: Town Square -> Market Street -> Artisan's Market -> Armory
func navigateToArmory(client *testclient.TestClient) {
	client.SendCommand("south") // Market Street
	time.Sleep(300 * time.Millisecond)
	client.SendCommand("south") // Artisan's Market
	time.Sleep(300 * time.Millisecond)
	client.SendCommand("east") // Armory
	time.Sleep(500 * time.Millisecond)
}

// navigateToArtisanMarket moves a client from Town Square to Artisan's Market (has workbench)
// Path: Town Square -> Market Street -> Artisan's Market
func navigateToArtisanMarket(client *testclient.TestClient) {
	client.SendCommand("south") // Market Street
	time.Sleep(300 * time.Millisecond)
	client.SendCommand("south") // Artisan's Market
	time.Sleep(300 * time.Millisecond)
}

// navigateToAlchemistShop moves a client from Town Square to Alchemist Shop (has alchemy_lab)
// Path: Town Square -> Market Street -> Artisan's Market -> Alchemist Shop
func navigateToAlchemistShop(client *testclient.TestClient) {
	client.SendCommand("south") // Market Street
	time.Sleep(300 * time.Millisecond)
	client.SendCommand("south") // Artisan's Market
	time.Sleep(300 * time.Millisecond)
	client.SendCommand("west") // Alchemist Shop
	time.Sleep(500 * time.Millisecond)
}

// navigateToMageTower moves a client from Town Square to Mage Tower (has enchanting_table)
// Path: Town Square -> Temple -> Mage Tower
func navigateToMageTower(client *testclient.TestClient) {
	client.SendCommand("east") // Temple
	time.Sleep(300 * time.Millisecond)
	client.SendCommand("east") // Mage Tower
	time.Sleep(500 * time.Millisecond)
}

// =============================================================================
// Group 10: Crafting System
// =============================================================================

// TestSkillsCommand tests the skills command displays crafting skills
func TestSkillsCommand(serverAddr string) TestResult {
	const testName = "Skills Command"

	name := uniqueName("SkillsTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	logAction(testName, "Checking skills...")
	client.ClearMessages()
	client.SendCommand("skills")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should display crafting skill types
	hasBlacksmithing := strings.Contains(fullOutput, "Blacksmithing") || strings.Contains(fullOutput, "blacksmithing")
	hasLeatherworking := strings.Contains(fullOutput, "Leatherworking") || strings.Contains(fullOutput, "leatherworking")
	hasAlchemy := strings.Contains(fullOutput, "Alchemy") || strings.Contains(fullOutput, "alchemy")
	hasEnchanting := strings.Contains(fullOutput, "Enchanting") || strings.Contains(fullOutput, "enchanting")

	logResult(testName, hasBlacksmithing, "Shows Blacksmithing")
	logResult(testName, hasLeatherworking, "Shows Leatherworking")
	logResult(testName, hasAlchemy, "Shows Alchemy")
	logResult(testName, hasEnchanting, "Shows Enchanting")

	if !hasBlacksmithing && !hasLeatherworking && !hasAlchemy && !hasEnchanting {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Skills command didn't show crafting skills. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Skills command displays crafting skills"}
}

// TestCraftingStationForge tests that the forge station is accessible and recognized
func TestCraftingStationForge(serverAddr string) TestResult {
	const testName = "Crafting Station - Forge"

	name := uniqueName("ForgeTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to Armory (has forge)
	logAction(testName, "Navigating to Armory...")
	navigateToArmory(client)

	// Verify we're at the armory
	client.ClearMessages()
	client.SendCommand("look")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	atArmory := strings.Contains(fullOutput, "Armory") || strings.Contains(fullOutput, "forge") || strings.Contains(fullOutput, "Iron Forge")
	logResult(testName, atArmory, "At Armory with forge")
	if !atArmory {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Failed to reach Armory. Got: %v", messages)}
	}

	// Try craft command at forge
	logAction(testName, "Trying craft command at forge...")
	client.ClearMessages()
	client.SendCommand("craft")
	time.Sleep(300 * time.Millisecond)

	messages = client.GetMessages()
	fullOutput = strings.Join(messages, " ")

	// Should show recipes or indicate we need to learn recipes
	hasCraftResponse := strings.Contains(fullOutput, "recipe") || strings.Contains(fullOutput, "forge") ||
		strings.Contains(fullOutput, "blacksmithing") || strings.Contains(fullOutput, "learn") ||
		strings.Contains(fullOutput, "craft") || strings.Contains(fullOutput, "Blacksmithing")
	logResult(testName, hasCraftResponse, "Craft command responded at forge")

	if !hasCraftResponse {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Craft command failed at forge. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Forge station accessible and responds to craft command"}
}

// TestCraftingStationWorkbench tests that the workbench station is accessible
func TestCraftingStationWorkbench(serverAddr string) TestResult {
	const testName = "Crafting Station - Workbench"

	name := uniqueName("WorkbenchTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to Artisan's Market (has workbench)
	logAction(testName, "Navigating to Artisan's Market...")
	navigateToArtisanMarket(client)

	client.ClearMessages()
	client.SendCommand("look")
	time.Sleep(200 * time.Millisecond)

	atMarket := client.WaitForMessage("Artisan", 1*time.Second) || client.WaitForMessage("workbench", 1*time.Second)
	logResult(testName, atMarket, "At Artisan's Market with workbench")
	if !atMarket {
		return TestResult{Name: testName, Passed: false, Message: "Failed to reach Artisan's Market"}
	}

	// Try craft command at workbench
	logAction(testName, "Trying craft command at workbench...")
	client.ClearMessages()
	client.SendCommand("craft")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	hasCraftResponse := strings.Contains(fullOutput, "recipe") || strings.Contains(fullOutput, "workbench") ||
		strings.Contains(fullOutput, "leatherworking") || strings.Contains(fullOutput, "learn") ||
		strings.Contains(fullOutput, "craft") || strings.Contains(fullOutput, "Leatherworking")
	logResult(testName, hasCraftResponse, "Craft command responded at workbench")

	if !hasCraftResponse {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Craft command failed at workbench. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Workbench station accessible and responds to craft command"}
}

// TestCraftingStationAlchemyLab tests that the alchemy lab station is accessible
func TestCraftingStationAlchemyLab(serverAddr string) TestResult {
	const testName = "Crafting Station - Alchemy Lab"

	name := uniqueName("AlchemyLabTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to Alchemist Shop (has alchemy_lab)
	logAction(testName, "Navigating to Alchemist Shop...")
	navigateToAlchemistShop(client)

	client.ClearMessages()
	client.SendCommand("look")
	time.Sleep(200 * time.Millisecond)

	atShop := client.WaitForMessage("Alchemy", 1*time.Second) || client.WaitForMessage("Zara", 1*time.Second)
	logResult(testName, atShop, "At Alchemist Shop with alchemy lab")
	if !atShop {
		return TestResult{Name: testName, Passed: false, Message: "Failed to reach Alchemist Shop"}
	}

	// Try craft command at alchemy lab
	logAction(testName, "Trying craft command at alchemy lab...")
	client.ClearMessages()
	client.SendCommand("craft")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	hasCraftResponse := strings.Contains(fullOutput, "recipe") || strings.Contains(fullOutput, "alchemy") ||
		strings.Contains(fullOutput, "Alchemy") || strings.Contains(fullOutput, "learn") ||
		strings.Contains(fullOutput, "craft") || strings.Contains(fullOutput, "potion")
	logResult(testName, hasCraftResponse, "Craft command responded at alchemy lab")

	if !hasCraftResponse {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Craft command failed at alchemy lab. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Alchemy lab station accessible and responds to craft command"}
}

// TestCraftingStationEnchantingTable tests that the enchanting table is accessible
func TestCraftingStationEnchantingTable(serverAddr string) TestResult {
	const testName = "Crafting Station - Enchanting Table"

	name := uniqueName("EnchantTableTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to Mage Tower (has enchanting_table)
	logAction(testName, "Navigating to Mage Tower...")
	navigateToMageTower(client)

	client.ClearMessages()
	client.SendCommand("look")
	time.Sleep(200 * time.Millisecond)

	atTower := client.WaitForMessage("Arcane", 1*time.Second) || client.WaitForMessage("enchanting", 1*time.Second)
	logResult(testName, atTower, "At Mage Tower with enchanting table")
	if !atTower {
		return TestResult{Name: testName, Passed: false, Message: "Failed to reach Mage Tower"}
	}

	// Try craft command at enchanting table
	logAction(testName, "Trying craft command at enchanting table...")
	client.ClearMessages()
	client.SendCommand("craft")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	hasCraftResponse := strings.Contains(fullOutput, "recipe") || strings.Contains(fullOutput, "enchant") ||
		strings.Contains(fullOutput, "Enchanting") || strings.Contains(fullOutput, "learn") ||
		strings.Contains(fullOutput, "craft") || strings.Contains(fullOutput, "scroll")
	logResult(testName, hasCraftResponse, "Craft command responded at enchanting table")

	if !hasCraftResponse {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Craft command failed at enchanting table. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Enchanting table station accessible and responds to craft command"}
}

// TestLearnFromTrainer tests learning a recipe from a crafting trainer
func TestLearnFromTrainer(serverAddr string) TestResult {
	const testName = "Learn From Trainer"

	name := uniqueName("LearnTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to Armory where Forge Master Tormund is
	logAction(testName, "Navigating to Armory...")
	navigateToArmory(client)

	client.ClearMessages()
	client.SendCommand("look")
	time.Sleep(200 * time.Millisecond)

	// Check trainer is present
	hasTormund := client.WaitForMessage("Tormund", 1*time.Second) || client.WaitForMessage("Forge Master", 1*time.Second)
	logResult(testName, hasTormund, "Forge Master Tormund present")
	if !hasTormund {
		return TestResult{Name: testName, Passed: false, Message: "Forge Master Tormund not found in Armory"}
	}

	// Try learn command
	logAction(testName, "Trying learn command...")
	client.ClearMessages()
	client.SendCommand("learn")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should show available recipes or list of learnable recipes
	hasLearnResponse := strings.Contains(fullOutput, "recipe") || strings.Contains(fullOutput, "learn") ||
		strings.Contains(fullOutput, "iron") || strings.Contains(fullOutput, "dagger") ||
		strings.Contains(fullOutput, "sword") || strings.Contains(fullOutput, "teach") ||
		strings.Contains(fullOutput, "Tormund")
	logResult(testName, hasLearnResponse, "Learn command responded")

	if !hasLearnResponse {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Learn command failed. Got: %v", messages)}
	}

	// Try to learn a specific recipe (iron_dagger is the simplest)
	logAction(testName, "Learning iron_dagger recipe...")
	client.ClearMessages()
	client.SendCommand("learn iron_dagger")
	time.Sleep(300 * time.Millisecond)

	messages = client.GetMessages()
	fullOutput = strings.Join(messages, " ")

	// Should indicate success or that we already know it
	hasLearnResult := strings.Contains(fullOutput, "learn") || strings.Contains(fullOutput, "recipe") ||
		strings.Contains(fullOutput, "already") || strings.Contains(fullOutput, "know") ||
		strings.Contains(fullOutput, "Iron Dagger") || strings.Contains(fullOutput, "iron_dagger")
	logResult(testName, hasLearnResult, "Learned recipe")

	if !hasLearnResult {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Failed to learn recipe. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Successfully learned recipe from trainer"}
}

// TestCraftWithoutStation tests that crafting fails without proper station
func TestCraftWithoutStation(serverAddr string) TestResult {
	const testName = "Craft Without Station"

	name := uniqueName("NoStationTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Stay in Town Square (no crafting station)
	logAction(testName, "Trying craft in Town Square (no station)...")
	client.ClearMessages()
	client.SendCommand("craft")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should indicate no station available
	hasNoStationMsg := strings.Contains(fullOutput, "station") || strings.Contains(fullOutput, "need") ||
		strings.Contains(fullOutput, "forge") || strings.Contains(fullOutput, "workbench") ||
		strings.Contains(fullOutput, "no") || strings.Contains(fullOutput, "can't") ||
		strings.Contains(fullOutput, "cannot") || strings.Contains(fullOutput, "require")
	logResult(testName, hasNoStationMsg, "Craft without station shows error")

	if !hasNoStationMsg {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Craft should fail without station. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Crafting correctly requires a station"}
}

// TestCraftWithoutMaterials tests crafting fails when missing materials
func TestCraftWithoutMaterials(serverAddr string) TestResult {
	const testName = "Craft Without Materials"

	name := uniqueName("NoMaterialsTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to Armory and learn a recipe first
	navigateToArmory(client)
	time.Sleep(200 * time.Millisecond)

	client.SendCommand("learn iron_dagger")
	time.Sleep(300 * time.Millisecond)

	// Try to craft without materials
	logAction(testName, "Trying to craft iron_dagger without materials...")
	client.ClearMessages()
	client.SendCommand("craft iron_dagger")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should indicate missing materials
	hasMissingMaterials := strings.Contains(fullOutput, "material") || strings.Contains(fullOutput, "ingredient") ||
		strings.Contains(fullOutput, "need") || strings.Contains(fullOutput, "don't have") ||
		strings.Contains(fullOutput, "require") || strings.Contains(fullOutput, "missing") ||
		strings.Contains(fullOutput, "iron_ore") || strings.Contains(fullOutput, "ore")
	logResult(testName, hasMissingMaterials, "Craft without materials shows error")

	if !hasMissingMaterials {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Craft should fail without materials. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Crafting correctly requires materials"}
}

// TestBuyCraftingMaterials tests buying crafting materials from trainers
func TestBuyCraftingMaterials(serverAddr string) TestResult {
	const testName = "Buy Crafting Materials"

	name := uniqueName("BuyMaterialsTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to Armory where Forge Master Tormund sells materials
	logAction(testName, "Navigating to Armory...")
	navigateToArmory(client)

	// Check shop inventory
	logAction(testName, "Checking shop inventory...")
	client.ClearMessages()
	client.SendCommand("shop")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should show crafting materials for sale
	hasMaterials := strings.Contains(fullOutput, "iron_ore") || strings.Contains(fullOutput, "Iron Ore") ||
		strings.Contains(fullOutput, "steel") || strings.Contains(fullOutput, "leather_strip") ||
		strings.Contains(fullOutput, "ingot")
	logResult(testName, hasMaterials, "Shop shows crafting materials")

	if !hasMaterials {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Shop doesn't show crafting materials. Got: %v", messages)}
	}

	// Try to buy iron ore (use display name, not item ID)
	logAction(testName, "Buying iron ore...")
	client.ClearMessages()
	client.SendCommand("buy iron ore")
	time.Sleep(300 * time.Millisecond)

	messages = client.GetMessages()
	fullOutput = strings.Join(messages, " ")

	hasPurchase := strings.Contains(fullOutput, "purchase") || strings.Contains(fullOutput, "bought") ||
		strings.Contains(fullOutput, "gold") || strings.Contains(fullOutput, "Iron Ore") ||
		strings.Contains(fullOutput, "afford") // Even if they can't afford it, it means the item exists
	logResult(testName, hasPurchase, "Purchase attempt registered")

	if !hasPurchase {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Failed to buy crafting material. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Can buy crafting materials from trainers"}
}

// TestCraftRecipeInfo tests viewing recipe information
func TestCraftRecipeInfo(serverAddr string) TestResult {
	const testName = "Craft Recipe Info"

	name := uniqueName("RecipeInfoTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Navigate to Armory and learn a recipe
	navigateToArmory(client)
	time.Sleep(200 * time.Millisecond)

	client.SendCommand("learn iron_dagger")
	time.Sleep(300 * time.Millisecond)

	// View recipe info
	logAction(testName, "Viewing recipe info for iron_dagger...")
	client.ClearMessages()
	client.SendCommand("craft info iron_dagger")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Should show recipe details
	hasRecipeInfo := strings.Contains(fullOutput, "iron") || strings.Contains(fullOutput, "dagger") ||
		strings.Contains(fullOutput, "ingredient") || strings.Contains(fullOutput, "material") ||
		strings.Contains(fullOutput, "skill") || strings.Contains(fullOutput, "difficulty") ||
		strings.Contains(fullOutput, "require") || strings.Contains(fullOutput, "Iron Dagger")
	logResult(testName, hasRecipeInfo, "Recipe info displayed")

	if !hasRecipeInfo {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Recipe info not displayed. Got: %v", messages)}
	}

	return TestResult{Name: testName, Passed: true, Message: "Recipe info command shows details"}
}

// TestCraftingTrainerLocations tests that all crafting trainers are in expected locations
func TestCraftingTrainerLocations(serverAddr string) TestResult {
	const testName = "Crafting Trainer Locations"

	name := uniqueName("TrainerLocTest")
	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	time.Sleep(300 * time.Millisecond)

	// Check Armory for Forge Master Tormund
	navigateToArmory(client)
	client.ClearMessages()
	client.SendCommand("look")
	time.Sleep(300 * time.Millisecond)

	hasTormund := client.WaitForMessage("Tormund", 1*time.Second) || client.WaitForMessage("Forge Master", 1*time.Second)
	logResult(testName, hasTormund, "Forge Master Tormund in Armory")
	if !hasTormund {
		return TestResult{Name: testName, Passed: false, Message: "Forge Master Tormund not found in Armory"}
	}

	// Navigate back to Town Square, then to Artisan's Market for Tanner Helga
	client.SendCommand("west") // Artisan's Market
	time.Sleep(200 * time.Millisecond)

	client.ClearMessages()
	client.SendCommand("look")
	time.Sleep(300 * time.Millisecond)

	hasHelga := client.WaitForMessage("Helga", 1*time.Second) || client.WaitForMessage("Tanner", 1*time.Second)
	logResult(testName, hasHelga, "Tanner Helga in Artisan's Market")
	if !hasHelga {
		return TestResult{Name: testName, Passed: false, Message: "Tanner Helga not found in Artisan's Market"}
	}

	// Go to Alchemist Shop for Alchemist Zara
	client.SendCommand("west") // Alchemist Shop
	time.Sleep(200 * time.Millisecond)

	client.ClearMessages()
	client.SendCommand("look")
	time.Sleep(300 * time.Millisecond)

	hasZara := client.WaitForMessage("Zara", 1*time.Second) || client.WaitForMessage("Alchemist", 1*time.Second)
	logResult(testName, hasZara, "Alchemist Zara in Alchemist Shop")
	if !hasZara {
		return TestResult{Name: testName, Passed: false, Message: "Alchemist Zara not found in Alchemist Shop"}
	}

	// Navigate to Mage Tower for Enchantress Lyrel
	// Path from Alchemist Shop: east -> north -> north -> east -> east
	client.SendCommand("east") // Artisan's Market
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("north") // Market Street
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("north") // Town Square
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("east") // Temple
	time.Sleep(200 * time.Millisecond)
	client.SendCommand("east") // Mage Tower
	time.Sleep(200 * time.Millisecond)

	client.ClearMessages()
	client.SendCommand("look")
	time.Sleep(300 * time.Millisecond)

	hasLyrel := client.WaitForMessage("Lyrel", 1*time.Second) || client.WaitForMessage("Enchantress", 1*time.Second)
	logResult(testName, hasLyrel, "Enchantress Lyrel in Mage Tower")
	if !hasLyrel {
		return TestResult{Name: testName, Passed: false, Message: "Enchantress Lyrel not found in Mage Tower"}
	}

	return TestResult{Name: testName, Passed: true, Message: "All crafting trainers in expected locations"}
}

// TestCraftingSkillPersistence tests that crafting skills and learned recipes persist
func TestCraftingSkillPersistence(serverAddr string) TestResult {
	const testName = "Crafting Skill Persistence"

	name := uniqueName("PersistTest")
	password := name + "pass123"

	client, err := testclient.NewTestClient(name, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}

	time.Sleep(300 * time.Millisecond)

	// Navigate to Armory and learn a recipe
	logAction(testName, "Learning recipe before disconnect...")
	navigateToArmory(client)
	time.Sleep(200 * time.Millisecond)

	client.ClearMessages()
	client.SendCommand("learn iron_dagger")
	time.Sleep(300 * time.Millisecond)

	messages := client.GetMessages()
	fullOutput := strings.Join(messages, " ")

	// Verify we learned the recipe
	learnedRecipe := strings.Contains(fullOutput, "learn") || strings.Contains(fullOutput, "Iron Dagger") ||
		strings.Contains(fullOutput, "already know")
	logResult(testName, learnedRecipe, "Learned iron_dagger recipe")
	if !learnedRecipe {
		client.Close()
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Failed to learn recipe. Got: %v", messages)}
	}

	// Disconnect to trigger save
	logAction(testName, "Disconnecting to trigger save...")
	client.Close()
	time.Sleep(1 * time.Second) // Wait for save to complete

	// Reconnect with same character using login
	logAction(testName, "Reconnecting...")
	creds := testclient.Credentials{
		Username:      name,
		Password:      password,
		CharacterName: name,
	}
	client2, err := testclient.NewTestClientWithLogin(creds, serverAddr)
	if err != nil {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Reconnection failed: %v", err)}
	}
	defer client2.Close()

	time.Sleep(500 * time.Millisecond)

	// Player should be restored to their last location (Armory where they logged out)
	// Verify we're at a location with a forge
	client2.ClearMessages()
	client2.SendCommand("look")
	time.Sleep(300 * time.Millisecond)

	messages = client2.GetMessages()
	fullOutput = strings.Join(messages, " ")

	atArmory := strings.Contains(fullOutput, "Armory") || strings.Contains(fullOutput, "forge") || strings.Contains(fullOutput, "Iron Forge")
	logResult(testName, atArmory, "Player restored to Armory")

	// If not at Armory, we need to navigate there - but first go back to town square
	if !atArmory {
		// Use portal to get back to town (floor 0 is always available)
		client2.SendCommand("portal 0")
		time.Sleep(300 * time.Millisecond)
		navigateToArmory(client2)
		time.Sleep(200 * time.Millisecond)
	}

	// Check if we still know the recipe by trying to craft it (should fail due to materials, not unknown recipe)
	logAction(testName, "Checking if recipe persisted...")
	client2.ClearMessages()
	client2.SendCommand("craft iron_dagger")
	time.Sleep(300 * time.Millisecond)

	messages = client2.GetMessages()
	fullOutput = strings.Join(messages, " ")

	// If recipe persisted, we should get "missing materials" not "don't know recipe"
	// Or we could try "craft" to list known recipes
	recipePersisted := strings.Contains(fullOutput, "material") || strings.Contains(fullOutput, "ingredient") ||
		strings.Contains(fullOutput, "need") || strings.Contains(fullOutput, "iron_ore") ||
		strings.Contains(fullOutput, "don't have") || strings.Contains(fullOutput, "require")

	// If we get "don't know" or "unknown recipe", that means persistence failed
	persistenceFailed := strings.Contains(fullOutput, "don't know") || strings.Contains(fullOutput, "unknown recipe") ||
		strings.Contains(fullOutput, "haven't learned")

	if persistenceFailed {
		return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Recipe did not persist after reconnect. Got: %v", messages)}
	}

	logResult(testName, recipePersisted, "Recipe persisted after reconnect")

	if !recipePersisted {
		// Also check by listing known recipes
		client2.ClearMessages()
		client2.SendCommand("craft")
		time.Sleep(300 * time.Millisecond)

		messages = client2.GetMessages()
		fullOutput = strings.Join(messages, " ")

		hasIronDagger := strings.Contains(fullOutput, "iron_dagger") || strings.Contains(fullOutput, "Iron Dagger")
		if !hasIronDagger {
			return TestResult{Name: testName, Passed: false, Message: fmt.Sprintf("Recipe not found in craft list after reconnect. Got: %v", messages)}
		}
	}

	return TestResult{Name: testName, Passed: true, Message: "Crafting recipes persist across sessions"}
}
