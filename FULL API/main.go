package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	tls "github.com/refraction-networking/utls"

	"github.com/go-resty/resty/v2"
	"github.com/x04/cclient"

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

func monitor(asin string) *string {
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

func stringToProxy(line string) (Proxy, error) {

	parts := strings.Split(line, ":")

	if len(parts) == 2 {
		return Proxy{parts[0], parts[1], "", ""}, nil

	} else if len(parts) == 4 {
		return Proxy{parts[0], parts[1], parts[2], parts[3]}, nil

	} else {
		return Proxy{"", "", "", ""}, errors.New("Error parsing proxy")
	}
}

func LoadProxies(proxyStr string) ([]Proxy, error) {
	proxyArr := strings.Split(proxyStr, "\n")
	var proxies []Proxy
	for _, v := range proxyArr {

		proxy, err := stringToProxy(v)

		if err == nil {
			proxies = append(proxies, proxy)
		}
	}

	return proxies, nil
}

func main() {
	var pxyList []string
	f, err := os.Open("proxy.txt")
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		pxyList = append(pxyList, scanner.Text())
	}
	skus := []string{"B07ZPC9QD4", "B08P3ZN62G"}
	for key := range skus {
		go func(asin string) *string {
			for {
				prox := pxyList[rand.Intn(len(pxyList))]
				splitproxy := strings.Split(prox, ":")
				// fmt.Println(splitproxy[0])
				// fmt.Println(splitproxy[1])
				// fmt.Println(splitproxy[2])
				// fmt.Println(splitproxy[3])
				proxyurl := "http://" + splitproxy[2] + ":" + splitproxy[3] + "@" + splitproxy[0] + ":" + splitproxy[1]
				url := "https://www.amazon.com/portal-migration/aod?asin=" + asin
				client, err := cclient.NewClient(tls.HelloChrome_Auto, proxyurl)
				if err != nil {
					log.Fatal(err)
				}
				req, err := client.Get(url)
				if err != nil {
					log.Fatal(err)
				}
				// client.SetTransport(&http.Transport{
				// 	Proxy: http.ProxyURL(&url.URL{
				// 		Scheme: "http",
				// 		User:   url.UserPassword(splitproxy[2], splitproxy[3]),
				// 		Host:   splitproxy[0] + ":" + splitproxy[1],
				// 	}),
				// })
				// client.SetProxy("http://" + prox)
				// resp, err := client.R().
				// 	SetHeaders(map[string]string{
				// 		"authority":                 "www.amazon.com",
				// 		"pragma":                    "no-cache",
				// 		"cache-control":             "no-cache",
				// 		"rtt":                       "0",
				// 		"downlink":                  "10",
				// 		"ect":                       "4g",
				// 		"sec-ch-ua":                 "' Not A;Brand';v='99', 'Chromium';v='90', 'Google Chrome';v='90'",
				// 		"sec-ch-ua-mobile":          "?0",
				// 		"upgrade-insecure-requests": "1",
				// 		"user-agent":                "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.93 Safari/537.36",
				// 		"accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
				// 		"sec-fetch-site":            "none",
				// 		"sec-fetch-mode":            "navigate",
				// 		"sec-fetch-user":            "?1",
				// 		"sec-fetch-dest":            "document",
				// 		"accept-language":           "en-US,en;q=0.9",
				// 	}).
				// 	SetContentLength(true).
				// 	SetDoNotParseResponse(true).Get(url)
				req.Header.Add("authority", "www.amazon.com")
				req.Header.Add("pragma", "no-cache")
				req.Header.Add("cache-control", "no-cache")
				req.Header.Add("rtt", "0")
				req.Header.Add("downlink", "10")
				req.Header.Add("ect", "4g")
				req.Header.Add("sec-ch-ua", "\" Not A;Brand\";v=\"99\", \"Chromium\";v=\"90\", \"Google Chrome\";v=\"90\"")
				req.Header.Add("sec-ch-ua-mobile", "?0")
				req.Header.Add("upgrade-insecure-requests", "1")
				req.Header.Add("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.93 Safari/537.36")
				req.Header.Add("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
				req.Header.Add("sec-fetch-site", "none")
				req.Header.Add("sec-fetch-mode", "navigate")
				req.Header.Add("sec-fetch-user", "?1")
				req.Header.Add("sec-fetch-dest", "document")
				req.Header.Add("accept-language", "en-US,en;q=0.9")
				doc, err := goquery.NewDocumentFromReader(req.Body)
				print(doc.Html())
				if err != nil {
					fmt.Println(err)
				}
				var offerID string
				doc.Find(".a-fixed-right-grid-col").Each(func(i int, s *goquery.Selection) {
					price, _ := s.Find(".a-button-input").Attr("aria-label")
					stock := strings.Contains(price, "Add to Cart from seller Amazon.com")
					if stock == true {
						offerID, _ = s.Find("input[name='offeringID.1']").Attr("value")
						testStruct := SendData{
							SKU:     asin,
							OfferID: offerID,
							Price:   price,
						}
						broadcast.Send(testStruct)
						return
					}
				})
				fmt.Println(offerID)
				time.Sleep(2000)
			}
		}(skus[key])
	}
	http.HandleFunc("/amazon", AzClient)
	log.Println("listening on 8080")
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}
