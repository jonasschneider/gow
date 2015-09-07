package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	errors := make(chan error)

	go func() {
		if err := ListenAndServeDNS("127.0.0.1:20560"); err != nil {
			log.Println("DNS error:", err)
			errors <- err
		}
	}()

	pool := NewBackendPool()

	termchan := make(chan os.Signal, 2)
	signal.Notify(termchan, os.Interrupt, syscall.SIGTERM)
	go func() {
		for sig := range termchan {
			log.Printf("received %#v, shutting down..", sig)
			pool.Close()
			log.Println("exiting.")
			os.Exit(0)
		}
	}()

	log.Println("Ready!")
	log.Fatalln(ListenAndServeHTTP("127.0.0.1:20559", pool))
}
