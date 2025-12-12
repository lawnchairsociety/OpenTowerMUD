package command

import (
	"fmt"
	"strings"

	"github.com/lawnchairsociety/opentowermud/server/internal/npc"
	"github.com/lawnchairsociety/opentowermud/server/internal/quest"
	"github.com/lawnchairsociety/opentowermud/server/internal/world"
)

// executeQuest handles the quest/quests/journal command
func executeQuest(c *Command, p PlayerInterface) string {
	// If no args, show quest log summary
	if len(c.Args) == 0 {
		return showQuestSummary(p)
	}

	// Check for subcommands
	subcommand := strings.ToLower(c.Args[0])

	switch subcommand {
	case "list":
		return showQuestList(p)
	case "available":
		// Check if a quest name was provided
		if len(c.Args) > 1 {
			questName := strings.Join(c.Args[1:], " ")
			return showAvailableQuestDetails(p, questName)
		}
		return showAvailableQuests(p)
	default:
		// Try to show details for a specific quest
		return showQuestDetails(c, p)
	}
}

// showQuestSummary shows a brief summary of quest status
func showQuestSummary(p PlayerInterface) string {
	questLog := p.GetQuestLog()
	if questLog == nil {
		return "You have no quests."
	}

	activeQuests := questLog.GetActiveQuests()
	completedCount := len(questLog.GetCompletedQuests())

	if len(activeQuests) == 0 && completedCount == 0 {
		return "Your quest journal is empty. Talk to NPCs to find quests!"
	}

	var sb strings.Builder
	sb.WriteString("=== Quest Journal ===\n")
	sb.WriteString(fmt.Sprintf("Active Quests: %d\n", len(activeQuests)))
	sb.WriteString(fmt.Sprintf("Completed Quests: %d\n", completedCount))

	if len(activeQuests) > 0 {
		sb.WriteString("\nUse 'quest list' to see details of active quests.")
	}

	return sb.String()
}

// showQuestList shows all active quests with progress
func showQuestList(p PlayerInterface) string {
	server := p.GetServer().(ServerInterface)
	questRegistry := server.GetQuestRegistry()
	if questRegistry == nil {
		return "Quest system not available."
	}

	questLog := p.GetQuestLog()
	if questLog == nil {
		return "You have no active quests."
	}

	activeQuestIDs := questLog.GetActiveQuests()
	if len(activeQuestIDs) == 0 {
		return "You have no active quests. Talk to NPCs to find quests!"
	}

	var sb strings.Builder
	sb.WriteString("=== Active Quests ===\n\n")

	for _, questID := range activeQuestIDs {
		questDef, exists := questRegistry.GetQuest(questID)
		if !exists {
			continue
		}

		progress, hasProgress := questLog.GetQuestProgress(questID)
		if !hasProgress {
			continue
		}

		// Determine status tag
		statusTag := "[IN PROGRESS]"
		if questLog.CanCompleteQuest(questID, questDef) {
			statusTag = "[COMPLETE]"
		}

		sb.WriteString(fmt.Sprintf("%s %s\n", statusTag, questDef.Name))

		// Show objectives with progress
		for i, obj := range questDef.Objectives {
			current := 0
			if i < len(progress.Objectives) {
				current = progress.Objectives[i].Current
			}
			targetName := obj.TargetName
			if targetName == "" {
				targetName = obj.Target
			}
			sb.WriteString(fmt.Sprintf("  - %s %s: %d/%d\n",
				getObjectiveVerb(obj.Type), targetName, current, obj.Required))
		}
		sb.WriteString("\n")
	}

	return strings.TrimSuffix(sb.String(), "\n")
}

// showQuestDetails shows details for a specific quest
func showQuestDetails(c *Command, p PlayerInterface) string {
	server := p.GetServer().(ServerInterface)
	questRegistry := server.GetQuestRegistry()
	if questRegistry == nil {
		return "Quest system not available."
	}

	questLog := p.GetQuestLog()
	if questLog == nil {
		return "You have no quests."
	}

	// Get the quest name from args
	questName := strings.ToLower(strings.Join(c.Args, " "))

	// Search through active quests for a match
	activeQuestIDs := questLog.GetActiveQuests()
	for _, questID := range activeQuestIDs {
		questDef, exists := questRegistry.GetQuest(questID)
		if !exists {
			continue
		}

		if strings.Contains(strings.ToLower(questDef.Name), questName) ||
			strings.Contains(strings.ToLower(questID), questName) {
			return formatQuestDetails(questDef, questLog)
		}
	}

	return fmt.Sprintf("No active quest matching '%s'. Use 'quest list' to see your quests.", strings.Join(c.Args, " "))
}

// formatQuestDetails formats detailed quest information
func formatQuestDetails(q *quest.Quest, questLog *quest.PlayerQuestLog) string {
	var sb strings.Builder

	progress, _ := questLog.GetQuestProgress(q.ID)

	// Status
	statusTag := "[IN PROGRESS]"
	if questLog.CanCompleteQuest(q.ID, q) {
		statusTag = "[COMPLETE]"
	}

	sb.WriteString(fmt.Sprintf("=== %s %s ===\n\n", statusTag, q.Name))
	sb.WriteString(fmt.Sprintf("%s\n\n", q.Description))

	// Objectives
	sb.WriteString("Objectives:\n")
	for i, obj := range q.Objectives {
		current := 0
		if progress != nil && i < len(progress.Objectives) {
			current = progress.Objectives[i].Current
		}
		complete := current >= obj.Required
		checkmark := " "
		if complete {
			checkmark = "x"
		}
		targetName := obj.TargetName
		if targetName == "" {
			targetName = obj.Target
		}
		sb.WriteString(fmt.Sprintf("  [%s] %s %s: %d/%d\n",
			checkmark, getObjectiveVerb(obj.Type), targetName, current, obj.Required))
	}

	// Turn-in NPC
	if q.TurnInNPC != "" {
		sb.WriteString(fmt.Sprintf("\nTurn in to: %s\n", q.TurnInNPC))
	}

	// Rewards
	sb.WriteString("\nRewards:\n")
	if q.Rewards.Gold > 0 {
		sb.WriteString(fmt.Sprintf("  - %d gold\n", q.Rewards.Gold))
	}
	if q.Rewards.Experience > 0 {
		sb.WriteString(fmt.Sprintf("  - %d experience\n", q.Rewards.Experience))
	}
	for _, item := range q.Rewards.Items {
		sb.WriteString(fmt.Sprintf("  - %s\n", item))
	}
	if len(q.Rewards.Recipes) > 0 {
		sb.WriteString(fmt.Sprintf("  - Recipe: %s\n", strings.Join(q.Rewards.Recipes, ", ")))
	}
	if q.Rewards.Title != "" {
		sb.WriteString(fmt.Sprintf("  - Title: %s\n", q.Rewards.Title))
	}

	return sb.String()
}

// showAvailableQuests shows quests available from NPCs in the current room
func showAvailableQuests(p PlayerInterface) string {
	room := p.GetCurrentRoom()
	if room == nil {
		return "You are nowhere."
	}

	worldRoom, ok := room.(*world.Room)
	if !ok {
		return "Internal error: invalid room type"
	}

	server := p.GetServer().(ServerInterface)
	questRegistry := server.GetQuestRegistry()
	if questRegistry == nil {
		return "Quest system not available."
	}

	// Find quest givers in the room
	var questGivers []*npc.NPC
	for _, n := range worldRoom.GetNPCs() {
		if n.IsQuestGiver() && n.IsAlive() {
			questGivers = append(questGivers, n)
		}
	}

	if len(questGivers) == 0 {
		return "There is no one here offering quests."
	}

	// Get player's quest state for filtering
	playerState := p.GetQuestState()

	return listAvailableQuests(questGivers, questRegistry, playerState)
}

// showAvailableQuestDetails shows details of an available quest before accepting
func showAvailableQuestDetails(p PlayerInterface, questName string) string {
	room := p.GetCurrentRoom()
	if room == nil {
		return "You are nowhere."
	}

	worldRoom, ok := room.(*world.Room)
	if !ok {
		return "Internal error: invalid room type"
	}

	server := p.GetServer().(ServerInterface)
	questRegistry := server.GetQuestRegistry()
	if questRegistry == nil {
		return "Quest system not available."
	}

	// Find quest givers in the room
	var questGivers []*npc.NPC
	for _, n := range worldRoom.GetNPCs() {
		if n.IsQuestGiver() && n.IsAlive() {
			questGivers = append(questGivers, n)
		}
	}

	if len(questGivers) == 0 {
		return "There is no one here offering quests."
	}

	// Get player's quest state for filtering
	playerState := p.GetQuestState()
	searchName := strings.ToLower(questName)

	// Search through available quests from all quest givers
	for _, giver := range questGivers {
		available := questRegistry.GetAvailableQuestsForPlayer(giver.GetName(), playerState)
		for _, q := range available {
			if strings.Contains(strings.ToLower(q.Name), searchName) ||
				strings.Contains(strings.ToLower(q.ID), searchName) {
				return formatAvailableQuestDetails(q, giver.GetName())
			}
		}
	}

	return fmt.Sprintf("No available quest matching '%s'. Use 'quests available' to see available quests.", questName)
}

// formatAvailableQuestDetails formats details for a quest that hasn't been accepted yet
func formatAvailableQuestDetails(q *quest.Quest, giverName string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("=== %s ===\n", q.Name))
	sb.WriteString(fmt.Sprintf("Offered by: %s\n\n", giverName))
	sb.WriteString(fmt.Sprintf("%s\n\n", q.Description))

	// Objectives preview
	sb.WriteString("Objectives:\n")
	for _, obj := range q.Objectives {
		targetName := obj.TargetName
		if targetName == "" {
			targetName = obj.Target
		}
		sb.WriteString(fmt.Sprintf("  - %s %s: 0/%d\n",
			getObjectiveVerb(obj.Type), targetName, obj.Required))
	}

	// Turn-in NPC
	if q.TurnInNPC != "" {
		sb.WriteString(fmt.Sprintf("\nTurn in to: %s\n", q.TurnInNPC))
	}

	// Rewards
	sb.WriteString("\nRewards:\n")
	if q.Rewards.Gold > 0 {
		sb.WriteString(fmt.Sprintf("  - %d gold\n", q.Rewards.Gold))
	}
	if q.Rewards.Experience > 0 {
		sb.WriteString(fmt.Sprintf("  - %d experience\n", q.Rewards.Experience))
	}
	for _, item := range q.Rewards.Items {
		sb.WriteString(fmt.Sprintf("  - %s\n", item))
	}
	if len(q.Rewards.Recipes) > 0 {
		sb.WriteString(fmt.Sprintf("  - Recipe: %s\n", strings.Join(q.Rewards.Recipes, ", ")))
	}
	if q.Rewards.Title != "" {
		sb.WriteString(fmt.Sprintf("  - Title: %s\n", q.Rewards.Title))
	}

	sb.WriteString(fmt.Sprintf("\nUse 'accept %s' to accept this quest.", strings.ToLower(q.Name)))

	return sb.String()
}

// getObjectiveVerb returns the appropriate verb for an objective type
func getObjectiveVerb(objType quest.QuestType) string {
	switch objType {
	case quest.QuestTypeKill:
		return "Kill"
	case quest.QuestTypeFetch:
		return "Collect"
	case quest.QuestTypeDelivery:
		return "Deliver"
	case quest.QuestTypeExplore:
		return "Explore"
	case quest.QuestTypeCraft:
		return "Craft"
	case quest.QuestTypeCast:
		return "Cast"
	default:
		return "Complete"
	}
}

// executeAccept handles the accept command for quests
func executeAccept(c *Command, p PlayerInterface) string {
	room := p.GetCurrentRoom()
	if room == nil {
		return "You are nowhere."
	}

	worldRoom, ok := room.(*world.Room)
	if !ok {
		return "Internal error: invalid room type"
	}

	server := p.GetServer().(ServerInterface)
	questRegistry := server.GetQuestRegistry()
	if questRegistry == nil {
		return "Quest system not available."
	}

	// Find quest givers in the room
	var questGivers []*npc.NPC
	for _, n := range worldRoom.GetNPCs() {
		if n.IsQuestGiver() && n.IsAlive() {
			questGivers = append(questGivers, n)
		}
	}

	if len(questGivers) == 0 {
		return "There is no one here offering quests."
	}

	// Get player's quest state for filtering
	playerState := p.GetQuestState()

	// Require a quest name argument
	if len(c.Args) == 0 {
		return "Usage: accept <quest name>\nUse 'quests available' to see available quests."
	}

	// Try to accept a specific quest
	questName := strings.ToLower(strings.Join(c.Args, " "))
	return acceptQuest(p, questGivers, questRegistry, playerState, questName, server)
}

// listAvailableQuests shows all quests available from nearby NPCs
func listAvailableQuests(questGivers []*npc.NPC, registry *quest.QuestRegistry, state *quest.PlayerQuestState) string {
	var sb strings.Builder
	sb.WriteString("=== Available Quests ===\n\n")

	foundQuests := false

	for _, giver := range questGivers {
		available := registry.GetAvailableQuestsForPlayer(giver.GetName(), state)
		if len(available) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("%s offers:\n", giver.GetName()))
		for _, q := range available {
			tag := "[NEW]"
			sb.WriteString(fmt.Sprintf("  %s %s\n", tag, q.Name))
		}
		sb.WriteString("\n")
		foundQuests = true
	}

	if !foundQuests {
		return "There are no quests available to you right now."
	}

	sb.WriteString("Use 'accept <quest name>' to accept a quest.")
	return sb.String()
}

// acceptQuest attempts to accept a specific quest
func acceptQuest(p PlayerInterface, questGivers []*npc.NPC, registry *quest.QuestRegistry, state *quest.PlayerQuestState, questName string, server ServerInterface) string {
	// Search for matching quest
	for _, giver := range questGivers {
		available := registry.GetAvailableQuestsForPlayer(giver.GetName(), state)
		for _, q := range available {
			if strings.Contains(strings.ToLower(q.Name), questName) ||
				strings.Contains(strings.ToLower(q.ID), questName) {
				// Found a matching quest - accept it
				questLog := p.GetQuestLog()
				if err := questLog.StartQuest(q); err != nil {
					return fmt.Sprintf("Failed to accept quest: %v", err)
				}

				// Give quest items if this is a delivery quest
				if len(q.QuestItems) > 0 {
					for _, itemID := range q.QuestItems {
						item := server.CreateItem(itemID)
						if item != nil {
							p.AddQuestItem(item)
						}
					}
				}

				var sb strings.Builder
				sb.WriteString(fmt.Sprintf("Quest Accepted: %s\n\n", q.Name))
				sb.WriteString(fmt.Sprintf("%s\n\n", q.Description))
				sb.WriteString("Objectives:\n")
				for _, obj := range q.Objectives {
					targetName := obj.TargetName
					if targetName == "" {
						targetName = obj.Target
					}
					sb.WriteString(fmt.Sprintf("  - %s %s: 0/%d\n",
						getObjectiveVerb(obj.Type), targetName, obj.Required))
				}

				if len(q.QuestItems) > 0 {
					sb.WriteString("\nYou received quest items.")
				}

				return sb.String()
			}
		}
	}

	return fmt.Sprintf("No quest matching '%s' is available here.", questName)
}

// executeComplete handles the complete/turnin command
func executeComplete(c *Command, p PlayerInterface) string {
	room := p.GetCurrentRoom()
	if room == nil {
		return "You are nowhere."
	}

	worldRoom, ok := room.(*world.Room)
	if !ok {
		return "Internal error: invalid room type"
	}

	server := p.GetServer().(ServerInterface)
	questRegistry := server.GetQuestRegistry()
	if questRegistry == nil {
		return "Quest system not available."
	}

	questLog := p.GetQuestLog()
	if questLog == nil {
		return "You have no quests to complete."
	}

	// Find NPCs that can accept quest turn-ins
	npcs := worldRoom.GetNPCs()

	// Find completable quests that can be turned in to NPCs in this room
	var completableQuests []*quest.Quest
	var turnInNPC *npc.NPC

	for _, questID := range questLog.GetActiveQuests() {
		questDef, exists := questRegistry.GetQuest(questID)
		if !exists {
			continue
		}

		if !questLog.CanCompleteQuest(questID, questDef) {
			continue
		}

		// Check if any NPC here can accept this turn-in
		for _, n := range npcs {
			if !n.IsAlive() {
				continue
			}
			// Check if NPC name matches turn-in NPC or NPC can turn in this quest
			if strings.EqualFold(n.GetName(), questDef.TurnInNPC) || n.CanTurnInQuest(questID) {
				completableQuests = append(completableQuests, questDef)
				turnInNPC = n
				break
			}
		}
	}

	if len(completableQuests) == 0 {
		// Check if there are completed quests but no NPC to turn them in to
		hasCompletedQuest := false
		for _, questID := range questLog.GetActiveQuests() {
			questDef, exists := questRegistry.GetQuest(questID)
			if exists && questLog.CanCompleteQuest(questID, questDef) {
				hasCompletedQuest = true
				break
			}
		}

		if hasCompletedQuest {
			return "You have completed quests, but the NPC to turn them in to is not here."
		}
		return "You have no quests ready to turn in."
	}

	// Complete the first available quest (or specific one if provided)
	var questToComplete *quest.Quest

	if len(c.Args) > 0 {
		questName := strings.ToLower(strings.Join(c.Args, " "))
		for _, q := range completableQuests {
			if strings.Contains(strings.ToLower(q.Name), questName) {
				questToComplete = q
				break
			}
		}
		if questToComplete == nil {
			return fmt.Sprintf("No completable quest matching '%s' found here.", questName)
		}
	} else if len(completableQuests) == 1 {
		questToComplete = completableQuests[0]
	} else {
		// Multiple completable quests - list them
		var sb strings.Builder
		sb.WriteString("Multiple quests can be completed here:\n")
		for _, q := range completableQuests {
			sb.WriteString(fmt.Sprintf("  - %s\n", q.Name))
		}
		sb.WriteString("\nUse 'complete <quest name>' to turn in a specific quest.")
		return sb.String()
	}

	// Award rewards
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== Quest Complete: %s ===\n\n", questToComplete.Name))
	sb.WriteString(fmt.Sprintf("%s says, \"Well done, adventurer!\"\n\n", turnInNPC.GetName()))
	sb.WriteString("Rewards:\n")

	if questToComplete.Rewards.Gold > 0 {
		p.AddGold(questToComplete.Rewards.Gold)
		sb.WriteString(fmt.Sprintf("  + %d gold\n", questToComplete.Rewards.Gold))
	}

	if questToComplete.Rewards.Experience > 0 {
		levelUps := p.GainExperience(questToComplete.Rewards.Experience)
		sb.WriteString(fmt.Sprintf("  + %d experience\n", questToComplete.Rewards.Experience))
		for _, lu := range levelUps {
			sb.WriteString(fmt.Sprintf("\n*** LEVEL UP! You are now level %d! ***\n", lu.NewLevel))
		}
	}

	for _, itemID := range questToComplete.Rewards.Items {
		item := server.CreateItem(itemID)
		if item != nil {
			p.AddItem(item)
			sb.WriteString(fmt.Sprintf("  + %s\n", item.Name))
		}
	}

	for _, recipeID := range questToComplete.Rewards.Recipes {
		if !p.KnowsRecipe(recipeID) {
			p.LearnRecipe(recipeID)
			sb.WriteString(fmt.Sprintf("  + Recipe learned: %s\n", recipeID))
		}
	}

	if questToComplete.Rewards.Title != "" {
		p.EarnTitle(questToComplete.Rewards.Title)
		sb.WriteString(fmt.Sprintf("  + Title earned: %s\n", questToComplete.Rewards.Title))
	}

	// Remove quest items
	if len(questToComplete.QuestItems) > 0 {
		p.ClearQuestInventoryForQuest(questToComplete.QuestItems)
	}

	// Complete the quest in the log
	questLog.TurnInQuest(questToComplete.ID, questToComplete.Repeatable)

	// Record quest completion in statistics
	p.RecordQuestCompleted()

	return sb.String()
}

// executeTitle handles the title command
func executeTitle(c *Command, p PlayerInterface) string {
	if len(c.Args) == 0 {
		return showTitles(p)
	}

	titleArg := strings.Join(c.Args, " ")

	if strings.EqualFold(titleArg, "none") || strings.EqualFold(titleArg, "clear") {
		if err := p.SetActiveTitle(""); err != nil {
			return fmt.Sprintf("Failed to clear title: %v", err)
		}
		return "Your title has been cleared."
	}

	// Find the matching title (case-insensitive) from earned titles
	earnedTitles := p.GetEarnedTitles()
	var matchedTitle string
	for _, title := range earnedTitles {
		if strings.EqualFold(title, titleArg) {
			matchedTitle = title
			break
		}
	}

	if matchedTitle == "" {
		return "Cannot set title: you have not earned that title"
	}

	if err := p.SetActiveTitle(matchedTitle); err != nil {
		return fmt.Sprintf("Cannot set title: %v", err)
	}

	return fmt.Sprintf("Your title is now: %s", p.GetActiveTitle())
}

// showTitles lists earned titles
func showTitles(p PlayerInterface) string {
	titles := p.GetEarnedTitles()
	activeTitle := p.GetActiveTitle()

	if len(titles) == 0 {
		return "You have not earned any titles yet."
	}

	var sb strings.Builder
	sb.WriteString("=== Your Titles ===\n\n")

	for _, title := range titles {
		marker := "  "
		if title == activeTitle {
			marker = "> "
		}
		sb.WriteString(fmt.Sprintf("%s%s\n", marker, title))
	}

	sb.WriteString("\nUse 'title <name>' to set your active title, or 'title none' to clear it.")
	return sb.String()
}
