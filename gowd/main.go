package main

import (
  "github.com/jonasschneider/gow"
  "log"
)

func main() {
  errors := make(chan error)

  go func() {
    if err := gow.ListenAndServeDNS("127.0.0.1:20560"); err != nil {
      log.Println("DNS error:",err)
      errors <- err
    }
  }()

  gow.SpawnBackend("/Users/jonas/code/telpit")

  log.Println("spawned all the things")

  <- errors
}
