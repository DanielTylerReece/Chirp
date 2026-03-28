package backend

import (
	"log"
	"os"
	"path/filepath"

	"github.com/tyler/gmessage/internal/app"
	"github.com/tyler/gmessage/internal/db"
)

// ContactManager handles contact syncing and avatar caching.
type ContactManager struct {
	client   GMClient
	database *db.DB
	config   *app.Config
}

func NewContactManager(client GMClient, database *db.DB, config *app.Config) *ContactManager {
	return &ContactManager{
		client:   client,
		database: database,
		config:   config,
	}
}

// SyncContacts fetches contacts from the phone and stores them locally.
func (cm *ContactManager) SyncContacts() error {
	// The actual contact data comes through events — just trigger the fetch
	return cm.client.ListContacts()
}

// ResolveParticipantName looks up a phone number in the contacts DB.
// Returns the contact name if found, or the formatted phone number.
func (cm *ContactManager) ResolveParticipantName(phone string) string {
	contact, err := cm.database.GetContactByPhone(phone)
	if err != nil || contact == nil {
		return formatPhone(phone)
	}
	return contact.Name
}

// GetAvatarPath returns the cached avatar path for a contact.
// Returns empty string if no avatar is cached.
func (cm *ContactManager) GetAvatarPath(contactID string) string {
	contact, err := cm.database.GetContact(contactID)
	if err != nil || contact == nil || !contact.AvatarCached {
		return ""
	}
	return contact.AvatarPath
}

// CacheAvatar saves avatar data to disk and updates the contact record.
func (cm *ContactManager) CacheAvatar(contactID string, data []byte) error {
	path := filepath.Join(cm.config.AvatarDir, contactID+".jpg")
	if err := os.WriteFile(path, data, 0600); err != nil {
		return err
	}
	return cm.database.UpdateContactAvatar(contactID, path)
}

// LinkParticipantsToContacts matches participants to contacts by phone number.
func (cm *ContactManager) LinkParticipantsToContacts() error {
	// Get all contacts
	contacts, err := cm.database.ListContacts("")
	if err != nil {
		return err
	}

	// Build phone→contact map
	phoneMap := make(map[string]*db.Contact)
	for i := range contacts {
		normalized := normalizePhone(contacts[i].PhoneNumber)
		if normalized != "" {
			phoneMap[normalized] = &contacts[i]
		}
	}

	// This is a simplified version — full implementation would iterate
	// all participants and link them
	log.Printf("contacts: built phone map with %d entries", len(phoneMap))

	return nil
}

// FetchAndCacheAvatars fetches participant thumbnails from the phone and
// saves them to disk. participantIDs should be the non-"me" participant IDs
// from conversations. Thumbnails are stored as JPEGs in config.AvatarDir
// and the participant DB record is updated with the file path.
func (cm *ContactManager) FetchAndCacheAvatars(participantIDs []string) error {
	if len(participantIDs) == 0 {
		return nil
	}

	cached := 0

	// Batch in groups of 20 to avoid overwhelming the API
	for i := 0; i < len(participantIDs); i += 20 {
		end := i + 20
		if end > len(participantIDs) {
			end = len(participantIDs)
		}
		batch := participantIDs[i:end]

		// Try GetParticipantThumbnail first
		thumbs, err := cm.client.FetchParticipantThumbnails(batch)
		if err != nil {
			log.Printf("contacts: participant thumbnail batch error: %v", err)
			continue
		}

		for id, data := range thumbs {
			if len(data) == 0 {
				continue
			}
			avatarPath := filepath.Join(cm.config.AvatarDir, id+".jpg")
			if err := os.WriteFile(avatarPath, data, 0600); err != nil {
				log.Printf("contacts: write avatar %s: %v", id, err)
				continue
			}
			if err := cm.database.UpdateParticipantAvatar(id, avatarPath); err != nil {
				log.Printf("contacts: update participant avatar %s: %v", id, err)
				continue
			}
			thumbs[id] = nil // release bytes after writing to disk
			cached++
		}
	}

	log.Printf("contacts: cached %d/%d participant avatars", cached, len(participantIDs))
	return nil
}

// normalizePhone strips formatting from a phone number for comparison.
func normalizePhone(phone string) string {
	var digits []byte
	for _, c := range phone {
		if c >= '0' && c <= '9' {
			digits = append(digits, byte(c))
		} else if c == '+' && len(digits) == 0 {
			digits = append(digits, byte(c))
		}
	}
	return string(digits)
}

// formatPhone formats a phone number for display.
func formatPhone(phone string) string {
	if phone == "" {
		return "Unknown"
	}
	// Simple formatting — just return as-is for now
	return phone
}
