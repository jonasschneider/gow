package gow

import (
	"io"
	"log"
	"net/http"
	"os"
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
}

type BackendSelector interface {
	Select(requestHost string) (string, error)
}

var ShutdownChan = make(chan bool, 1)

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
			w.Write([]byte(os.Getenv("PATH")))
			w.Write([]byte(err.Error()))
		}
	}
}

func proxyRequest(w http.ResponseWriter, r *http.Request, backendAddress string) {
	// TODO: we should also filter request hop headers

	r.RequestURI = ""
	r.URL.Scheme = "http"
	r.URL.Host = backendAddress

	resp, err := http.DefaultTransport.RoundTrip(r)

	if err != nil {
		log.Println(err)
		w.WriteHeader(502)
		w.Write([]byte{})
	} else {
		for k := range resp.Header {
			found := false
			for i := range hopHeaders {
				if k == hopHeaders[i] {
					found = true
					break
				}
			}
			if !found {
				w.Header()[k] = resp.Header[k]
			}
		}
		w.Header().Set("X-Forwarded-For", "127.0.0.1")
		w.WriteHeader(resp.StatusCode)
		_, err := io.Copy(w, resp.Body)
		if err != nil {
			log.Println(err)
		}
	}
}
