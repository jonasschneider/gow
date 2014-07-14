package gow

import (
  "regexp"
  "log"
)

type BackendPool struct {
  backends map[string]*Backend
}

func NewBackendPool() *BackendPool {
  return &BackendPool{backends: make(map[string]*Backend) }
}

func (p *BackendPool) Select(host string) (string, error) {
  name := appNameFromHost(host)
  var err error
  p.restartIfRequested(name)

  backend := p.backends[name]

  if backend == nil {
    backend, err = SpawnBackend(name)

    if err == nil {
      p.backends[name] = backend
    } else {
      return "", err
    }
  }

  backend.Touch()

  return backend.Address(), nil
}

func (p *BackendPool) restartIfRequested(name string) error {
  if p.backends[name] == nil || !p.backends[name].IsRestartRequested() {
    return nil
  }
  log.Println("restarting",name)

  p.backends[name].Close()

  refreshed_backend, err := SpawnBackend(name)

  if err != nil { return err }

  p.backends[name] = refreshed_backend

  return nil
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
