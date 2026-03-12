package main

// WSMessage is the JSON message exchanged over the WebSocket.
type WSMessage struct {
	Type    string `json:"type"`
	Data    string `json:"data,omitempty"`
	Key     string `json:"key,omitempty"`
	Message string `json:"message,omitempty"`
}
