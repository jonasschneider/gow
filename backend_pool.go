package main

import (
	"log"
	"regexp"
	"sync"
)

type BackendPool struct {
	backends map[string]*Backend
	mtx      sync.Mutex
}

func NewBackendPool() *BackendPool {
	return &BackendPool{backends: make(map[string]*Backend)}
}

func (p *BackendPool) Select(host string) (string, error) {
	// Yes, we are this crazy. Lock the mutex during the entire lookup time, which could potentially include
	// (re)spawning an application. Serialize all of this so that we never have to deal with thundering-herd
	// spawns and such.
	// TODO: we could at least lock the spawn process by app name
	p.mtx.Lock()
	defer p.mtx.Unlock()

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
	log.Println("restarting", name)

	p.backends[name].Close()

	refreshed_backend, err := SpawnBackend(name)

	if err != nil {
		return err
	}

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
