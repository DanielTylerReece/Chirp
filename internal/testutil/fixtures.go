package testutil

import "fmt"

// ConversationFixture holds sample conversation data for tests.
type ConversationFixture struct {
	ID                 string
	Name               string
	IsGroup            bool
	LastMessageTS      int64
	LastMessagePreview string
	UnreadCount        int
}

// MessageFixture holds sample message data for tests.
type MessageFixture struct {
	ID             string
	ConversationID string
	ParticipantID  string
	Body           string
	TimestampMS    int64
	IsFromMe       bool
	Status         int
}

// ContactFixture holds sample contact data for tests.
type ContactFixture struct {
	ID          string
	Name        string
	PhoneNumber string
}

// MakeConversation creates a single ConversationFixture with defaults.
func MakeConversation(id, name string, isGroup bool) ConversationFixture {
	return ConversationFixture{
		ID:                 id,
		Name:               name,
		IsGroup:            isGroup,
		LastMessageTS:      1711400000000,
		LastMessagePreview: "Last message preview",
		UnreadCount:        0,
	}
}

// MakeMessage creates a single MessageFixture.
func MakeMessage(id, convID string, fromMe bool, body string, timestampMS int64) MessageFixture {
	participantID := "other-participant"
	if fromMe {
		participantID = "me"
	}
	return MessageFixture{
		ID:             id,
		ConversationID: convID,
		ParticipantID:  participantID,
		Body:           body,
		TimestampMS:    timestampMS,
		IsFromMe:       fromMe,
		Status:         2, // delivered
	}
}

// MakeContact creates a single ContactFixture.
func MakeContact(id, name, phone string) ContactFixture {
	return ContactFixture{
		ID:          id,
		Name:        name,
		PhoneNumber: phone,
	}
}

// SampleConversations returns 5 sample conversations for testing.
func SampleConversations() []ConversationFixture {
	return []ConversationFixture{
		{ID: "conv-1", Name: "Alice", IsGroup: false, LastMessageTS: 1711400005000, LastMessagePreview: "See you tomorrow!", UnreadCount: 1},
		{ID: "conv-2", Name: "Bob", IsGroup: false, LastMessageTS: 1711400004000, LastMessagePreview: "Thanks!", UnreadCount: 0},
		{ID: "conv-3", Name: "Work Group", IsGroup: true, LastMessageTS: 1711400003000, LastMessagePreview: "Meeting at 3pm", UnreadCount: 3},
		{ID: "conv-4", Name: "Charlie", IsGroup: false, LastMessageTS: 1711400002000, LastMessagePreview: "Got it", UnreadCount: 0},
		{ID: "conv-5", Name: "Family", IsGroup: true, LastMessageTS: 1711400001000, LastMessagePreview: "Happy birthday!", UnreadCount: 0},
	}
}

// SampleMessages returns messages for a conversation, alternating sent/received.
func SampleMessages(convID string, count int) []MessageFixture {
	msgs := make([]MessageFixture, count)
	baseTS := int64(1711400000000)
	for i := 0; i < count; i++ {
		fromMe := i%2 == 0
		msgs[i] = MakeMessage(
			fmt.Sprintf("msg-%s-%d", convID, i),
			convID,
			fromMe,
			fmt.Sprintf("Message %d in conversation %s", i, convID),
			baseTS+int64(i*60000),
		)
	}
	return msgs
}

// SampleContacts returns 5 sample contacts.
func SampleContacts() []ContactFixture {
	return []ContactFixture{
		{ID: "contact-1", Name: "Alice Smith", PhoneNumber: "+15551234567"},
		{ID: "contact-2", Name: "Bob Jones", PhoneNumber: "+15559876543"},
		{ID: "contact-3", Name: "Charlie Brown", PhoneNumber: "+15555551234"},
		{ID: "contact-4", Name: "Diana Prince", PhoneNumber: "+15558675309"},
		{ID: "contact-5", Name: "Eve Wilson", PhoneNumber: "+15551112222"},
	}
}
