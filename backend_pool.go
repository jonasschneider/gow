package gow

import (
  "os"
  "regexp"
)

type BackendPool struct {
  backends map[string]*Backend
}

func NewBackendPool() *BackendPool {
  return &BackendPool{backends: make(map[string]*Backend) }
}

func (p *BackendPool) Select(host string) (string, error) {
  name := appNameFromHost(host)
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

func (p *BackendPool) Close() {
  for k := range p.backends {
    p.backends[k].Close()
  }
}

var hostRegex = regexp.MustCompile("([a-z_\\-0-9A-Z]+)")

func appNameFromHost(host string) string {
  return hostRegex.FindString(host)
}
