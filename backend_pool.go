package main

import (
	"log"
	"strings"
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
	name := p.findSpawnableAppName(host)

	var err error
	backend, err := p.maybeInitBackend(name)

	if err != nil {
		return "", err
	}

	err = backend.MaybeSpawnBackend()

	if err != nil {
		return "", err
	}

	backend.Touch()

	return backend.Address(), nil
}

func (p *BackendPool) maybeInitBackend(name string) (*Backend, error) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	var err error
	backend := p.backends[name]

	if backend == nil {
		backend, err = InitBackend(name)

		if err == nil {
			p.backends[name] = backend
		} else {
			return nil, err
		}
	} else if backend.IsRestartRequested() {
		log.Println("restarting", name)

		p.backends[name] = nil
		backend.Close()

		refreshed_backend, err := InitBackend(name)

		if err != nil {
			return nil, err
		}

		p.backends[name] = refreshed_backend
		backend = refreshed_backend
	}

	return backend, nil
}

func (p *BackendPool) findSpawnableAppName(host string) string {
	name := hostDropLast(host)
	ok := false
	for name != "" && ok == false {
		ok = IsSpawnableBackend(name)
		if ok == false {
			if strings.Contains(name, ".") {
				name = hostDropFirst(name)
			} else {
				name = ""
			}
		}
	}
	if name == "" {
		return host
	}
	return name
}

func (p *BackendPool) Close() {
	for k := range p.backends {
		p.backends[k].Close()
	}
}

func hostDropFirst(host string) string {
	dotIndex := strings.Index(host, ".")
	if dotIndex != -1 {
		return host[dotIndex+1 : len(host)]
	} else {
		return host
	}
}

func hostDropLast(host string) string {
	dotIndex := strings.LastIndex(host, ".")
	if dotIndex != -1 {
		return host[0:dotIndex]
	} else {
		return host
	}
}
