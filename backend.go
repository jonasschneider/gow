package gow

import (
  "os/exec"
  "os"
  "strconv"
  "net"
)


type Backend struct {
  appPath string
  port int
  process *exec.Cmd
}

func SpawnBackend(pathToApp string) (*Backend, error) {
  port, err := getFreeTCPPort()
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

  return &Backend{appPath: pathToApp, port: port, process: cmd}, nil
}

func getFreeTCPPort() (port int, err error) {
  // We still have a small race condition here, but meh.
  l, err := net.Listen("tcp", "127.0.0.1:0") // listen on localhost
  if err != nil { return 0, err }
  port = l.Addr().(*net.TCPAddr).Port
  return port, nil
}
