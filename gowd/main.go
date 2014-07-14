package main

import (
	"github.com/jonasschneider/gow"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	errors := make(chan error)

	go func() {
		if err := gow.ListenAndServeDNS("127.0.0.1:20560"); err != nil {
			log.Println("DNS error:", err)
			errors <- err
		}
	}()

	pool := gow.NewBackendPool()

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
	log.Fatalln(gow.ListenAndServeHTTP("127.0.0.1:20559", pool))
}
