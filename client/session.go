package client

type SessionResponse struct {
	SessionID string `json:"sessionid"`
	EditURL   string `json:"editurl"`
}

type SaveMessage struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

type ResultMessage struct {
	Type    string `json:"type"`
	Success bool   `json:"success"`
	Reason  string `json:"reason,omitempty"`
}
