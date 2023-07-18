package main

import (
	"encoding/json"
	"log"
	"strings"
)

type UserEvent struct {
	message []byte
	user    *Client
}

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	userEvent chan UserEvent
}

func newHub() *Hub {
	return &Hub{
		userEvent:  make(chan UserEvent),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		case messageEvent := <-h.userEvent:
			HandleEvent(h, messageEvent)
		}

	}
}

type MessageCommand struct {
	Command string `json:"command"`
	Data    string `json:"data"`
}

func HandleEvent(h *Hub, query UserEvent) {
	var message MessageCommand
	json.Unmarshal(query.message, &message)
	log.Printf("received: %s %v", query.message, message)

	if message.Command == "GetChatroom" {
		GroupList := GetChatroom(appHub.enMicroMsgConn)
		responseByte := rjson(message.Command, "ok", 200, GroupList)
		if len(responseByte) > 0 {
			query.user.send <- responseByte
		}

	} else if message.Command == "SubmitCleanTask" {
		usernames := strings.Split(message.Data, ",")
		SubmitCleanTask(appHub.enMicroMsgConn, appHub.wxFileConn, usernames)
		query.user.send <- rjson(message.Command, "cleanup complete!", 200, nil)
	}
}
