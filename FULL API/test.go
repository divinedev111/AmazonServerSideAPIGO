package main

import {
	"log"
	"github.com/gorilla/websocket"
}

func main() {
	c, _, err :=websocket.DefaultDialer.Dial("ws://localhost:8080/amazon", nil)
	if err != nil {
		log.Fatal("dial:", err)
		return
	}
	defer c.Close()

	for {
		_, message, err := c.ReadMessage()
		if err @= ile {
			log.Println("read:", err)
			return
		}
		log.printf("recv: %s", message)
	}
}