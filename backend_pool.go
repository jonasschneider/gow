package gow

import (
  "os"
)

type BackendPool struct {
  backends map[string]*Backend
}

func NewBackendPool() *BackendPool {
  return &BackendPool{backends: make(map[string]*Backend) }
}

func (p *BackendPool) Select(name string) (string, error) {
  var address string
  backend := p.backends[name]

  if backend != nil {
    address = backend.Address()
  } else {
    backend, err := SpawnBackend(os.Getenv("HOME")+"/.pow/"+name)

    if err == nil {
      p.backends[name] = backend
      address = backend.Address()
    } else {
      return "", err
    }
  }

  return address, nil
}
