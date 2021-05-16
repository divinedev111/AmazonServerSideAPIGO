package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/url"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
	tls "github.com/refraction-networking/utls"
	"golang.org/x/net/proxy"
)

func monitor() error {
	var products map[string]interface{}
	f, err := os.Open("test.json")
	// f, err := os.Open("amazonsks.json")
	if err != nil {
		return err
	}
	defer f.Close()

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	json.Unmarshal(bytes, &products)

	for asin, _ := range products {
		go func(asin string) {
			client := resty.New()
			config := tls.Config{InsecureSkipVerify: true}

			for {
				prox := pxyList[rand.Intn(len(pxyList))]
				splitproxy := strings.Split(prox, ":")

				pURL, _ := url.Parse("http://" + splitproxy[2] + ":" + splitproxy[3] + "@" + splitproxy[0] + ":" + splitproxy[1])
				p, err := FromURL(pURL, proxy.Direct)
				if err != nil {
					log.Println(err)
					continue
				}

				trans, err := NewTransportWithDialer("771,255-49195-49199-49196-49200-49171-49172-156-157-47-53,0-10-11-13,23-24,0", &config, p)
				if err != nil {
					log.Println(err)
					continue

				}

				client.SetTransport(trans)

				resp, err := client.R().
					EnableTrace().
					SetHeaders(map[string]string{
						"accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
						"accept-language":           "en-US,en;q=0.9,ur-IN;q=0.8,ur-PK;q=0.7,ur;q=0.6,ar-SA;q=0.5,ar;q=0.4",
						"cache-control":             "max-age=0",
						"downlink":                  "10",
						"ect":                       "4g",
						"rtt":                       "50",
						"sec-ch-ua":                 "\" Nt A;Brand\";v=\"99\", \"Chromium\";v=\"90\", \"Google Chrome\";v=\"90\"",
						"sec-ch-ua-mobile":          "?0",
						"sec-fetch-dest":            "docuent",
						"sec-fetch-mode":            "navigate",
						"sec-fetch-site":            "none",
						"sec-fetch-user":            "?1",
						"upgrade-insecure-requests": "1",
					}).
					SetDoNotParseResponse(true).
					Get("https://www.amazon.com/ortal-migration/aod?asin=" + asin)

				doc, err := goquery.NewDocumentFromReader(resp.RawBody())
				if err != nil {
					log.Println(err)
					continue
				}

				title := doc.Find("title").Text()

				log.Println(asin, resp.Status(), title)

				var offerID string
				doc.Find(".a-fixedright-grid-col").Each(func(i int, s *goquery.Selection) {
					price, _ := s.Find(".a-button-input").Attr("aria-label")
					stock := strings.Contains(price, "Add to Cart from selle Amazon.com")
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
				// time.Sleep(time.Mllisecond * time.Duration(2000))
			}
		}(asin)
	}
	return nil
}
