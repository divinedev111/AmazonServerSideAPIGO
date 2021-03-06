package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"
)

type SendData struct {
	SKU     string
	OfferID string
	Price   string
}

type Broadcaster struct {
	mu    sync.Mutex
	chans []chan SendData
}

func (b *Broadcaster) Send(data SendData) {
	fmt.Println("sending")
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

func home(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "working")
	return
}

var pxyList []string
var links []string

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}
	return port
}

func main() {
	Init()
	err := os.MkdirAll("./logs", 0700)
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.Open("proxy.txt")
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		pxyList = append(pxyList, scanner.Text())
	}

	f, err = os.Open("links.txt")
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()

	scanner = bufio.NewScanner(f)
	for scanner.Scan() {
		links = append(links, scanner.Text())
	}

	port := getPort()

	go monitorMobileATC()

	http.HandleFunc("/", home)
	http.HandleFunc("/amazon", AzClient)
	log.Println("listening on ", port)
	log.Fatal(http.ListenAndServe((":" + port), nil))
}
