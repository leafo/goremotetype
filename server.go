package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type Server struct {
	hub      *Hub
	typer    *Typer
	upgrader websocket.Upgrader
}

func NewServer(typer *Typer) *Server {
	return &Server{
		hub:   NewHub(),
		typer: typer,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool {
				return true
			},
		},
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/assets/app.js", s.handleAppJS)
	mux.HandleFunc("/assets/favicon.svg", s.handleFaviconSVG)
	mux.HandleFunc("/assets/styles.css", s.handleStylesCSS)
	mux.HandleFunc("/ws", s.handleWS)
	return mux
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(indexHTML)
}

func (s *Server) handleAppJS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	_, _ = w.Write(appJS)
}

func (s *Server) handleStylesCSS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	_, _ = w.Write(stylesCSS)
}

func (s *Server) handleFaviconSVG(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	_, _ = w.Write(faviconSVG)
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := s.hub.Add(conn)
	if err := client.writeJSON(WSMessage{Type: "status", Message: "connected"}); err != nil {
		s.hub.Remove(client)
		return
	}

	log.Printf("client connected: %s", conn.RemoteAddr())

	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg WSMessage
		if err := json.Unmarshal(payload, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "text":
			if msg.Data != "" {
				s.typer.SendText(msg.Data)
			}
		case "key":
			if msg.Key != "" {
				s.typer.SendKey(msg.Key)
			}
		case "compositionupdate":
			s.typer.SetComposition(msg.Data)
		case "compositioncommit":
			s.typer.CommitComposition(msg.Data)
		case "clear":
			s.typer.Clear()
		case "debuglog":
			log.Printf("frontend debug log begin: %s", conn.RemoteAddr())
			fmt.Println(msg.Data)
			log.Printf("frontend debug log end: %s", conn.RemoteAddr())
		}
	}

	log.Printf("client disconnected: %s", conn.RemoteAddr())
	s.hub.Remove(client)
}
