package db

import (
	"database/sql"
	"fmt"
)

// Contact represents a phone contact.
type Contact struct {
	ID           string
	Name         string
	PhoneNumber  string
	AvatarCached bool
	AvatarPath   string
}

// UpsertContact inserts or updates a contact.
func (db *DB) UpsertContact(c *Contact) error {
	_, err := db.Exec(`
		INSERT INTO contacts (id, name, phone_number, avatar_cached, avatar_path)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			phone_number = excluded.phone_number,
			avatar_cached = excluded.avatar_cached,
			avatar_path = excluded.avatar_path`,
		c.ID, c.Name, c.PhoneNumber, boolToInt(c.AvatarCached), c.AvatarPath,
	)
	if err != nil {
		return fmt.Errorf("upsert contact %s: %w", c.ID, err)
	}
	return nil
}

// GetContact retrieves a contact by ID.
func (db *DB) GetContact(id string) (*Contact, error) {
	c := &Contact{}
	var avatarCached int
	err := db.QueryRow(`
		SELECT id, name, phone_number, avatar_cached, avatar_path
		FROM contacts WHERE id = ?`, id,
	).Scan(&c.ID, &c.Name, &c.PhoneNumber, &avatarCached, &c.AvatarPath)
	if err != nil {
		return nil, fmt.Errorf("get contact %s: %w", id, err)
	}
	c.AvatarCached = avatarCached != 0
	return c, nil
}

// GetContactByPhone retrieves a contact by phone number.
func (db *DB) GetContactByPhone(phone string) (*Contact, error) {
	c := &Contact{}
	var avatarCached int
	err := db.QueryRow(`
		SELECT id, name, phone_number, avatar_cached, avatar_path
		FROM contacts WHERE phone_number = ?`, phone,
	).Scan(&c.ID, &c.Name, &c.PhoneNumber, &avatarCached, &c.AvatarPath)
	if err != nil {
		return nil, fmt.Errorf("get contact by phone %s: %w", phone, err)
	}
	c.AvatarCached = avatarCached != 0
	return c, nil
}

// ListContacts returns contacts, optionally filtered by name or phone number.
// If query is empty, all contacts are returned.
func (db *DB) ListContacts(query string) ([]Contact, error) {
	var rows *sql.Rows
	var err error

	if query == "" {
		rows, err = db.Query(`SELECT id, name, phone_number, avatar_cached, avatar_path FROM contacts ORDER BY name`)
	} else {
		like := "%" + query + "%"
		rows, err = db.Query(`
			SELECT id, name, phone_number, avatar_cached, avatar_path
			FROM contacts
			WHERE name LIKE ? OR phone_number LIKE ?
			ORDER BY name`, like, like,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("list contacts: %w", err)
	}
	defer rows.Close()

	var contacts []Contact
	for rows.Next() {
		var c Contact
		var avatarCached int
		if err := rows.Scan(&c.ID, &c.Name, &c.PhoneNumber, &avatarCached, &c.AvatarPath); err != nil {
			return nil, fmt.Errorf("scan contact: %w", err)
		}
		c.AvatarCached = avatarCached != 0
		contacts = append(contacts, c)
	}
	return contacts, rows.Err()
}

// UpdateContactAvatar sets the avatar path and marks it as cached.
func (db *DB) UpdateContactAvatar(id string, path string) error {
	res, err := db.Exec(`UPDATE contacts SET avatar_cached = 1, avatar_path = ? WHERE id = ?`, path, id)
	if err != nil {
		return fmt.Errorf("update contact avatar %s: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("update contact avatar %s: %w", id, sql.ErrNoRows)
	}
	return nil
}
