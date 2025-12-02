package command

import (
	"fmt"
	"strings"

	"github.com/lawnchairsociety/opentowermud/server/internal/class"
)

// executeClass handles all class-related commands
func executeClass(c *Command, p PlayerInterface) string {
	if len(c.Args) == 0 {
		// Show current class information
		return showClassInfo(p)
	}

	subcommand := strings.ToLower(c.Args[0])

	switch subcommand {
	case "list", "all":
		return listAllClasses(p)
	case "info":
		if len(c.Args) < 2 {
			return "Usage: class info <classname>"
		}
		return showClassDetails(c.Args[1])
	case "switch", "active":
		if len(c.Args) < 2 {
			return "Usage: class switch <classname>\nSwitch which class gains XP from combat."
		}
		return switchActiveClass(p, c.Args[1])
	default:
		return fmt.Sprintf("Unknown class command: %s\nUse 'class' to see your classes, 'class list' for all classes, or 'class switch <class>' to change active class.", subcommand)
	}
}

// showClassInfo displays the player's current class information
func showClassInfo(p PlayerInterface) string {
	var sb strings.Builder

	sb.WriteString("=== Your Classes ===\n")
	sb.WriteString(fmt.Sprintf("Classes: %s\n", p.GetClassLevelsSummary()))
	sb.WriteString(fmt.Sprintf("Active Class: %s (gains XP from combat)\n", p.GetActiveClassName()))
	sb.WriteString(fmt.Sprintf("Primary Class: %s\n", p.GetPrimaryClassName()))

	// Show multiclass status
	if p.CanMulticlass() {
		sb.WriteString("\nMulticlassing: UNLOCKED")
		sb.WriteString("\n  Visit a class trainer to learn a new class.")
		sb.WriteString("\n  Use 'class list' to see available classes and requirements.")
	} else {
		sb.WriteString(fmt.Sprintf("\nMulticlassing: Reach level %d in your primary class to unlock.", class.MinLevelForMulticlass))
	}

	sb.WriteString("\n\nCommands:")
	sb.WriteString("\n  class list         - View all classes and requirements")
	sb.WriteString("\n  class info <class> - View detailed class information")
	sb.WriteString("\n  class switch <class> - Change which class gains XP")

	return sb.String()
}

// listAllClasses shows all available classes with their requirements
func listAllClasses(p PlayerInterface) string {
	var sb strings.Builder

	sb.WriteString("=== Available Classes ===\n")

	for _, c := range class.AllClasses() {
		def := class.GetDefinition(c)
		if def == nil {
			continue
		}

		// Show if player has this class
		hasClass := false
		classLevel := 0
		classLevels := p.GetAllClassLevelsMap()
		if level, ok := classLevels[string(c)]; ok && level > 0 {
			hasClass = true
			classLevel = level
		}

		// Build status indicator
		status := ""
		if hasClass {
			status = fmt.Sprintf(" [Level %d]", classLevel)
		}

		sb.WriteString(fmt.Sprintf("\n%s%s\n", c.String(), status))
		sb.WriteString(fmt.Sprintf("  %s\n", def.Description))
		sb.WriteString(fmt.Sprintf("  Hit Die: d%d | Primary: %s\n", def.HitDie, def.PrimaryStat))
		sb.WriteString(fmt.Sprintf("  Multiclass Requirements: %s\n", def.GetMulticlassRequirementsString()))

		// Show if player can multiclass into this
		if !hasClass && p.CanMulticlass() {
			canMulti, reason := p.CanMulticlassInto(string(c))
			if canMulti {
				sb.WriteString("  Status: Available to learn!\n")
			} else {
				sb.WriteString(fmt.Sprintf("  Status: %s\n", reason))
			}
		}
	}

	return sb.String()
}

// showClassDetails shows detailed information about a specific class
func showClassDetails(className string) string {
	c, err := class.ParseClass(className)
	if err != nil {
		return fmt.Sprintf("Unknown class: %s\nValid classes: warrior, mage, cleric, rogue, ranger, paladin", className)
	}

	def := class.GetDefinition(c)
	if def == nil {
		return fmt.Sprintf("Class definition not found for %s", c.String())
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("=== %s ===\n", c.String()))
	sb.WriteString(fmt.Sprintf("%s\n\n", def.Description))

	sb.WriteString("Combat Stats:\n")
	sb.WriteString(fmt.Sprintf("  Hit Die: d%d (average %d HP per level)\n", def.HitDie, (def.HitDie/2)+1))
	sb.WriteString(fmt.Sprintf("  Starting HP: %d + CON modifier\n", def.StartingHP))
	sb.WriteString(fmt.Sprintf("  Starting Mana: %d\n", def.StartingMana))
	sb.WriteString(fmt.Sprintf("  Mana per Level: %d\n", def.ManaPerLevel))
	sb.WriteString(fmt.Sprintf("  Primary Stat: %s\n", def.PrimaryStat))

	sb.WriteString("\nProficiencies:\n")
	sb.WriteString(fmt.Sprintf("  Armor: %s\n", formatArmorProficiencies(def.ArmorProficiencies)))
	sb.WriteString(fmt.Sprintf("  Weapons: %s\n", formatWeaponProficiencies(def.WeaponProficiencies)))

	sb.WriteString(fmt.Sprintf("\nMulticlass Requirements: %s\n", def.GetMulticlassRequirementsString()))

	// Class-specific abilities preview
	sb.WriteString("\nClass Abilities:\n")
	sb.WriteString(getClassAbilitiesPreview(c))

	return sb.String()
}

// switchActiveClass changes which class gains XP
func switchActiveClass(p PlayerInterface, className string) string {
	err := p.SwitchActiveClass(className)
	if err != nil {
		return fmt.Sprintf("Cannot switch to %s: %s", className, err.Error())
	}

	return fmt.Sprintf("Active class changed to %s. You will now gain XP in this class.", p.GetActiveClassName())
}

// formatArmorProficiencies formats armor proficiencies for display
func formatArmorProficiencies(profs []class.ArmorType) string {
	if len(profs) == 0 || (len(profs) == 1 && profs[0] == class.ArmorNone) {
		return "None"
	}
	parts := make([]string, 0, len(profs))
	for _, p := range profs {
		if p != class.ArmorNone {
			parts = append(parts, string(p))
		}
	}
	if len(parts) == 0 {
		return "None"
	}
	return strings.Join(parts, ", ")
}

// formatWeaponProficiencies formats weapon proficiencies for display
func formatWeaponProficiencies(profs []class.WeaponType) string {
	if len(profs) == 0 {
		return "None"
	}
	parts := make([]string, len(profs))
	for i, p := range profs {
		parts[i] = string(p)
	}
	return strings.Join(parts, ", ")
}

// getClassAbilitiesPreview returns a preview of class abilities
func getClassAbilitiesPreview(c class.Class) string {
	switch c {
	case class.Warrior:
		return `  - Melee damage bonus (+1 per 3 levels)
  - Heavy armor AC bonus (level 10+)
  - Second Wind: HP regen in combat (level 15+)
  - HP bonus (+10% at level 20)`
	case class.Mage:
		return `  - Powerful damage spells (fireball, ice storm, meteor)
  - INT-based spellcasting
  - Arcane Shield: +2 AC (level 15+)
  - Highest spell damage potential`
	case class.Cleric:
		return `  - Healing spells (heal, cure wounds, resurrection)
  - WIS-based spellcasting
  - Divine Protection: +1 AC (level 10+)
  - Sanctuary: 25% damage reduction below 25% HP (level 20+)`
	case class.Rogue:
		return `  - Sneak Attack (+1d6, +1d6 every 5 levels)
  - Finesse weapon proficiency (DEX for attack/damage)
  - Evasion: 10% dodge chance (level 15+)
  - Assassinate: Execute enemies below 20% HP (level 20+)`
	case class.Ranger:
		return `  - Ranged damage bonus (+2 base, +1 per 3 levels)
  - Favored Enemy: +25% damage vs beasts
  - Nature spells (hunter's mark, spike growth)
  - Multishot: 20% chance for double attack (level 20+)`
	case class.Paladin:
		return `  - Smite: Extra radiant damage
  - Holy damage bonus vs undead/demons (+2)
  - Healing spells (lay on hands, cure wounds)
  - Lay on Hands: HP regen out of combat (level 15+)`
	default:
		return "  No special abilities defined."
	}
}
