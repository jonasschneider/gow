package gow

import (
	"io"
	"log"
	"net/http"

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
			// serve error
			w.WriteHeader(500)
			w.Write([]byte("Failed to spawn backend: "))
			w.Write([]byte(err.Error()))
		}
	}
}

func proxyRequest(w http.ResponseWriter, r *http.Request, backendAddress string) {
	r.RequestURI = ""
	r.URL.Scheme = "http"
	r.URL.Host = backendAddress

	if r.Header["Connection"] != nil && r.Header["Connection"][0] == "Upgrade" &&
		r.Header["Upgrade"] != nil && r.Header["Upgrade"][0] == "websocket" {
		log.Println("websocket connection requested by client; doing things")
		proxyWebsocket(w, r)
		return
	}

	resp, err := http.DefaultTransport.RoundTrip(r)

	if err != nil {
		log.Println(err)
		w.WriteHeader(502)
		w.Write([]byte{})
		return
	}

	writeResponseHeader(w, resp)

	// just stream the body to the client
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Println(err)
	}
}

func proxyWebsocket(w http.ResponseWriter, clientRequest *http.Request) {
	clientRequest.URL.Scheme = "ws"

	upstream_conn, resp, err := websocket.DefaultDialer.Dial(clientRequest.URL.String(), clientRequest.Header)
	if err != nil {
		log.Println(err, resp)
		w.WriteHeader(502)
		w.Write([]byte{})
		return
	}

	for k := range hopHeaders {
		delete(resp.Header, hopHeaders[k])
	}

	client_conn, err := websocket.Upgrade(w, clientRequest, resp.Header, 4096, 4096)
	if err != nil {
		log.Println(err)
		w.WriteHeader(502)
		w.Write([]byte{})
		return
	}

	go func() {
		for {
			messageType, p, err := client_conn.ReadMessage()
			if err != nil {
				log.Println("error while reading from client:", err)
				break
			}
			if err = upstream_conn.WriteMessage(messageType, p); err != nil {
				log.Println("error while writing to upstream:", err)
				break
			}
		}
		upstream_conn.Close()
	}()

	go func() {
		for {
			messageType, p, err := upstream_conn.ReadMessage()
			if err != nil {
				log.Println("error while reading from upstream:", err)
				break
			}
			if err = client_conn.WriteMessage(messageType, p); err != nil {
				log.Println("error while writing to client:", err)
				break
			}
		}

		client_conn.Close()
	}()
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
