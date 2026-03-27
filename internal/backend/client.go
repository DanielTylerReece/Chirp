package backend

// GMClient abstracts the libgm client for testability.
// The real implementation wraps *libgm.Client.
// The mock implementation is in testutil/mock_client.go.
type GMClient interface {
	Connect() error
	Disconnect()
	IsConnected() bool
	IsLoggedIn() bool
	StartLogin() (string, error) // Returns QR code URL
	SetEventHandler(handler func(evt any))
	// Conversations
	ListConversations(count int) error
	FetchMessages(conversationID string, count int64) error
	// Messaging
	SendMessage(conversationID string, text string) error
	SendMediaMessage(conversationID string, text string, mediaData []byte, fileName string, mimeType string) error
	// Contacts
	ListContacts() error
	// Read receipts and typing
	MarkRead(conversationID string, messageID string) error
	SetTyping(conversationID string) error
	// Reactions
	SendReaction(messageID string, emoji string) error
}
