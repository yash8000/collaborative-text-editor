package rga

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"
)

type RGAMessage struct {
	Type      string `json:"type"`
	After     string `json:"after"`
	ID        string `json:"id"`
	Value     string `json:"value"`
	TimeStamp string `json:"timestamp"`
}

type RGAElement struct {
	ID        string
	Value     string
	Timestamp time.Time
	Deleted   bool
	Prev      *RGAElement
	Next      *RGAElement
}

type RGA struct {
	mu       sync.Mutex
	elements map[string]*RGAElement
	order    []*RGAElement
	head     *RGAElement
	tail     *RGAElement
	// idIndex  map[*RGAElement]int
}

type VisibleElement struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

func NewRGA() *RGA {
	head := &RGAElement{ID: "head"}
	tail := &RGAElement{ID: "tail"}
	head.Next = tail
	tail.Prev = head

	return &RGA{
		elements: map[string]*RGAElement{
			"head": head,
			"tail": tail,
		},
		head:  head,
		tail:  tail,
		order: []*RGAElement{head, tail},
	}
}

func (r *RGA) Insert(afterID string, newID string, value string, ts time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	log.Printf("Inserting element: %s after %s value %s", newID, afterID, value)

	if _, exists := r.elements[newID]; exists {
		return // Element already exists
	}

	afterElem, ok := r.elements[afterID]
	if !ok {
		afterElem = r.head // fallback to head if not found
	}

	nextElem := afterElem.Next

	elem := RGAElement{
		ID:        newID,
		Value:     value,
		Timestamp: ts,
		Prev:      afterElem,
		Next:      nextElem,
		Deleted:   false,
	}

	afterElem.Next = &elem

	if nextElem != nil {
		nextElem.Prev = &elem
	}

	r.elements[newID] = &elem
	r.order = append(r.order, &elem)
}

func (r *RGA) Delete(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if elem, exists := r.elements[id]; exists {
		elem.Deleted = true
	}
}

func (r *RGA) GetDocument() []VisibleElement {
	r.mu.Lock()
	defer r.mu.Unlock()

	var document []VisibleElement
	for elem := r.head.Next; elem != nil && elem != r.tail; elem = elem.Next {
		if !elem.Deleted {
			document = append(document, VisibleElement{
				ID:    elem.ID,
				Value: elem.Value,
			})
		}
	}
	return document
}

func (r *RGA) SaveToFile(filename string) error {
	document := r.GetDocument()
	data, err := json.Marshal(document)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func LoadFromFile(filename string) (*RGA, error) {
	if _, err := os.Stat(filename); err == nil {
		data, err := os.ReadFile(filename)
		if err != nil {
			return nil, err
		}

		var document []string
		if err := json.Unmarshal(data, &document); err != nil {
			return nil, err
		}

		rgaDoc := NewRGA()
		prevID := "head"
		for i, char := range document {
			id := "init:" + string(i)
			rgaDoc.Insert(prevID, id, char, time.Now())
			prevID = id
		}
		return rgaDoc, nil
	}

	return NewRGA(), nil
}
