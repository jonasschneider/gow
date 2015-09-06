package gow

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
	"bytes"
	"io"
)

type Backend struct {
	appPath      string
	port         int
	process      *os.Process
	startedAt    time.Time
	exited       bool
	activityChan chan interface{}
}

func (b *Backend) Close() {
	log.Println("Terminating", b.appPath, "pid", b.process.Pid)

	done := make(chan interface{})

	go func() {
		b.process.Wait()
		close(done)
	}()

	// so sorry for this: SIGTERM the forego child process, not (a) forego itself, and (b) not the shell forego spawns
	shellcmd := fmt.Sprintf("/usr/local/bin/pstree %d|sed 's/^[^0-9]*//'| grep -v forego| grep -v .profile | cut -d ' ' -f 1|xargs kill; echo done", b.process.Pid)
	cmd := exec.Command("bash", "-c", shellcmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("failed to kill process: ", err, string(out))
		return
	}

	<-done

	log.Println("Terminated", b.appPath)
}

func (b *Backend) IsRestartRequested() bool {
	if b.exited {
		return true
	}
	fi, err := os.Stat(b.appPath + "/tmp/restart.txt")
	if err != nil {
		return false
	}
	return fi.ModTime().After(b.startedAt)
}

type BootCrash struct {
	Log bytes.Buffer
	Env []string
	Cmd string
}
func (b BootCrash) Error() string {
	return "app crashed during boot"
}

func SpawnBackend(appName string) (*Backend, error) {
	pathToApp := appDir(appName)
	port, err := getFreeTCPPort()
	log.Println("Spawning", pathToApp, "on port", port)
	if err != nil {
		return nil, err
	}

	env := os.Environ()

	pathbytes, err := ioutil.ReadFile(os.Getenv("HOME") + "/.pow/.path")
	path := os.Getenv("PATH")
	if err == nil {
		path = string(pathbytes)
	} else {
		log.Println("while reading path file:", err)
	}
	env = append([]string{"PATH="+path}, env...)

	gobin, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Println("while determining GOPATH:", err)
		gobin = "."
	}
	cmd := exec.Command(gobin+"/forego", "start", "-p", strconv.Itoa(port), "web")
	procfile, err := ReadProcfile(pathToApp+"/Procfile")
	if err != nil {
		log.Println("while parsing procfile:", err)
	}
	var CmdName string
	for _, v := range procfile.Entries {
		if v.Name == "web" {
			CmdName = v.Command
		}
	}

	var bootlog bytes.Buffer

	toStderrWithCapture := io.MultiWriter(os.Stderr, &bootlog)

	cmd.Stdout = toStderrWithCapture // never write to gowd's stdout
	cmd.Stderr = toStderrWithCapture
	cmd.Dir = pathToApp
	cmd.Env = env

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	b := &Backend{appPath: pathToApp, port: port, process: cmd.Process, startedAt: time.Now(), activityChan: make(chan interface{})}
	booting := true
	crashChan := make(chan error, 1)
	go func() {
		cmd.Wait()
		b.exited = true

		if booting {
			crashChan <- BootCrash{Log: bootlog, Env: env, Cmd: CmdName}
		}
	}()

	log.Println("waiting for spawn result for", pathToApp)

	select {
	case <-awaitTCP(b.Address()):
		log.Println(pathToApp, "came up successfully")
		booting = false
		go b.watchForActivity()

		return b, nil
	case <-time.After(30 * time.Second):
		log.Println(pathToApp, "failed to bind")
		cmd.Process.Kill()
		return nil, errors.New("app failed to bind")
	case err := <-crashChan:
		log.Println(pathToApp, "crashed while starting")
		return nil, err
	}
}

func (b *Backend) Touch() {
	if b.activityChan != nil {
		b.activityChan <- new(interface{})
	}
}

func (b *Backend) Address() string {
	return "127.0.0.1:" + strconv.Itoa(b.port)
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

func appDir(name string) string {
	return os.Getenv("HOME") + "/.pow/" + name
}
