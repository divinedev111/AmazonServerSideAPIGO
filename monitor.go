package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	tls "github.com/refraction-networking/utls"
	"golang.org/x/net/proxy"
)

type ATCresp struct {
	CartSubtotalString       *string   `json:"cartSubtotalString"`
	FormattedTotalPrice      *string   `json:"formattedTotalPrice"`
	IncludedAsins            []*string `json:"includedAsins"`
	ItemQuantity             *string   `json:"itemQuantity"`
	ItemQuantityString       *string   `json:"itemQuantityString"`
	TotalPrice               *string   `json:"totalPrice"`
	TotalPriceInBaseCurrency *string   `json:"totalPriceInBaseCurrency"`
}

type product struct {
	Asin    string `json:"asin"`
	OfferID string `json:"offer_id"`
	Name    string `json:"product_name"`
}

func monitorMobileATC() error {

	var products []product
	//f, err := os.Open("test.json")
	f, err := os.Open("amazonskus.json")
	if err != nil {
		return err
	}
	defer f.Close()

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	config := tls.Config{}

	json.Unmarshal(bytes, &products)

	x := 0
	for _, prod := range products {
		Pindex := (len(pxyList) / len(products) * x)
		//Lindex := 0

		x++

		go func(prod product, i int) {
			ticker := time.NewTicker(time.Millisecond * time.Duration(1700))
			createSession := true
			client := resty.New()

			for {
				select {
				case <-ticker.C:
					go func() {

						pxy := pxyList[i]
						i = (i + 1) % len(pxyList)

						if prod.OfferID == "" {
							return
						}

						pxyParts := strings.Split(pxy, ":")
						pURL, _ := url.Parse("http://" + pxyParts[2] + ":" + pxyParts[3] + "@" + pxyParts[0] + ":" + pxyParts[1])
						p, err := FromURL(pURL, proxy.Direct)

						tr, err := NewTransportWithDialer("771,49200-49195-61-157-49172-49196-49171-60-156-49199-49192-49188-49162-165-163-161-159-107-106-105-104-57-56-55-54-49202-49198-49194-49190-49167-49157-53-49191-49187-49161-164-162-160-158-103-64-63-62-51-50-49-48-49201-49197-49193-49189-49166-49156-47-154-153-152-151-150-10-255,0-11-10-35-13-5-21,23-25-28-27-24-26-22-14-13-11-12-9-10,0-1-2", &config, p)

						client.SetTransport(tr)

						if createSession {
							client = resty.New()
							client.SetTransport(tr)
							client.Cookies = nil
							log.Println("GENERATING NEW SESSION")
							_, err := client.R().
								SetHeaders(map[string]string{
									"accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
									"accept-language":           "en-US,en;q=0.9",
									"cache-control":             "no-cache",
									"pragma":                    "no-cache",
									"sec-ch-ua":                 "\" Not;A Brand\";v=\"99\", \"Google Chrome\";v=\"91\", \"Chromium\";v=\"91\"",
									"sec-ch-ua-mobile":          "?0",
									"sec-fetch-dest":            "document",
									"sec-fetch-mode":            "navigate",
									"sec-fetch-site":            "none",
									"sec-fetch-user":            "?1",
									"upgrade-insecure-requests": "1",
									"user-agent":                "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.77 Safari/537.36",
								}).
								Get("https://smile.amazon.com/gp/mobile/udp/ajax-handlers/reftag.html?ref_=dp_atch_abb_i%22")

							if err != nil {
								log.Println(prod.Asin, err)
								return
							}

							_, err = client.R().
								SetHeaders(map[string]string{
									"accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
									"accept-language":           "en-US,en;q=0.9",
									"cache-control":             "no-cache",
									"pragma":                    "no-cache",
									"sec-ch-ua":                 "\" Not;A Brand\";v=\"99\", \"Google Chrome\";v=\"91\", \"Chromium\";v=\"91\"",
									"sec-ch-ua-mobile":          "?0",
									"sec-fetch-dest":            "document",
									"sec-fetch-mode":            "navigate",
									"sec-fetch-site":            "none",
									"sec-fetch-user":            "?1",
									"upgrade-insecure-requests": "1",
									"user-agent":                "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.77 Safari/537.36",
								}).
								Get("https://smile.amazon.com/gp/mobile/udp/ajax-handlers/reftag.html?ref_=dp_atch_abb_i%22")

							if err != nil {
								log.Println(prod.Asin, err)
								return
							}
							createSession = false
						}

						var sid string
						for i := range client.Cookies {
							if client.Cookies[i].Name == "session-id" {
								sid = client.Cookies[i].Value
							}
						}

						body := fmt.Sprintf(`marketplaceId=ATVPDKIKX0DER&asin=B005QIYL7E&customerId=&sessionId=%v&accessoryItemAsin=B002M40VJM&accessoryItemOfferingId=%v&languageOfPreference=en_US&accessoryItemQuantity=1&accessoryItemPrice=9.99&accessoryMerchantId=ATVPDKIKX0DER&accessoryProductGroupId=8652000`, sid, prod.OfferID)

						resp, err := client.R().
							SetHeaders(map[string]string{
								"accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
								"accept-encoding":           "gzip, deflate, br",
								"accept-language":           "en-US,en;q=0.9",
								"content-type":              "application/x-www-form-urlencoded",
								"cache-control":             "max-age=0",
								"downlink":                  "10",
								"ect":                       "4g",
								"rtt":                       "50",
								"sec-ch-ua":                 "\" Not A;Brand\";v=\"99\", \"Chromium\";v=\"90\", \"Google Chrome\";v=\"90\"",
								"sec-ch-ua-mobile":          "?0",
								"sec-fetch-dest":            "empty",
								"sec-fetch-mode":            "cors",
								"sec-fetch-site":            "same-origin",
								"sec-fetch-user":            "?1",
								"upgrade-insecure-requests": "1",
								"x-requested-with":          "XHMHttpRequest",
								"user-agent":                "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.77 Safari/537.36",
							}).
							SetContentLength(true).
							SetBody(body).
							Post("https://smile.amazon.com/gp/product/features/aloha-ppd/udp-ajax-handler/attach-add-to-cart.html")

						if err != nil {
							log.Println(err)
							return
						}

						log.Println(resp.StatusCode(), prod.Asin, resp.Time(), resp.Size())
						if resp.StatusCode() == 400 {
							createSession = true
						}

						data := ATCresp{}
						json.Unmarshal(resp.Body(), &data)

						if data.IncludedAsins != nil {
							for i := range data.IncludedAsins {
								if *data.IncludedAsins[i] == prod.Asin {
									log.Println("FOUND:", prod.Asin, prod.Name)
									broadcast.Send(SendData{
										SKU:     prod.Asin,
										OfferID: prod.OfferID,
										Price:   *data.TotalPrice,
									})
								}
							}
						}
					}()
				}
			}
		}(prod, Pindex)
	}
	return nil
}
