package ws

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"crdt/pkg/rga"

	"github.com/coder/websocket"
)

type Manager struct {
	clients      map[*websocket.Conn]bool
	clientsMutex sync.Mutex
	rgaDoc       *rga.RGA
	documentFile string
}

func NewManager(rgaDoc *rga.RGA, documentFile string) *Manager {
	return &Manager{
		clients:      make(map[*websocket.Conn]bool),
		rgaDoc:       rgaDoc,
		documentFile: documentFile,
	}
}

func (m *Manager) HandleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Println("WebSocket upgrade failed:", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "closing")

	log.Println("Client connected")

	// Add the client to the list of active connections
	m.clientsMutex.Lock()
	m.clients[conn] = true
	m.clientsMutex.Unlock()

	// Send the current document to the client
	m.sendDocument(conn, r.Context())

	// Handle incoming messages
	for {
		_, data, err := conn.Read(r.Context())
		if err != nil {
			log.Println("Read error:", err)
			break
		}

		log.Printf("Received: %s", string(data))
		m.handleMessage(data)
	}

	// Remove the client from the list of active connections
	m.clientsMutex.Lock()
	delete(m.clients, conn)
	m.clientsMutex.Unlock()
}

func (m *Manager) handleMessage(data []byte) {
	var msg rga.RGAMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Println("Error unmarshalling message:", err)
		return
	}

	switch msg.Type {
	case "Insert":
		m.rgaDoc.Insert(msg.After, msg.ID, msg.Value, time.Now())
	case "Delete":
		m.rgaDoc.Delete(msg.ID)
	default:
		log.Println("Unknown message type:", msg.Type)
		return
	}

	// Save the updated document to the file
	if err := m.rgaDoc.SaveToFile(m.documentFile); err != nil {
		log.Println("Error saving document:", err)
	}

	// Broadcast the updated document to all clients
	m.broadcast()
}

func (m *Manager) sendDocument(conn *websocket.Conn, ctx context.Context) {
	document := m.rgaDoc.GetDocument()
	documentJSON, err := json.Marshal(document)
	if err != nil {
		log.Println("Error marshalling document:", err)
		return
	}
	// Send the document to the client
	if err := conn.Write(ctx, websocket.MessageText, documentJSON); err != nil {
		log.Println("Error sending document to client:", err)
	}
}

func (m *Manager) broadcast() {
	m.clientsMutex.Lock()
	defer m.clientsMutex.Unlock()

	document := m.rgaDoc.GetDocument()
	documentJSON, err := json.Marshal(document)
	if err != nil {
		log.Println("Error marshalling document for broadcast:", err)
		return
	}

	for client := range m.clients {
		err := client.Write(context.Background(), websocket.MessageText, documentJSON)
		if err != nil {
			log.Println("Error broadcasting message:", err)
			client.Close(websocket.StatusInternalError, "error broadcasting")
			delete(m.clients, client)
		}
	}
}
