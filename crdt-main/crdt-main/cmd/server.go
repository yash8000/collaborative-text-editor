package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"crdt/pkg/rga"
	"crdt/pkg/ws"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Load the RGA document
	documentFile := "document.json"
	rgaDoc, err := rga.LoadFromFile(documentFile)
	if err != nil {
		log.Println("Error loading document:", err)
		rgaDoc = rga.NewRGA()
	}

	// Initialize WebSocket manager
	wsManager := ws.NewManager(rgaDoc, documentFile)

	// Start the WebSocket server
	addr := fmt.Sprintf(":%s", os.Getenv("PORT"))
	http.HandleFunc("/ws", wsManager.HandleConnection)

	// Serve static files
	http.Handle("/", http.FileServer(http.Dir("./static")))

	fmt.Printf("WebSocket server running at ws://localhost%s/ws\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
