package db

import (
	"testing"
)

func TestUpsertAndGetContact(t *testing.T) {
	db := mustOpenTestDB(t)

	c := &Contact{
		ID:           "contact-1",
		Name:         "Alice Smith",
		PhoneNumber:  "+15551234567",
		AvatarCached: false,
		AvatarPath:   "",
	}

	if err := db.UpsertContact(c); err != nil {
		t.Fatalf("UpsertContact: %v", err)
	}

	got, err := db.GetContact("contact-1")
	if err != nil {
		t.Fatalf("GetContact: %v", err)
	}

	if got.ID != c.ID {
		t.Errorf("ID: got %q, want %q", got.ID, c.ID)
	}
	if got.Name != c.Name {
		t.Errorf("Name: got %q, want %q", got.Name, c.Name)
	}
	if got.PhoneNumber != c.PhoneNumber {
		t.Errorf("PhoneNumber: got %q, want %q", got.PhoneNumber, c.PhoneNumber)
	}
	if got.AvatarCached != c.AvatarCached {
		t.Errorf("AvatarCached: got %v, want %v", got.AvatarCached, c.AvatarCached)
	}
	if got.AvatarPath != c.AvatarPath {
		t.Errorf("AvatarPath: got %q, want %q", got.AvatarPath, c.AvatarPath)
	}
}

func TestUpsertContactUpdatesExisting(t *testing.T) {
	db := mustOpenTestDB(t)

	c := &Contact{ID: "contact-1", Name: "Alice", PhoneNumber: "+15551234567"}
	if err := db.UpsertContact(c); err != nil {
		t.Fatalf("UpsertContact (insert): %v", err)
	}

	c.Name = "Alice Smith-Jones"
	c.PhoneNumber = "+15559999999"
	if err := db.UpsertContact(c); err != nil {
		t.Fatalf("UpsertContact (update): %v", err)
	}

	got, err := db.GetContact("contact-1")
	if err != nil {
		t.Fatalf("GetContact: %v", err)
	}
	if got.Name != "Alice Smith-Jones" {
		t.Errorf("Name: got %q, want %q", got.Name, "Alice Smith-Jones")
	}
	if got.PhoneNumber != "+15559999999" {
		t.Errorf("PhoneNumber: got %q, want %q", got.PhoneNumber, "+15559999999")
	}
}

func TestGetContactByPhone(t *testing.T) {
	db := mustOpenTestDB(t)

	contacts := []*Contact{
		{ID: "c1", Name: "Alice", PhoneNumber: "+15551234567"},
		{ID: "c2", Name: "Bob", PhoneNumber: "+15559876543"},
	}
	for _, c := range contacts {
		if err := db.UpsertContact(c); err != nil {
			t.Fatalf("UpsertContact: %v", err)
		}
	}

	got, err := db.GetContactByPhone("+15559876543")
	if err != nil {
		t.Fatalf("GetContactByPhone: %v", err)
	}
	if got.Name != "Bob" {
		t.Errorf("Name: got %q, want %q", got.Name, "Bob")
	}
	if got.ID != "c2" {
		t.Errorf("ID: got %q, want %q", got.ID, "c2")
	}
}

func TestGetContactByPhoneNotFound(t *testing.T) {
	db := mustOpenTestDB(t)

	_, err := db.GetContactByPhone("+10000000000")
	if err == nil {
		t.Error("expected error for nonexistent phone")
	}
}

func TestListContactsAll(t *testing.T) {
	db := mustOpenTestDB(t)

	contacts := []*Contact{
		{ID: "c1", Name: "Charlie", PhoneNumber: "+15551111111"},
		{ID: "c2", Name: "Alice", PhoneNumber: "+15552222222"},
		{ID: "c3", Name: "Bob", PhoneNumber: "+15553333333"},
	}
	for _, c := range contacts {
		if err := db.UpsertContact(c); err != nil {
			t.Fatalf("UpsertContact: %v", err)
		}
	}

	got, err := db.ListContacts("")
	if err != nil {
		t.Fatalf("ListContacts: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 contacts, got %d", len(got))
	}
	// Should be ordered by name
	if got[0].Name != "Alice" {
		t.Errorf("first contact: got %q, want %q", got[0].Name, "Alice")
	}
	if got[1].Name != "Bob" {
		t.Errorf("second contact: got %q, want %q", got[1].Name, "Bob")
	}
	if got[2].Name != "Charlie" {
		t.Errorf("third contact: got %q, want %q", got[2].Name, "Charlie")
	}
}

func TestListContactsWithNameQuery(t *testing.T) {
	db := mustOpenTestDB(t)

	contacts := []*Contact{
		{ID: "c1", Name: "Alice Smith", PhoneNumber: "+15551111111"},
		{ID: "c2", Name: "Bob Jones", PhoneNumber: "+15552222222"},
		{ID: "c3", Name: "Alice Johnson", PhoneNumber: "+15553333333"},
	}
	for _, c := range contacts {
		if err := db.UpsertContact(c); err != nil {
			t.Fatalf("UpsertContact: %v", err)
		}
	}

	got, err := db.ListContacts("Alice")
	if err != nil {
		t.Fatalf("ListContacts: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 contacts matching 'Alice', got %d", len(got))
	}
}

func TestListContactsWithPhoneQuery(t *testing.T) {
	db := mustOpenTestDB(t)

	contacts := []*Contact{
		{ID: "c1", Name: "Alice", PhoneNumber: "+15551234567"},
		{ID: "c2", Name: "Bob", PhoneNumber: "+15559876543"},
	}
	for _, c := range contacts {
		if err := db.UpsertContact(c); err != nil {
			t.Fatalf("UpsertContact: %v", err)
		}
	}

	got, err := db.ListContacts("9876")
	if err != nil {
		t.Fatalf("ListContacts: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 contact matching phone '9876', got %d", len(got))
	}
	if got[0].Name != "Bob" {
		t.Errorf("Name: got %q, want %q", got[0].Name, "Bob")
	}
}

func TestUpdateContactAvatar(t *testing.T) {
	db := mustOpenTestDB(t)

	c := &Contact{ID: "c1", Name: "Alice", PhoneNumber: "+15551234567"}
	if err := db.UpsertContact(c); err != nil {
		t.Fatalf("UpsertContact: %v", err)
	}

	if err := db.UpdateContactAvatar("c1", "/cache/avatars/c1.png"); err != nil {
		t.Fatalf("UpdateContactAvatar: %v", err)
	}

	got, err := db.GetContact("c1")
	if err != nil {
		t.Fatalf("GetContact: %v", err)
	}
	if !got.AvatarCached {
		t.Error("AvatarCached should be true after update")
	}
	if got.AvatarPath != "/cache/avatars/c1.png" {
		t.Errorf("AvatarPath: got %q, want %q", got.AvatarPath, "/cache/avatars/c1.png")
	}
}

func TestUpdateContactAvatarNotFound(t *testing.T) {
	db := mustOpenTestDB(t)

	err := db.UpdateContactAvatar("nonexistent", "/some/path.png")
	if err == nil {
		t.Error("expected error updating avatar for nonexistent contact")
	}
}

func TestGetContactNotFound(t *testing.T) {
	db := mustOpenTestDB(t)

	_, err := db.GetContact("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent contact")
	}
}

func TestListContactsEmpty(t *testing.T) {
	db := mustOpenTestDB(t)

	got, err := db.ListContacts("")
	if err != nil {
		t.Fatalf("ListContacts: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for empty list, got %v", got)
	}
}
