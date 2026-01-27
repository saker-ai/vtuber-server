package ws

// Message represents a message.
type Message struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}
