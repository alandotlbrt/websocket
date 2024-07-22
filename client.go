package main

import (
	"sync"

	"github.com/gorilla/websocket"
)

type ClientList map[*Client]bool

type Client struct {
	connection *websocket.Conn

	manager *Manager

	roomID string

	egress chan []byte
}

func NewClient(conn *websocket.Conn, manager *Manager, roomID string) *Client {
	return &Client{
		connection: conn,
		manager:    manager,
		roomID:     roomID,
		egress:     make(chan []byte),
	}
}

type Manager struct {
	rooms map[string]ClientList
	sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		rooms: make(map[string]ClientList),
	}
}
