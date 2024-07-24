package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

func main() {
	setupAPI()
	fmt.Println("serv is listening..")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func setupAPI() {
	manager := NewManager()
	http.Handle("/", http.FileServer(http.Dir("./frontend")))
	http.HandleFunc("/ws", manager.serveWS)
}

var (
	websocketUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

func (m *Manager) serveWS(w http.ResponseWriter, r *http.Request) {
	log.Println("New connection")

	roomID := r.URL.Query().Get("room")
	if roomID == "" {
		http.Error(w, "Room ID is required", http.StatusBadRequest)
		return
	}

	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := NewClient(conn, m, roomID)

	m.addClient(roomID, client)

	go client.readMessages()
	go client.writeMessages()
}

func (m *Manager) addClient(roomID string, client *Client) {
	m.Lock()
	defer m.Unlock()
	if _, ok := m.rooms[roomID]; !ok {
		m.rooms[roomID] = make(ClientList)
	}
	m.rooms[roomID][client] = true
}

func (m *Manager) removeClient(roomID string, client *Client) {
	m.Lock()
	defer m.Unlock()
	if clients, ok := m.rooms[roomID]; ok {
		if _, ok := clients[client]; ok {
			client.connection.Close()
			delete(clients, client)
		}
		if len(clients) == 0 {
			delete(m.rooms, roomID)
		}
	}
}

func (c *Client) readMessages() {
	defer func() {
		c.manager.removeClient(c.roomID, c)
	}()
	for {
		_, payload, err := c.connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error reading message: %v", err)
			}
			break
		}
		log.Println("Payload: ", string(payload))
		for wsclient := range c.manager.rooms[c.roomID] {
			wsclient.egress <- payload
		}
	}
}

func (c *Client) writeMessages() {
	defer func() {
		c.manager.removeClient(c.roomID, c)
	}()
	for {
		select {
		case message, ok := <-c.egress:
			if !ok {
				if err := c.connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
					log.Println("connection closed: ", err)
				}
				return
			}
			if err := c.connection.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Println(err)
			}
			log.Println("sent message")
		}
	}
}
