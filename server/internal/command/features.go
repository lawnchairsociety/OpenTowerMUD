package command

// FeatureHandler is a function that handles using a room feature
type FeatureHandler func(c *Command, p PlayerInterface, room RoomInterface) string

// featureHandlers maps feature names to their use handlers
var featureHandlers = map[string]FeatureHandler{
	"workbench": useWorkbench,
	"forge":     useForge,
}

// GetFeatureHandler returns the handler for a given feature, or nil if none exists
func GetFeatureHandler(feature string) FeatureHandler {
	return featureHandlers[feature]
}

// RegisterFeatureHandler allows registering custom feature handlers
func RegisterFeatureHandler(feature string, handler FeatureHandler) {
	featureHandlers[feature] = handler
}

// useWorkbench handles using a workbench for crafting
func useWorkbench(c *Command, p PlayerInterface, room RoomInterface) string {
	// For now, provide a placeholder message
	// This will be expanded when crafting system is implemented
	return `You approach the sturdy workbench and examine the tools laid out upon it.

The workbench is equipped for basic crafting:
  - Leather working tools
  - Woodworking implements
  - Simple assembly equipment

(Crafting system coming soon! For now, you can examine the workbench with 'look workbench'.)`
}

// useForge handles using a forge for smithing
func useForge(c *Command, p PlayerInterface, room RoomInterface) string {
	// For now, provide a placeholder message
	// This will be expanded when crafting system is implemented
	return `You step up to the blazing forge. The heat is intense, and sparks fly from the
coals as the bellows pump rhythmically.

The forge is equipped for metalworking:
  - Anvil and hammers
  - Tongs and quenching trough
  - Ore smelting crucible

(Smithing system coming soon! For now, you can examine the forge with 'look forge'.)`
}
