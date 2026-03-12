package main

import (
	"log"
	"os/exec"
)

// TypeCommand represents a single typing action to execute.
type TypeCommand struct {
	Text string
	Key  string
}

var keyMap = map[string]string{
	"Backspace":  "BackSpace",
	"Enter":      "Return",
	"Tab":        "Tab",
	"Escape":     "Escape",
	"ArrowLeft":  "Left",
	"ArrowRight": "Right",
	"ArrowUp":    "Up",
	"ArrowDown":  "Down",
	"Delete":     "Delete",
}

func mapKeyName(key string) string {
	if mapped, ok := keyMap[key]; ok {
		return mapped
	}
	return key
}

func typeText(text string) error {
	cmd := exec.Command("xdotool", "type", "--clearmodifiers", "--delay", "0", "--", text)
	return cmd.Run()
}

func sendKey(key string) error {
	cmd := exec.Command("xdotool", "key", "--clearmodifiers", mapKeyName(key))
	return cmd.Run()
}

// StartTyper starts a goroutine that processes typing commands sequentially.
func StartTyper() chan<- TypeCommand {
	ch := make(chan TypeCommand, 64)
	go func() {
		for cmd := range ch {
			var err error
			if cmd.Text != "" {
				err = typeText(cmd.Text)
			} else if cmd.Key != "" {
				err = sendKey(cmd.Key)
			}
			if err != nil {
				log.Printf("xdotool error: %v", err)
			}
		}
	}()
	return ch
}
