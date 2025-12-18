package world

import (
	"fmt"
	"strings"
	"sync"

	"github.com/lawnchairsociety/opentowermud/server/internal/items"
	"github.com/lawnchairsociety/opentowermud/server/internal/npc"
)

type Room struct {
	ID               string
	Name             string
	Description      string
	DescriptionDay   string   // Day-specific description variant
	DescriptionNight string   // Night-specific description variant
	Type             RoomType // Room type (city, corridor, etc.)
	Features         []string // Interactive room features (altar, portal, stairs, etc.)
	Floor            int      // Tower floor number (0 = ground/city)
	Exits            map[string]*Room
	LockedExits      map[string]string // direction -> key ID required to unlock
	Items            []*items.Item
	NPCs             []*npc.NPC // NPCs in this room
	Players          []string   // Names of players currently in this room
	mu               sync.RWMutex
}

func NewRoom(id, name, description string, roomType RoomType) *Room {
	return &Room{
		ID:          id,
		Name:        name,
		Description: description,
		Type:        roomType,
		Features:    make([]string, 0),
		Exits:       make(map[string]*Room),
		LockedExits: make(map[string]string),
		Items:       make([]*items.Item, 0),
		NPCs:        make([]*npc.NPC, 0),
		Players:     make([]string, 0),
	}
}

func (r *Room) AddExit(direction string, room *Room) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Exits[direction] = room
}

func (r *Room) GetExit(direction string) interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	room := r.Exits[direction]
	if room == nil {
		return nil
	}
	return room
}

func (r *Room) AddItem(item *items.Item) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items.AddItem(&r.Items, item)
}

func (r *Room) RemoveItem(itemName string) (*items.Item, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return items.RemoveItem(&r.Items, itemName)
}

func (r *Room) HasItem(itemName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return items.HasItem(r.Items, itemName)
}

func (r *Room) FindItem(partial string) (*items.Item, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return items.FindItem(r.Items, partial)
}

// AddPlayer adds a player to this room
func (r *Room) AddPlayer(playerName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Players = append(r.Players, playerName)
}

// RemovePlayer removes a player from this room
func (r *Room) RemovePlayer(playerName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, name := range r.Players {
		if name == playerName {
			r.Players = append(r.Players[:i], r.Players[i+1:]...)
			return
		}
	}
}

// GetPlayers returns a copy of the player names in this room
func (r *Room) GetPlayers() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	players := make([]string, len(r.Players))
	copy(players, r.Players)
	return players
}

// AddNPC adds an NPC to this room
func (r *Room) AddNPC(n *npc.NPC) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.NPCs = append(r.NPCs, n)
}

// RemoveNPC removes an NPC from this room
func (r *Room) RemoveNPC(n *npc.NPC) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, roomNPC := range r.NPCs {
		if roomNPC == n {
			r.NPCs = append(r.NPCs[:i], r.NPCs[i+1:]...)
			return
		}
	}
}

// FindNPC finds an NPC by partial name match (case-insensitive)
// Prioritizes: exact match > prefix match > word match > contains match
func (r *Room) FindNPC(partial string) *npc.NPC {
	r.mu.RLock()
	defer r.mu.RUnlock()

	partial = strings.ToLower(partial)

	// First pass: exact match
	for _, n := range r.NPCs {
		if strings.ToLower(n.GetName()) == partial {
			return n
		}
	}

	// Second pass: name starts with search term
	for _, n := range r.NPCs {
		if strings.HasPrefix(strings.ToLower(n.GetName()), partial) {
			return n
		}
	}

	// Third pass: search term matches a word in the name
	for _, n := range r.NPCs {
		nameLower := strings.ToLower(n.GetName())
		words := strings.Fields(nameLower)
		for _, word := range words {
			if word == partial {
				return n
			}
		}
	}

	// Fourth pass: contains match (fallback)
	for _, n := range r.NPCs {
		if strings.Contains(strings.ToLower(n.GetName()), partial) {
			return n
		}
	}

	return nil
}

// GetNPCs returns a copy of the NPCs list
func (r *Room) GetNPCs() []*npc.NPC {
	r.mu.RLock()
	defer r.mu.RUnlock()

	npcs := make([]*npc.NPC, len(r.NPCs))
	copy(npcs, r.NPCs)
	return npcs
}

// GetDescription returns the room description (for a specific player to exclude them from the list)
func (r *Room) GetDescriptionForPlayer(playerName string) string {
	return r.GetDescriptionForPlayerWithCustomDesc(playerName, r.Description)
}

// GetDescriptionForPlayerWithCustomDesc builds a full room description using a custom base description
func (r *Room) GetDescriptionForPlayerWithCustomDesc(playerName string, baseDesc string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	desc := fmt.Sprintf("\n=== %s ===\n%s\n", r.Name, baseDesc)

	// Show NPCs in the room
	if len(r.NPCs) > 0 {
		npcNames := make([]string, len(r.NPCs))
		for i, n := range r.NPCs {
			npcNames[i] = fmt.Sprintf("%s (Level %d)", n.GetName(), n.GetLevel())
		}
		desc += "\nNPCs here: " + strings.Join(npcNames, ", ") + "\n"
	}

	// Show other players in the room
	otherPlayers := make([]string, 0)
	for _, name := range r.Players {
		if name != playerName {
			otherPlayers = append(otherPlayers, name)
		}
	}
	if len(otherPlayers) > 0 {
		desc += "\nPlayers here: " + strings.Join(otherPlayers, ", ") + "\n"
	}

	if len(r.Items) > 0 {
		itemNames := make([]string, len(r.Items))
		for i, item := range r.Items {
			itemNames[i] = item.Name
		}
		desc += "\nYou can see: " + strings.Join(itemNames, ", ") + "\n"
	}

	// Collect exits, including implicit exits from stairs features
	exits := make([]string, 0, len(r.Exits)+2)
	for direction := range r.Exits {
		exits = append(exits, direction)
	}
	// Add implicit exits for stairs features (floors are generated on-demand)
	// Check features inline since we already hold the lock
	hasStairsUp := false
	hasStairsDown := false
	for _, f := range r.Features {
		if f == "stairs_up" {
			hasStairsUp = true
		}
		if f == "stairs_down" {
			hasStairsDown = true
		}
	}
	if hasStairsUp && r.Exits["up"] == nil {
		exits = append(exits, "up")
	}
	if hasStairsDown && r.Exits["down"] == nil {
		exits = append(exits, "down")
	}
	if len(exits) > 0 {
		desc += "\nExits: " + strings.Join(exits, ", ") + "\n"
	}

	// Show room features that players can interact with
	if len(r.Features) > 0 {
		featureDescs := make([]string, 0, len(r.Features))
		for _, f := range r.Features {
			switch f {
			case "stairs_up":
				featureDescs = append(featureDescs, "a stairway leading up")
			case "stairs_down":
				featureDescs = append(featureDescs, "a stairway leading down")
			case "portal":
				featureDescs = append(featureDescs, "a glowing portal")
			case "altar":
				featureDescs = append(featureDescs, "an altar for respawning")
			case "treasure":
				featureDescs = append(featureDescs, "an opened treasure chest")
			case "boss":
				featureDescs = append(featureDescs, "an ominous presence")
			case "merchant":
				featureDescs = append(featureDescs, "a merchant's stall")
			case "locked_door":
				featureDescs = append(featureDescs, "a locked door")
			case "shortcut":
				featureDescs = append(featureDescs, "a shimmering portal to elsewhere in the labyrinth")
			case "labyrinth_entrance":
				featureDescs = append(featureDescs, "an entrance to the great labyrinth")
			case "lore_npc":
				featureDescs = append(featureDescs, "a scholar who knows ancient lore")
			case "forge":
				featureDescs = append(featureDescs, "a blazing forge")
			case "workbench":
				featureDescs = append(featureDescs, "a crafting workbench")
			case "alchemy_lab":
				featureDescs = append(featureDescs, "an alchemy laboratory")
			case "enchanting_table":
				featureDescs = append(featureDescs, "a glowing enchanting table")
			default:
				featureDescs = append(featureDescs, f)
			}
		}
		desc += "\nFeatures: " + strings.Join(featureDescs, ", ") + "\n"
	}

	return desc
}

// GetDescriptionForPlayerFiltered returns a room description with items filtered.
// excludeItemIDs contains item IDs that should not be shown (e.g., unique items the player already owns).
func (r *Room) GetDescriptionForPlayerFiltered(playerName string, excludeItemIDs []string) string {
	return r.GetDescriptionForPlayerFilteredWithCustomDesc(playerName, r.Description, excludeItemIDs)
}

// GetDescriptionForPlayerFilteredWithCustomDesc returns a room description with a custom base description and items filtered.
// excludeItemIDs contains item IDs that should not be shown (e.g., unique items the player already owns).
func (r *Room) GetDescriptionForPlayerFilteredWithCustomDesc(playerName string, baseDesc string, excludeItemIDs []string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	desc := fmt.Sprintf("\n=== %s ===\n%s\n", r.Name, baseDesc)

	// Show NPCs in the room
	if len(r.NPCs) > 0 {
		npcNames := make([]string, len(r.NPCs))
		for i, n := range r.NPCs {
			npcNames[i] = fmt.Sprintf("%s (Level %d)", n.GetName(), n.GetLevel())
		}
		desc += "\nNPCs here: " + strings.Join(npcNames, ", ") + "\n"
	}

	// Show other players in the room
	otherPlayers := make([]string, 0)
	for _, name := range r.Players {
		if name != playerName {
			otherPlayers = append(otherPlayers, name)
		}
	}
	if len(otherPlayers) > 0 {
		desc += "\nPlayers here: " + strings.Join(otherPlayers, ", ") + "\n"
	}

	// Show items, filtering out excluded IDs (unique items the player already owns)
	if len(r.Items) > 0 {
		// Build exclusion map for O(1) lookup
		excludeMap := make(map[string]bool)
		for _, id := range excludeItemIDs {
			excludeMap[id] = true
		}

		// Filter items
		var visibleItems []string
		for _, item := range r.Items {
			if !excludeMap[item.ID] {
				visibleItems = append(visibleItems, item.Name)
			}
		}

		if len(visibleItems) > 0 {
			desc += "\nYou can see: " + strings.Join(visibleItems, ", ") + "\n"
		}
	}

	// Collect exits, including implicit exits from stairs features
	exits := make([]string, 0, len(r.Exits)+2)
	for direction := range r.Exits {
		exits = append(exits, direction)
	}
	// Add implicit exits for stairs features (floors are generated on-demand)
	hasStairsUp := false
	hasStairsDown := false
	for _, f := range r.Features {
		if f == "stairs_up" {
			hasStairsUp = true
		}
		if f == "stairs_down" {
			hasStairsDown = true
		}
	}
	if hasStairsUp && r.Exits["up"] == nil {
		exits = append(exits, "up")
	}
	if hasStairsDown && r.Exits["down"] == nil {
		exits = append(exits, "down")
	}
	if len(exits) > 0 {
		desc += "\nExits: " + strings.Join(exits, ", ") + "\n"
	}

	// Show room features that players can interact with
	if len(r.Features) > 0 {
		featureDescs := make([]string, 0, len(r.Features))
		for _, f := range r.Features {
			switch f {
			case "stairs_up":
				featureDescs = append(featureDescs, "a stairway leading up")
			case "stairs_down":
				featureDescs = append(featureDescs, "a stairway leading down")
			case "portal":
				featureDescs = append(featureDescs, "a glowing portal")
			case "altar":
				featureDescs = append(featureDescs, "an altar for respawning")
			case "treasure":
				featureDescs = append(featureDescs, "an opened treasure chest")
			case "boss":
				featureDescs = append(featureDescs, "an ominous presence")
			case "merchant":
				featureDescs = append(featureDescs, "a merchant's stall")
			case "locked_door":
				featureDescs = append(featureDescs, "a locked door")
			case "shortcut":
				featureDescs = append(featureDescs, "a shimmering portal to elsewhere in the labyrinth")
			case "labyrinth_entrance":
				featureDescs = append(featureDescs, "an entrance to the great labyrinth")
			case "lore_npc":
				featureDescs = append(featureDescs, "a scholar who knows ancient lore")
			case "forge":
				featureDescs = append(featureDescs, "a blazing forge")
			case "workbench":
				featureDescs = append(featureDescs, "a crafting workbench")
			case "alchemy_lab":
				featureDescs = append(featureDescs, "an alchemy laboratory")
			case "enchanting_table":
				featureDescs = append(featureDescs, "a glowing enchanting table")
			default:
				featureDescs = append(featureDescs, f)
			}
		}
		desc += "\nFeatures: " + strings.Join(featureDescs, ", ") + "\n"
	}

	return desc
}

// GetDescription returns the room description (deprecated - use GetDescriptionForPlayer)
func (r *Room) GetDescription() string {
	return r.GetDescriptionForPlayer("")
}

// GetBaseDescription returns just the base room description text (without NPCs, items, etc.)
func (r *Room) GetBaseDescription() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.Description
}

// SetDescription sets the room's base description (thread-safe)
func (r *Room) SetDescription(desc string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Description = desc
}

// GetDescriptionDay returns the day-specific room description
func (r *Room) GetDescriptionDay() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.DescriptionDay
}

// GetDescriptionNight returns the night-specific room description
func (r *Room) GetDescriptionNight() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.DescriptionNight
}

// GetID returns the room's ID
func (r *Room) GetID() string {
	return r.ID
}

// GetExits returns a map of direction -> room name for all exits
func (r *Room) GetExits() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	exits := make(map[string]string)
	for direction, room := range r.Exits {
		if room != nil {
			exits[direction] = room.Name
		}
	}
	return exits
}

// HasFeature checks if the room has a specific feature
func (r *Room) HasFeature(feature string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, f := range r.Features {
		if f == feature {
			return true
		}
	}
	return false
}

// AddFeature adds a feature to the room
func (r *Room) AddFeature(feature string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if feature already exists
	for _, f := range r.Features {
		if f == feature {
			return
		}
	}
	r.Features = append(r.Features, feature)
}

// GetFeatures returns a copy of the room's features
func (r *Room) GetFeatures() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	features := make([]string, len(r.Features))
	copy(features, r.Features)
	return features
}

// GetFloor returns the tower floor number (0 = ground/city)
func (r *Room) GetFloor() int {
	return r.Floor
}

// LockExit locks an exit in the given direction, requiring the specified key to unlock
func (r *Room) LockExit(direction, keyID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.LockedExits == nil {
		r.LockedExits = make(map[string]string)
	}
	r.LockedExits[direction] = keyID
}

// UnlockExit removes the lock from an exit
func (r *Room) UnlockExit(direction string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.LockedExits != nil {
		delete(r.LockedExits, direction)
	}
}

// IsExitLocked returns true if the exit in the given direction is locked
func (r *Room) IsExitLocked(direction string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.LockedExits == nil {
		return false
	}
	_, locked := r.LockedExits[direction]
	return locked
}

// GetExitKeyRequired returns the key ID required to unlock an exit, or empty string if not locked
func (r *Room) GetExitKeyRequired(direction string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.LockedExits == nil {
		return ""
	}
	return r.LockedExits[direction]
}

// RemoveFeature removes a feature from the room
func (r *Room) RemoveFeature(feature string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, f := range r.Features {
		if f == feature {
			r.Features = append(r.Features[:i], r.Features[i+1:]...)
			return
		}
	}
}
