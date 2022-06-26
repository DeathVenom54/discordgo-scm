package discordgo_scm

import (
	"errors"
	"github.com/bwmarrin/discordgo"
)

// Some constants for convenience
const (
	InteractionTypeApplicationCommand  = discordgo.InteractionApplicationCommand
	InteractionTypeMessageComponent    = discordgo.InteractionMessageComponent
	InteractionTypeCommandAutocomplete = discordgo.InteractionApplicationCommandAutocomplete
	InteractionTypeModalSubmit         = discordgo.InteractionModalSubmit
)

// Feature essentially represents a command or a message component that
// you want to receive and respond to.
type Feature struct {
	Type    discordgo.InteractionType
	Handler func(*discordgo.Session, *discordgo.InteractionCreate)

	// ApplicationCommand if Type is discordgo.InteractionApplicationCommand or discordgo.InteractionApplicationCommandAutocomplete
	// Not needed for Type discordgo.InteractionMessageComponent
	ApplicationCommand *discordgo.ApplicationCommand

	// CustomID if Type is discordgo.InteractionMessageComponent
	CustomID string
}

// SCM represents the interaction manager, which handles
// responding to interactions as well as registering slash
// commands with the Discord API.
type SCM struct {
	Features      []*Feature
	botCommandIDs map[string][]string
}

// NewSCM creates a new SCM instance.
func NewSCM() *SCM {
	return &SCM{
		Features:      []*Feature{},
		botCommandIDs: map[string][]string{},
	}
}

// AddFeature adds a Feature to the SCM.
func (scm *SCM) AddFeature(feature *Feature) {
	scm.Features = append(scm.Features, feature)
}

// AddFeatures adds multiple Features to the SCM.
func (scm *SCM) AddFeatures(features []*Feature) {
	scm.Features = append(scm.Features, features...)
}

// CreateCommands registers any commands (Features with Type
// discordgo.InteractionApplicationCommand or
// discordgo.InteractionApplicationCommandAutocomplete) with the API.
// Leave guildID as empty string for global commands.
// NOTE: Bot must already be started beforehand.
func (scm *SCM) CreateCommands(s *discordgo.Session, guildID string) error {
	appID := s.State.User.ID
	// check if commands have already been registered
	if _, ok := scm.botCommandIDs[appID]; ok {
		return errors.New("this application already has registered commands once")
	}

	var applicationCommands []*discordgo.ApplicationCommand

	for _, f := range scm.Features {
		if f.Type == discordgo.InteractionApplicationCommand || f.Type == discordgo.InteractionApplicationCommandAutocomplete {
			applicationCommands = append(applicationCommands, f.ApplicationCommand)
		}
	}

	createdCommands, err := s.ApplicationCommandBulkOverwrite(appID, guildID, applicationCommands)

	if err != nil {
		return err
	}

	var createdCommandIDs []string

	for _, cc := range createdCommands {
		createdCommandIDs = append(createdCommandIDs, cc.ID)
	}

	scm.botCommandIDs[appID] = createdCommandIDs

	return nil
}

// DeleteCommands deletes any commands registered using CreateCommands with the API.
func (scm *SCM) DeleteCommands(s *discordgo.Session, guildID string) error {
	appID := s.State.User.ID

	for _, ccID := range scm.botCommandIDs[appID] {
		if err := s.ApplicationCommandDelete(appID, guildID, ccID); err != nil {
			return err
		}
	}

	return nil
}

// HandleInteractionCreate receives incoming interactions and runs the
// respective Feature's Handler.
func (scm *SCM) HandleInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Find relevant Feature
	var relevantFeature *Feature

	for _, f := range scm.Features {
		if f.Type == i.Type {
			if i.Type == discordgo.InteractionMessageComponent {
				// check if the CustomID matches
				if f.CustomID == i.MessageComponentData().CustomID {
					relevantFeature = f
					break
				}
			} else {
				// check the name of the command
				if f.ApplicationCommand.Name == i.ApplicationCommandData().Name {
					relevantFeature = f
					break
				}
			}
		}
	}

	// Handle if we have identified a relevant Feature
	if relevantFeature != nil {
		relevantFeature.Handler(s, i)
	}
}
