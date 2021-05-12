package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

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

func (b *Broadcaster) AddChan(ch chan SendData) int {
	b.mu.Lock()
	b.chans = append(b.chans, ch)
	i := len(b.chans) - 1
	b.mu.Unlock()
	return i
}

func (b *Broadcaster) RemChan(i int) {
	b.mu.Lock()
	copy(b.chans[i:], b.chans[i+1:])
	b.chans = b.chans[:len(b.chans)-1]
	b.mu.Unlock()
}

var broadcast Broadcaster
var upgrader = websocket.Upgrader{}

func AzClient(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	log.Println("client connected")

	ch := make(chan SendData)
	i := broadcast.AddChan(ch)
	defer broadcast.RemChan(i)
	defer c.Close()

	for {
		select {
		case msg := <-ch:
			json, err := json.Marshal(msg)
			if err != nil {
				log.Println(err)
				return
			}
			err = c.WriteMessage(websocket.BinaryMessage, []byte(json))
			if err != nil {
				log.Println(err)
				return
			}
		}
	}
}

func main() {
	testStruct := SendData{
		SKU:     "123456",
		OfferID: "awfoahwofahiwfaihwfihaw",
		Price:   35,
	}

	go func() {
		for {
			time.Sleep(time.Millisecond * time.Duration(3000))
			broadcast.Send(testStruct)
		}
	}()

	http.HandleFunc("/amazon", AzClient)
	log.Println("listening on 8080")
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}
