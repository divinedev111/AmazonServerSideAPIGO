package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
	tls "github.com/refraction-networking/utls"
	"golang.org/x/net/proxy"
)

func test() {

	Init()
	var tr http.Transport
	config := tls.Config{
		InsecureSkipVerify: true,
	}

	f, err := os.Open("./proxy.txt")
	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		time.Sleep(time.Millisecond * time.Duration(200))
		pxy := scanner.Text()
		pxyParts := strings.Split(pxy, ":")

		pURL, _ := url.Parse("http://" + pxyParts[2] + ":" + pxyParts[3] + "@" + pxyParts[0] + ":" + pxyParts[1])
		p, err := FromURL(pURL, proxy.Direct)

		trans, err := NewTransportWithDialer("771,255-49195-49199-49196-49200-49171-49172-156-157-47-53,0-10-11-13,23-24,0", &config, p)
		if err != nil {
			log.Fatal(err)
		}

		tr = *trans

		client := resty.New()
		client.SetTransport(&http.Transport{
			Proxy: http.ProxyURL(&url.URL{
				Scheme: "http",
				User:   url.UserPassword(pxyParts[2], pxyParts[3]),
				Host:   pxyParts[0] + ":" + pxyParts[1],
			}),
		})
		client.SetTransport(&tr)

		resp, err := client.R().
			EnableTrace().
			SetHeaders(map[string]string{
				"accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
				"accept-language":           "en-US,en;q=0.9,ur-IN;q=0.8,ur-PK;q=0.7,ur;q=0.6,ar-SA;q=0.5,ar;q=0.4",
				"cache-control":             "max-age=0",
				"downlink":                  "10",
				"ect":                       "4g",
				"rtt":                       "50",
				"sec-ch-ua":                 "\" Not A;Brand\";v=\"99\", \"Chromium\";v=\"90\", \"Google Chrome\";v=\"90\"",
				"sec-ch-ua-mobile":          "?0",
				"sec-fetch-dest":            "document",
				"sec-fetch-mode":            "navigate",
				"sec-fetch-site":            "none",
				"sec-fetch-user":            "?1",
				"upgrade-insecure-requests": "1",
			}).
			SetDoNotParseResponse(true).
			Get("https://ok54crjodozxdxppjctwis33ha-b25orc35okxta-www-amazon-com.translate.goog/portal-migration/aod?asin=B08WM28PVH")

		doc, err := goquery.NewDocumentFromReader(resp.RawBody())
		if err != nil {
			log.Fatal(err)
		}

		title := doc.Find("title").Text()
		log.Println(title)
		txt, _ := doc.Html()
		if strings.Contains(txt, "dogs of Amazon") {
			log.Println("FAILED")
		}
	}
}

// greasePlaceholder is a random value (well, kindof '0x?a?a) specified in a
// random RFC.
const greasePlaceholder = 0x0a0a

// ErrExtensionNotExist is returned when an extension is not supported by the library
type ErrExtensionNotExist string

// Error is the error value which contains the extension that does not exist
func (e ErrExtensionNotExist) Error() string {
	return fmt.Sprintf("Extension does not exist: %s\n", e)
}

type Dialer interface {
	Dial(network, addr string) (net.Conn, error)
}

// extMap maps extension values to the TLSExtension object associated with the
// number. Some values are not put in here because they must be applied in a
// special way. For example, "10" is the SupportedCurves extension which is also
// used to calculate the JA3 signature. These JA3-dependent values are applied
// after the instantiation of the map.
var extMap = map[string]tls.TLSExtension{
	"0": &tls.SNIExtension{},
	"5": &tls.StatusRequestExtension{},
	// These are applied later
	// "10": &tls.SupportedCurvesExtension{...}
	// "11": &tls.SupportedPointsExtension{...}
	"13": &tls.SignatureAlgorithmsExtension{
		SupportedSignatureAlgorithms: []tls.SignatureScheme{
			tls.ECDSAWithP256AndSHA256,
			tls.PSSWithSHA256,
			tls.PKCS1WithSHA256,
			tls.ECDSAWithP384AndSHA384,
			tls.PSSWithSHA384,
			tls.PKCS1WithSHA384,
			tls.PSSWithSHA512,
			tls.PKCS1WithSHA512,
			tls.PKCS1WithSHA1,
		},
	},
	"16": &tls.ALPNExtension{
		AlpnProtocols: []string{"h2", "http/1.1"},
	},
	"18": &tls.SCTExtension{},
	"21": &tls.UtlsPaddingExtension{GetPaddingLen: tls.BoringPaddingStyle},
	"23": &tls.UtlsExtendedMasterSecretExtension{},
	"27": &tls.FakeCertCompressionAlgsExtension{},
	"28": &tls.FakeRecordSizeLimitExtension{},
	"35": &tls.SessionTicketExtension{},
	"43": &tls.SupportedVersionsExtension{[]uint16{
		tls.GREASE_PLACEHOLDER,
		tls.VersionTLS13,
		tls.VersionTLS12,
		tls.VersionTLS11,
		tls.VersionTLS10}},
	"44": &tls.CookieExtension{},
	"45": &tls.PSKKeyExchangeModesExtension{[]uint8{
		tls.PskModeDHE,
	}},
	"51":    &tls.KeyShareExtension{[]tls.KeyShare{}},
	"13172": &tls.NPNExtension{},
	"65281": &tls.RenegotiationInfoExtension{
		Renegotiation: tls.RenegotiateOnceAsClient,
	},
}

// NewTransport creates an http.Transport which mocks the given JA3 signature when HTTPS is used
func NewTransport(ja3 string) (*http.Transport, error) {
	return NewTransportWithConfig(ja3, &tls.Config{})
}

// NewTransportWithConfig creates an http.Transport object given a utls.Config
func NewTransportWithConfig(ja3 string, config *tls.Config) (*http.Transport, error) {
	spec, err := stringToSpec(ja3)
	if err != nil {
		return nil, err
	}

	dialtls := func(network, addr string) (net.Conn, error) {
		dialConn, err := net.Dial(network, addr)
		if err != nil {
			return nil, err
		}

		config.ServerName = strings.Split(addr, ":")[0]

		uTlsConn := tls.UClient(dialConn, config, tls.HelloCustom)
		if err := uTlsConn.ApplyPreset(spec); err != nil {
			return nil, err
		}
		if err := uTlsConn.Handshake(); err != nil {
			return nil, err
		}
		return uTlsConn, nil
	}

	return &http.Transport{DialTLS: dialtls}, nil
}

// stringToSpec creates a ClientHelloSpec based on a JA3 string
func stringToSpec(ja3 string) (*tls.ClientHelloSpec, error) {
	tokens := strings.Split(ja3, ",")

	version := tokens[0]
	ciphers := strings.Split(tokens[1], "-")
	extensions := strings.Split(tokens[2], "-")
	curves := strings.Split(tokens[3], "-")
	if len(curves) == 1 && curves[0] == "" {
		curves = []string{}
	}
	pointFormats := strings.Split(tokens[4], "-")
	if len(pointFormats) == 1 && pointFormats[0] == "" {
		pointFormats = []string{}
	}

	// parse curves
	var targetCurves []tls.CurveID
	for _, c := range curves {
		cid, err := strconv.ParseUint(c, 10, 16)
		if err != nil {
			return nil, err
		}
		targetCurves = append(targetCurves, tls.CurveID(cid))
	}
	extMap["10"] = &tls.SupportedCurvesExtension{targetCurves}

	// parse point formats
	var targetPointFormats []byte
	for _, p := range pointFormats {
		pid, err := strconv.ParseUint(p, 10, 8)
		if err != nil {
			return nil, err
		}
		targetPointFormats = append(targetPointFormats, byte(pid))
	}
	extMap["11"] = &tls.SupportedPointsExtension{SupportedPoints: targetPointFormats}

	// build extenions list
	var exts []tls.TLSExtension
	for _, e := range extensions {
		te, ok := extMap[e]
		if !ok {
			return nil, ErrExtensionNotExist(e)
		}
		exts = append(exts, te)
	}
	// build SSLVersion
	vid64, err := strconv.ParseUint(version, 10, 16)
	if err != nil {
		return nil, err
	}
	vid := uint16(vid64)

	// build CipherSuites
	var suites []uint16
	for _, c := range ciphers {
		cid, err := strconv.ParseUint(c, 10, 16)
		if err != nil {
			return nil, err
		}
		suites = append(suites, uint16(cid))
	}

	return &tls.ClientHelloSpec{
		TLSVersMin:         vid,
		TLSVersMax:         vid,
		CipherSuites:       suites,
		CompressionMethods: []byte{0},
		Extensions:         exts,
		GetSessionID:       sha256.Sum256,
	}, nil
}

func urlToHost(target *url.URL) *url.URL {
	if !strings.Contains(target.Host, ":") {
		if target.Scheme == "http" {
			target.Host = target.Host + ":80"
		} else if target.Scheme == "https" {
			target.Host = target.Host + ":443"
		}
	}
	return target
}

// NewTransportWithConfig - creates an http.Transport object given a utls.Config
func NewTransportWithDialer(ja3 string, config *tls.Config, dialer Dialer) (*http.Transport, error) {

	dialtls := func(network, addr string) (net.Conn, error) {
		dialConn, err := dialer.Dial(network, addr)
		if err != nil {
			return nil, err
		}

		spec, err := stringToSpec(ja3)
		if err != nil {
			return nil, err
		}
		uTlsConn := tls.UClient(dialConn, config, tls.HelloCustom)
		if err := uTlsConn.ApplyPreset(spec); err != nil {
			return nil, err
		}

		uTlsConn.SetSNI(config.ServerName)
		if err := uTlsConn.Handshake(); err != nil {
			return nil, err
		}
		return uTlsConn, nil
	}

	return &http.Transport{DialTLS: dialtls, Dial: dialer.Dial}, nil
}

type direct struct{}

// Direct is a direct proxy: one that makes network connections directly.
var Direct = direct{}

//func (direct) Dial(network, addr string) (net.Conn, error) {
//	return net.Dial(network, addr)
//}

// httpsDialer
type httpsDialer struct{}

// HTTPSDialer is a https proxy: one that makes network connections on tls.
var HttpsDialer = httpsDialer{}
var TlsConfig = &tls.Config{}

func (d httpsDialer) Dial(network, addr string) (c net.Conn, err error) {
	c, err = tls.Dial("tcp", addr, TlsConfig)
	if err != nil {
		fmt.Println(err)
	}
	return
}

// httpProxy is a HTTP/HTTPS connect proxy.
type httpProxy struct {
	host     string
	haveAuth bool
	username string
	password string
	forward  proxy.Dialer
}

func newHTTPProxy(uri *url.URL, forward proxy.Dialer) (proxy.Dialer, error) {
	s := new(httpProxy)
	s.host = uri.Host
	s.forward = forward
	if uri.User != nil {
		s.haveAuth = true
		s.username = uri.User.Username()
		s.password, _ = uri.User.Password()
	}

	return s, nil
}

func (s *httpProxy) Dial(network, addr string) (net.Conn, error) {
	// Dial and create the https client connection.
	c, err := s.forward.Dial("tcp", s.host)
	if err != nil {
		return nil, err
	}

	// HACK. http.ReadRequest also does this.
	reqURL, err := url.Parse("http://" + addr)
	if err != nil {
		c.Close()
		return nil, err
	}
	reqURL.Scheme = ""

	req, err := http.NewRequest("CONNECT", reqURL.String(), nil)
	if err != nil {
		c.Close()
		return nil, err
	}
	req.Close = false
	if s.haveAuth {
		req.SetBasicAuth(s.username, s.password)
		auth := s.username + ":" + s.password
		basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
		req.Header.Add("Proxy-Authorization", basicAuth)
	}

	err = req.Write(c)
	if err != nil {
		c.Close()
		return nil, err
	}

	resp, err := http.ReadResponse(bufio.NewReader(c), req)
	if err != nil {
		// TODO close resp body ?
		if resp != nil {
			resp.Body.Close()
		}
		c.Close()
		return nil, err
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		c.Close()
		err = fmt.Errorf("Connect server using proxy error u, StatusCode [%d]", resp.StatusCode)
		return nil, err
	}

	return c, nil
}

func FromURL(u *url.URL, forward proxy.Dialer) (proxy.Dialer, error) {
	return proxy.FromURL(u, forward)
}

func FromURLnil(u *url.URL) (proxy.Dialer, error) {
	return proxy.FromURL(u, proxy.Direct)
}

func FromEnvironment() proxy.Dialer {
	return proxy.FromEnvironment()
}

func Init() {
	proxy.RegisterDialerType("http", newHTTPProxy)
	proxy.RegisterDialerType("https", newHTTPProxy)
}
