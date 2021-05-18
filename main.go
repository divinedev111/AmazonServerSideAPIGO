package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/go-resty/resty/v2"

	"github.com/PuerkitoBio/goquery"
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

func monitorold(asin string) *string {
	for true {
		url := "https://www.amazon.com/portal-migration/aod?asin=" + asin

		client := resty.New()
		resp, err := client.R().
			SetHeaders(map[string]string{
				"authority":                 "www.amazon.com",
				"pragma":                    "no-cache",
				"cache-control":             "no-cache",
				"rtt":                       "0",
				"downlink":                  "10",
				"ect":                       "4g",
				"sec-ch-ua":                 "' Not A;Brand';v='99', 'Chromium';v='90', 'Google Chrome';v='90'",
				"sec-ch-ua-mobile":          "?0",
				"upgrade-insecure-requests": "1",
				"user-agent":                "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.93 Safari/537.36",
				"accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
				"sec-fetch-site":            "none",
				"sec-fetch-mode":            "navigate",
				"sec-fetch-user":            "?1",
				"sec-fetch-dest":            "document",
				"accept-language":           "en-US,en;q=0.9",
			}).
			SetContentLength(true).
			SetDoNotParseResponse(true).Get(url)

		doc, err := goquery.NewDocumentFromReader(resp.RawBody())
		if err != nil {
			fmt.Println(err)
			return nil
		}
		var offerID string
		doc.Find(".a-fixed-right-grid-col").Each(func(i int, s *goquery.Selection) {
			price, _ := s.Find(".a-button-input").Attr("aria-label")
			stock := strings.Contains(price, "Add to Cart from seller Amazon.com")
			if stock == true {
				offerID, _ = s.Find("input[name='offeringID.1']").Attr("value")
				return
			}
		})
		fmt.Println(offerID)
	}
	return nil
}

type Proxy struct {
	IP   string
	Port string
	User string
	Pass string
}

var pxyList []string

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
	f, err := os.Open("proxy.txt")
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		pxyList = append(pxyList, scanner.Text())
	}

	port := getPort()

	go monitor()

	http.HandleFunc("/", home)
	http.HandleFunc("/amazon", AzClient)
	log.Println("listening on ", port)
	log.Fatal(http.ListenAndServe((":" + port), nil))
}
