package backend

import "fmt"

// SIMInfo is the backend-agnostic representation of a SIM card.
type SIMInfo struct {
	ParticipantID string // The participant ID associated with this SIM
	SIMNumber     int32  // 1-indexed SIM slot number
	CarrierName   string // e.g., "US Mobile"
	PhoneNumber   string // Formatted phone number, e.g., "(865) 320-5104"
	ColorHex      string // Carrier color
}

// DisplayLabel returns a short label for the SIM selector UI.
func (s SIMInfo) DisplayLabel() string {
	if s.CarrierName != "" && s.PhoneNumber != "" {
		return s.CarrierName + " " + s.PhoneNumber
	}
	if s.PhoneNumber != "" {
		return s.PhoneNumber
	}
	if s.CarrierName != "" {
		return s.CarrierName
	}
	return "SIM " + fmt.Sprintf("%d", s.SIMNumber)
}

// ConversationData is the backend-agnostic representation of a conversation
// returned from ListConversations. Maps 1:1 to protobuf fields.
type ConversationData struct {
	ID                 string
	Name               string
	IsGroup            bool
	LastMessageTS      int64
	LastMessagePreview string
	Unread             bool
	IsPinned           bool
	IsArchived         bool
	IsRCS              bool
	AvatarHexColor     string
	DefaultOutgoingID  string // Participant ID of the "me" SIM for this conversation
	Participants       []ParticipantData
}

// ParticipantData is the backend-agnostic representation of a participant.
type ParticipantData struct {
	ID             string
	Name           string
	PhoneNumber    string
	IsMe           bool
	AvatarHexColor string
	ContactID      string
	SIMNumber      int32 // SIM slot number (from participant's SimPayload), 0 if unknown
}

// MessageData is the backend-agnostic representation of a message
// returned from FetchMessages.
type MessageData struct {
	ID             string
	ConversationID string
	ParticipantID  string
	Body           string
	TimestampMS    int64
	IsFromMe       bool
	Status         int
	MediaID        string
	MediaMimeType  string
	MediaDecryptKey []byte
	MediaSize       int64
	MediaWidth      int
	MediaHeight     int
	ThumbnailID     string
	ThumbnailKey    []byte
	ReplyToID       string
}

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
	ListConversations(count int) ([]ConversationData, error)
	FetchMessages(conversationID string, count int64) ([]MessageData, error)
	// Messaging — simNumber is the 1-indexed SIM slot (0 = no preference / use default)
	SendMessage(conversationID string, text string, simNumber int32) error
	SendMediaMessage(conversationID string, text string, mediaData []byte, fileName string, mimeType string, simNumber int32) error
	// Contacts
	ListContacts() error
	// Read receipts and typing
	MarkRead(conversationID string, messageID string) error
	SetTyping(conversationID string) error
	// Reactions
	SendReaction(messageID string, emoji string) error
	// Media
	DownloadMedia(mediaID string, decryptKey []byte) ([]byte, error)
	// Thumbnails — returns participantID→JPEG bytes map
	FetchParticipantThumbnails(participantIDs []string) (map[string][]byte, error)
	// SIM info
	GetSIMs() []SIMInfo
	// Conversations (create)
	GetOrCreateConversation(numbers []string) (string, error)
}
