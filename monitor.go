package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
)

func monitor() error {
	var products map[string]interface{}
	f, err := os.Open("amazonskus.json")
	//f, err := os.Open("test.json")
	if err != nil {
		return err
	}
	defer f.Close()

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	json.Unmarshal(bytes, &products)

	x := 0

	for asin, _ := range products {
		i := (len(pxyList) / len(products) * x)
		x++
		go func(asin string, i int) {
			client := resty.New()
			//config := tls.Config{InsecureSkipVerify: true}

			for {
				prox := pxyList[i]
				i = (i + 1) % len(pxyList)
				pxyParts := strings.Split(prox, ":")

				link := links[rand.Intn(len(links))]

				//pURL, _ := url.Parse("http://" + splitproxy[2] + ":" + splitproxy[3] + "@" + splitproxy[0] + ":" + splitproxy[1])
				//p, err := FromURL(pURL, proxy.Direct)
				//if err != nil {
				//	log.Println(err)
				//	continue
				//}

				//trans, err := NewTransportWithDialer("771,255-49195-49199-49196-49200-49171-49172-156-157-47-53,0-10-11-13,23-24,0", &config, p)
				//if err != nil {
				//	log.Println(err)
				//	continue
				//}

				client.SetTransport(&http.Transport{
					Proxy: http.ProxyURL(&url.URL{
						Scheme: "http",
						User:   url.UserPassword(pxyParts[2], pxyParts[3]),
						Host:   pxyParts[0] + ":" + pxyParts[1],
					}),
				})

				resp, err := client.R().
					SetDoNotParseResponse(true).
					Get(link + asin)

				if err != nil {
					log.Println(err)
					time.Sleep(time.Millisecond * time.Duration(500))
					continue
				}

				var doc *goquery.Document
				if resp != nil {
					doc, err = goquery.NewDocumentFromReader(resp.RawBody())
					if err != nil {
						log.Println(err)
						continue
					}
					resp.RawBody().Close()
				}

				title := doc.Find("title").Text()

				log.Println(asin, resp.Status(), title)

				var offerID string
				data := SendData{}
				var found bool = false
				doc.Find(".a-fixed-right-grid-col").Each(func(i int, s *goquery.Selection) {
					price, _ := s.Find(".a-button-input").Attr("aria-label")
					stock := strings.Contains(price, "Add to Cart from seller Amazon.com")
					if stock == true {
						offerID, _ = s.Find("input[name='offeringID.1']").Attr("value")
						data.OfferID = offerID
						data.SKU = asin
						data.Price = price
						found = true
						return
					}
				})
				if found {
					fileLog.Println("FOUND: ", products[asin])
					log.Println("FOUND: ", products[asin])
					broadcast.Send(data)
				}
				//fmt.Println(offerID)
				time.Sleep(time.Millisecond * time.Duration(500))
			}
		}(asin, i)
	}
	return nil
}
