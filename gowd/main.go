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
      log.Println("DNS error:",err)
      errors <- err
    }
  }()

  pool := gow.NewBackendPool()

  c := make(chan os.Signal, 1)
  signal.Notify(c, os.Interrupt)
  go func(){
      for _ = range c {
        pool.Close()
        os.Exit(0)
      }
  }()

  termchan := make(chan os.Signal, 2)
  signal.Notify(termchan, os.Interrupt, syscall.SIGTERM)

  log.Fatalln(gow.ListenAndServeHTTP("127.0.0.1:20559", pool))
}
