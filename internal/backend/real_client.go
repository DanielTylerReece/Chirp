package backend

import (
	"fmt"
	"sync"

	"github.com/rs/zerolog"
	"go.mau.fi/mautrix-gmessages/pkg/libgm"
	"go.mau.fi/mautrix-gmessages/pkg/libgm/gmproto"
)

// Compile-time assertion: RealClient implements GMClient.
var _ GMClient = (*RealClient)(nil)

// RealClient wraps libgm.Client and implements GMClient.
type RealClient struct {
	mu           sync.RWMutex
	client       *libgm.Client
	authData     *libgm.AuthData
	logger       zerolog.Logger
	eventHandler func(evt any)
}

// NewRealClient creates a new RealClient. If authData is nil, creates fresh auth data.
func NewRealClient(authData *libgm.AuthData, logger zerolog.Logger) *RealClient {
	if authData == nil {
		authData = libgm.NewAuthData()
	}
	rc := &RealClient{
		authData: authData,
		logger:   logger,
	}
	rc.client = libgm.NewClient(authData, nil, logger)
	rc.client.SetEventHandler(rc.handleEvent)
	return rc
}

func (rc *RealClient) handleEvent(evt any) {
	rc.mu.RLock()
	handler := rc.eventHandler
	rc.mu.RUnlock()
	if handler != nil {
		handler(evt)
	}
}

// AuthData returns the current auth data for persistence.
func (rc *RealClient) AuthData() *libgm.AuthData {
	return rc.authData
}

func (rc *RealClient) Connect() error {
	return rc.client.Connect()
}

func (rc *RealClient) Disconnect() {
	rc.client.Disconnect()
}

func (rc *RealClient) IsConnected() bool {
	return rc.client.IsConnected()
}

func (rc *RealClient) IsLoggedIn() bool {
	return rc.client.IsLoggedIn()
}

func (rc *RealClient) StartLogin() (string, error) {
	return rc.client.StartLogin()
}

func (rc *RealClient) SetEventHandler(handler func(evt any)) {
	rc.mu.Lock()
	rc.eventHandler = handler
	rc.mu.Unlock()
}

func (rc *RealClient) ListConversations(count int) error {
	_, err := rc.client.ListConversations(count, gmproto.ListConversationsRequest_INBOX)
	return err
}

func (rc *RealClient) FetchMessages(conversationID string, count int64) error {
	_, err := rc.client.FetchMessages(conversationID, count, nil)
	return err
}

func (rc *RealClient) SendMessage(conversationID string, text string) error {
	_, err := rc.client.SendMessage(&gmproto.SendMessageRequest{
		ConversationID: conversationID,
		MessagePayload: &gmproto.MessagePayload{
			MessagePayloadContent: &gmproto.MessagePayloadContent{
				MessageContent: &gmproto.MessageContent{
					Content: text,
				},
			},
		},
	})
	return err
}

func (rc *RealClient) SendMediaMessage(conversationID string, text string, mediaData []byte, fileName string, mimeType string) error {
	media, err := rc.client.UploadMedia(mediaData, fileName, mimeType)
	if err != nil {
		return fmt.Errorf("upload media: %w", err)
	}
	_, err = rc.client.SendMessage(&gmproto.SendMessageRequest{
		ConversationID: conversationID,
		MessagePayload: &gmproto.MessagePayload{
			MessagePayloadContent: &gmproto.MessagePayloadContent{
				MessageContent: &gmproto.MessageContent{
					Content: text,
				},
			},
			MessageInfo: []*gmproto.MessageInfo{{
				Data: &gmproto.MessageInfo_MediaContent{
					MediaContent: media,
				},
			}},
		},
	})
	return err
}

func (rc *RealClient) ListContacts() error {
	_, err := rc.client.ListContacts()
	return err
}

func (rc *RealClient) MarkRead(conversationID string, messageID string) error {
	return rc.client.MarkRead(conversationID, messageID)
}

func (rc *RealClient) SetTyping(conversationID string) error {
	return rc.client.SetTyping(conversationID, nil)
}

func (rc *RealClient) SendReaction(messageID string, emoji string) error {
	_, err := rc.client.SendReaction(&gmproto.SendReactionRequest{
		MessageID: messageID,
		ReactionData: &gmproto.ReactionData{
			Unicode: emoji,
		},
		Action: gmproto.SendReactionRequest_ADD,
	})
	return err
}
