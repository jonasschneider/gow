package main

import (
	"io"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

// headers to drop
var hopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te", // canonicalized version of "TE"
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",

	// added by me
	"Sec-Websocket-Accept",
}

type BackendSelector interface {
	Select(requestHost string) (string, error)
}

func ListenAndServeHTTP(address string, sel BackendSelector) error {
	proxyHandler := http.HandlerFunc(makeProxyHandlerFunc(sel))
	return http.ListenAndServe(address, proxyHandler)
}

func makeProxyHandlerFunc(sel BackendSelector) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		backend, err := sel.Select(r.Host)

		if err == nil {
			proxyRequest(w, r, backend)
		} else {
			writeErrorPage(w, err)
		}
	}
}

func proxyRequest(w http.ResponseWriter, r *http.Request, backendAddress string) {
	r.RequestURI = ""

	if r.Header["Connection"] != nil && r.Header["Connection"][0] == "Upgrade" &&
		r.Header["Upgrade"] != nil && r.Header["Upgrade"][0] == "websocket" {
		proxyWebsocket(w, r, backendAddress)
		return
	}

	r.URL.Scheme = "http"
	r.URL.Host = backendAddress

	resp, err := http.DefaultTransport.RoundTrip(r)

	if err != nil {
		writeErrorPage(w, err)
		return
	}

	writeResponseHeader(w, resp)

	// just stream the body to the client
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Println(err)
	}
}

// Portions from https://github.com/koding/websocketproxy
var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var dialer = websocket.DefaultDialer

func proxyWebsocket(w http.ResponseWriter, req *http.Request, backendAddress string) {
	backendURL := *req.URL
	backendURL.Scheme = "ws"
	backendURL.Host = backendAddress

	// Pass headers from the incoming request to the dialer to forward them to
	// the final destinations.
	requestHeader := http.Header{}
	requestHeader.Add("Origin", req.Header.Get("Origin"))
	for _, prot := range req.Header[http.CanonicalHeaderKey("Sec-WebSocket-Protocol")] {
		requestHeader.Add("Sec-WebSocket-Protocol", prot)
	}
	for _, cookie := range req.Header[http.CanonicalHeaderKey("Cookie")] {
		requestHeader.Add("Cookie", cookie)
	}

	// Pass X-Forwarded-For headers too, code below is a part of
	// httputil.ReverseProxy. See http://en.wikipedia.org/wiki/X-Forwarded-For
	// for more information
	// TODO: use RFC7239 http://tools.ietf.org/html/rfc7239
	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		// If we aren't the first proxy retain prior
		// X-Forwarded-For information as a comma+space
		// separated list and fold multiple headers into one.
		if prior, ok := req.Header["X-Forwarded-For"]; ok {
			clientIP = strings.Join(prior, ", ") + ", " + clientIP
		}
		requestHeader.Set("X-Forwarded-For", clientIP)
	}

	// Set the originating protocol of the incoming HTTP request. The SSL might
	// be terminated on our site and because we doing proxy adding this would
	// be helpful for applications on the backend.
	requestHeader.Set("X-Forwarded-Proto", "http")
	if req.TLS != nil {
		requestHeader.Set("X-Forwarded-Proto", "https")
	}

	// Connect to the backend URL, also pass the headers we get from the requst
	// together with the Forwarded headers we prepared above.
	// TODO: support multiplexing on the same backend connection instead of
	// opening a new TCP connection time for each request. This should be
	// optional:
	// http://tools.ietf.org/html/draft-ietf-hybi-websocket-multiplexing-01
	connBackend, resp, err := dialer.Dial(backendURL.String(), requestHeader)
	if err != nil {
		log.Println(err, resp)
		w.WriteHeader(502)
		w.Write([]byte{})
		return
	}
	defer connBackend.Close()

	// Only pass those headers to the upgrader.
	upgradeHeader := http.Header{}
	upgradeHeader.Set("Sec-WebSocket-Protocol",
		resp.Header.Get(http.CanonicalHeaderKey("Sec-WebSocket-Protocol")))
	upgradeHeader.Set("Set-Cookie",
		resp.Header.Get(http.CanonicalHeaderKey("Set-Cookie")))

	// Now upgrade the existing incoming request to a WebSocket connection.
	// Also pass the header that we gathered from the Dial handshake.
	connPub, err := upgrader.Upgrade(w, req, upgradeHeader)
	if err != nil {
		log.Println(err)
		w.WriteHeader(502)
		w.Write([]byte{})
		return
	}
	defer connPub.Close()

	errc := make(chan error, 2)
	cp := func(dst io.Writer, src io.Reader) {
		_, err := io.Copy(dst, src)
		errc <- err
	}

	// Start our proxy now, everything is ready...
	go cp(connBackend.UnderlyingConn(), connPub.UnderlyingConn())
	go cp(connPub.UnderlyingConn(), connBackend.UnderlyingConn())
	<-errc
}

func writeResponseHeader(w http.ResponseWriter, r *http.Response) {
	for k := range r.Header {
		should_drop := false
		for i := range hopHeaders {
			if k == hopHeaders[i] {
				should_drop = true
				break
			}
		}

		if !should_drop {
			w.Header()[k] = r.Header[k]
		}
	}

	w.Header().Set("X-Forwarded-For", "127.0.0.1")
	w.WriteHeader(r.StatusCode)
}
