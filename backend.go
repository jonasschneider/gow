package gow

import (
  "os/exec"
  "os"
  "strconv"
  "net"
  "time"
  "errors"
  "log"
  "syscall"
)


type Backend struct {
  appPath string
  port int
  process *os.Process
  startedAt time.Time
  exited bool
}

func (b *Backend) Close() {
  log.Println("Terminating",b.appPath,"pid",b.process.Pid)
  b.process.Signal(syscall.SIGTERM)
}

func (b *Backend) IsRestartRequested() bool {
  if b.exited { return true }
  fi, err := os.Stat(b.appPath+"/tmp/restart.txt")
  if err != nil { return false }
  return fi.ModTime().After(b.startedAt)
}

func SpawnBackend(appName string) (*Backend, error) {
  pathToApp := appDir(appName)
  port, err := getFreeTCPPort()
  log.Println("Spawning",pathToApp,"on port",port)
  if err != nil { return nil, err }

  env := os.Environ()
  env = append(env, "PORT="+strconv.Itoa(port))
  cmd := exec.Command("foreman", "start", "web")
  cmd.Stdout = os.Stdout // TODO: logging
  cmd.Stderr = os.Stderr
  cmd.Dir = pathToApp
  cmd.Env = env

  err = cmd.Start()
  if err != nil {
    return nil, err
  }

  b := &Backend{appPath: pathToApp, port: port, process: cmd.Process, startedAt: time.Now()}

  crashChan := make(chan error, 1)
  go func() {
    crash := cmd.Wait()
    b.exited = true

    if crash != nil {
      crashChan <- crash
    }
  }()

  select {
  case <- awaitTCP(b.Address()):
    log.Println(pathToApp,"came up successfully")
    return b, nil
  case <- time.After(30 * time.Second):
    log.Println(pathToApp,"failed to bind")
    cmd.Process.Kill()
    return nil, errors.New("app failed to bind")
  case crash := <-crashChan:
    log.Println(pathToApp,"crashed while starting")
    return nil, crash
  }
}

func (b *Backend) Address() string {
  return "localhost:"+strconv.Itoa(b.port)
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
  if err != nil { return 0, err }
  port = l.Addr().(*net.TCPAddr).Port
  l.Close()
  return port, nil
}

func appDir(name string) string {
  return os.Getenv("HOME")+"/.pow/"+name
}
