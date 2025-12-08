package main

import (
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

// FloorYAML represents a floor in YAML format
type FloorYAML struct {
	Floor         int                  `yaml:"floor"`
	Tower         string               `yaml:"tower"`
	GeneratedSeed int64                `yaml:"generated_seed"`
	StairsUp      string               `yaml:"stairs_up,omitempty"`
	StairsDown    string               `yaml:"stairs_down,omitempty"`
	PortalRoom    string               `yaml:"portal_room,omitempty"`
	Rooms         map[string]*RoomYAML `yaml:"rooms"`
}

// RoomYAML represents a room in YAML format
type RoomYAML struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Type        string            `yaml:"type"`
	Features    []string          `yaml:"features,omitempty"`
	Exits       map[string]string `yaml:"exits,omitempty"`
}

// WriteFloorYAML writes a floor to a YAML file
func WriteFloorYAML(floor *FloorYAML, path string) error {
	// Create the file
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	// Write header comment
	fmt.Fprintf(f, "# Floor %d - %s tower\n", floor.Floor, floor.Tower)
	fmt.Fprintf(f, "# Generated with seed: %d\n", floor.GeneratedSeed)
	fmt.Fprintf(f, "# Room count: %d\n\n", len(floor.Rooms))

	// Create encoder with nice formatting
	encoder := yaml.NewEncoder(f)
	encoder.SetIndent(2)

	// We need to write in a specific order for readability
	// First write the metadata, then rooms sorted by ID
	orderedFloor := &orderedFloorYAML{
		Floor:         floor.Floor,
		Tower:         floor.Tower,
		GeneratedSeed: floor.GeneratedSeed,
		StairsUp:      floor.StairsUp,
		StairsDown:    floor.StairsDown,
		PortalRoom:    floor.PortalRoom,
		Rooms:         sortRooms(floor.Rooms),
	}

	if err := encoder.Encode(orderedFloor); err != nil {
		return fmt.Errorf("failed to encode YAML: %w", err)
	}

	return nil
}

// orderedFloorYAML is used for serialization with ordered rooms
type orderedFloorYAML struct {
	Floor         int                  `yaml:"floor"`
	Tower         string               `yaml:"tower"`
	GeneratedSeed int64                `yaml:"generated_seed"`
	StairsUp      string               `yaml:"stairs_up,omitempty"`
	StairsDown    string               `yaml:"stairs_down,omitempty"`
	PortalRoom    string               `yaml:"portal_room,omitempty"`
	Rooms         yaml.Node            `yaml:"rooms"`
}

// sortRooms returns rooms as an ordered YAML node
func sortRooms(rooms map[string]*RoomYAML) yaml.Node {
	// Get sorted room IDs
	ids := make([]string, 0, len(rooms))
	for id := range rooms {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	// Build ordered map node
	node := yaml.Node{
		Kind: yaml.MappingNode,
	}

	for _, id := range ids {
		room := rooms[id]

		// Room ID key
		keyNode := yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: id,
		}

		// Room value as a mapping
		valueNode := yaml.Node{
			Kind: yaml.MappingNode,
		}

		// Add room fields in order
		addStringField(&valueNode, "name", room.Name)
		addStringField(&valueNode, "description", room.Description)
		addStringField(&valueNode, "type", room.Type)

		if len(room.Features) > 0 {
			addSequenceField(&valueNode, "features", room.Features)
		}

		if len(room.Exits) > 0 {
			addMapField(&valueNode, "exits", room.Exits)
		}

		node.Content = append(node.Content, &keyNode, &valueNode)
	}

	return node
}

func addStringField(node *yaml.Node, key, value string) {
	node.Content = append(node.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Value: value},
	)
}

func addSequenceField(node *yaml.Node, key string, values []string) {
	seqNode := yaml.Node{Kind: yaml.SequenceNode}
	for _, v := range values {
		seqNode.Content = append(seqNode.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: v})
	}
	node.Content = append(node.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		&seqNode,
	)
}

func addMapField(node *yaml.Node, key string, values map[string]string) {
	// Sort keys for consistent output
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	// Sort exits in a logical order: north, south, east, west
	sort.Slice(keys, func(i, j int) bool {
		order := map[string]int{"north": 0, "south": 1, "east": 2, "west": 3, "up": 4, "down": 5}
		return order[keys[i]] < order[keys[j]]
	})

	mapNode := yaml.Node{Kind: yaml.MappingNode}
	for _, k := range keys {
		mapNode.Content = append(mapNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: k},
			&yaml.Node{Kind: yaml.ScalarNode, Value: values[k]},
		)
	}
	node.Content = append(node.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		&mapNode,
	)
}
