package main

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

type Backend struct {
	appPath      string
	mtx          sync.Mutex
	host         string
	port         int
	proxy        bool
	process      *os.Process
	spawned      bool
	startedAt    time.Time
	exited       bool
	exitChan     chan interface{}
	activityChan chan interface{}
}

func (b *Backend) Close() {
	if b.proxy {
		log.Println("Terminating", b.appPath, "proxy")

		b.exitChan <- true

		<-b.exitChan

		log.Println("Terminated", b.appPath)
		return
	}
	log.Println("Terminating", b.appPath, "pid", b.process.Pid)

	err := b.process.Signal(syscall.SIGTERM)
	if err != nil {
		log.Println("failed to kill process: ", err)
		return
	}

	<-b.exitChan

	log.Println("Terminated", b.appPath)
}

func (b *Backend) IsRestartRequested() bool {
	if b.exited {
		return true
	}
	if b.proxy {
		fi, err := os.Stat(b.appPath)
		if err != nil {
			return false
		}
		return fi.ModTime().After(b.startedAt)
	}
	fi, err := os.Stat(b.appPath + "/tmp/restart.txt")
	if err != nil {
		return false
	}
	return fi.ModTime().After(b.startedAt)
}

func IsSpawnableBackend(appName string) bool {
	pathToApp, err := appDir(appName)
	if err != nil {
		return false
	}
	_, err = os.Stat(pathToApp)
	if err != nil {
		return false
	}
	return true
}

type BootCrash struct {
	Log  bytes.Buffer
	Env  []string
	Cmd  string
	Path string
}

func (b BootCrash) Error() string {
	return "app crashed during boot"
}

func InitBackend(appName string) (*Backend, error) {
	pathToApp, err := appDir(appName)
	if err != nil {
		return nil, err
	}
	fileInfo, err := os.Stat(pathToApp)
	if err != nil {
		return nil, err
	}
	if fileInfo.IsDir() {
		return &Backend{appPath: pathToApp, proxy: false, spawned: false}, nil
	}
	return &Backend{appPath: pathToApp, proxy: true, spawned: false}, nil
}

func (b *Backend) MaybeSpawnBackend() error {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	if b.spawned == true {
		return nil
	}

	if b.proxy == false {
		return b.SpawnBackendProcfile()
	}

	return b.SpawnBackendProxy()
}

func (b *Backend) SpawnBackendProcfile() error {
	port, err := getFreeTCPPort()
	if err != nil {
		return err
	}
	log.Println("Spawning", b.appPath, "on port", port)

	env := os.Environ()

	pathbytes, err := ioutil.ReadFile(os.Getenv("HOME") + "/.pow/.path")
	path := os.Getenv("PATH")
	if err == nil {
		path = string(pathbytes)
	}
	// remove the old PATH
	for i, v := range env {
		if strings.Index(v, "PATH=") == 0 {
			env = append(env[:i], env[i+1:]...)
		}
	}
	env = append(env, "PATH="+path, "PORT="+strconv.Itoa(port))

	// add .env
	entries, err := godotenv.Read(b.appPath + "/.env")
	if err == nil {
		for k, v := range entries {
			env = append(env, k+"="+v)
		}
	} else {
		// allow absence of .env
		if !os.IsNotExist(err) {
			return err
		}
	}

	procfile, err := ReadProcfile(b.appPath + "/Procfile")
	if err != nil {
		return err
	}
	var CmdName string
	for _, v := range procfile.Entries {
		if v.Name == "web" {
			CmdName = v.Command
		}
	}

	if CmdName == "" {
		return errors.New("No 'web' entry found in Procfile")
	}

	cmd := exec.Command("bash", "-c", "exec "+CmdName)

	var bootlog bytes.Buffer

	toStderrWithCapture := io.MultiWriter(os.Stderr, &bootlog)

	cmd.Stdout = toStderrWithCapture // never write to gowd's stdout
	cmd.Stderr = toStderrWithCapture
	cmd.Dir = b.appPath
	cmd.Env = env

	err = cmd.Start()
	if err != nil {
		return err
	}

	exitChan := make(chan interface{}, 1)
	b.host = "127.0.0.1"
	b.port = port
	b.process = cmd.Process
	b.startedAt = time.Now()
	b.activityChan = make(chan interface{})
	b.exitChan = exitChan
	b.spawned = true
	booting := true
	crashChan := make(chan error, 1)
	go func() {
		cmd.Wait()
		b.exited = true
		b.exitChan <- new(interface{})

		if booting {
			crashChan <- BootCrash{Log: bootlog, Env: env, Cmd: CmdName, Path: b.appPath}
		}
	}()

	log.Println("waiting for spawn result for", b.appPath)

	select {
	case <-awaitTCP(b.Address()):
		log.Println(b.appPath, "came up successfully")
		booting = false
		go b.watchForActivity()

		return nil
	case <-time.After(30 * time.Second):
		log.Println(b.appPath, "failed to bind")
		cmd.Process.Kill()
		return errors.New("app failed to bind")
	case err := <-crashChan:
		log.Println(b.appPath, "crashed while starting")
		return err
	}
}

func (b *Backend) SpawnBackendProxy() error {
	appbytes, err := ioutil.ReadFile(b.appPath)
	app := ""
	if err == nil {
		app = strings.TrimSpace(string(appbytes))
	} else {
		log.Println("while reading app file:", err)
	}

	if strings.HasPrefix(app, "http://") {
		app = app[7:len(app)]
	} else {
		app = "127.0.0.1:" + app
	}

	host, portstring, err := net.SplitHostPort(app)
	if err != nil {
		return err
	}

	port, err := strconv.Atoi(portstring)
	if err != nil {
		return err
	}

	log.Println("Proxying", b.appPath, "to host", host, "on port", port)

	exitChan := make(chan interface{}, 1)
	b.host = host
	b.port = port
	b.process = nil
	b.startedAt = time.Now()
	b.activityChan = make(chan interface{})
	b.exitChan = exitChan
	b.spawned = true
	go func() {
		<-b.exitChan
		b.exited = true
		b.exitChan <- new(interface{})
	}()

	select {
	case <-awaitTCP(b.Address()):
		log.Println(b.appPath, "came up successfully")
		go b.watchForActivity()

		return nil
	case <-time.After(5 * time.Second):
		log.Println(b.appPath, "failed to bind")
		return errors.New("app failed to bind")
	}
}

func (b *Backend) Touch() {
	if b.activityChan != nil {
		b.activityChan <- new(interface{})
	}
}

func (b *Backend) Address() string {
	return net.JoinHostPort(b.host, strconv.Itoa(b.port))
}

// Close the backend after inactivity
func (b *Backend) watchForActivity() {
outer:
	for {
		select {
		case _, ok := <-b.activityChan:
			if ok {
				continue
			} else {
				b.Close()
				b.activityChan = nil
				break outer
			}

		case <-time.After(30 * time.Minute):
			log.Println(b.appPath, "backend idling.")
			b.Close()
			b.activityChan = nil
			break outer
		}
	}
}

func awaitTCP(address string) chan bool {
	c := make(chan bool)
	go func() {
		for {
			_, err := net.Dial("tcp", address)
			if err == nil {
				c <- true
				break
			}
			time.Sleep(200 * time.Millisecond)
		}
	}()
	return c
}

func getFreeTCPPort() (port int, err error) {
	// We still have a small race condition here, but meh.
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	port = l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port, nil
}

func appDir(name string) (path string, err error) {
	path, err = filepath.EvalSymlinks(os.Getenv("HOME") + "/.pow/" + name)
	return
}
