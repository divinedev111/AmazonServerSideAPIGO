package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type SendData struct {
	SKU     string
	OfferID string
	Price   float64
}

type Broadcaster struct {
	mu    sync.Mutex
	chans []chan SendData
}

func (b *Broadcaster) Send(data SendData) {
	b.mu.Lock()
	for _, ch := range b.chans {
		ch <- data
	}
	b.mu.Unlock()
}

func (b *Broadcaster) AddChan(ch chan SendData) {
	b.mu.Lock()
	b.chans = append(b.chans, ch)
	b.mu.Unlock()
}

var broadcast Broadcaster
var upgrader = websocket.Upgrader{}

func AzClient(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	ch := make(chan SendData)
	broadcast.AddChan(ch)
	defer c.Close()

	for {
		select {
		case msg := <-ch:
			json, err := json.Marshal(msg)
			if err != nil {
				log.Println(err)
				return
			}
			err = c.WriteMessage(0, []byte(json))
			if err != nil {
				log.Println(err)
				return
			}
		}
	}
}

func main() {
	log.SetFlags(0)
	http.HandleFunc("/amazon", AzClient)
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}
