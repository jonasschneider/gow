package main

import (
  "testing"
  "os"
  "io/ioutil"
  "log"
  "net/http"
)

func TestSimpleBackendSpawn(t *testing.T) {
    err := os.Mkdir(Tempdir+"/.pow/app1", 0700)
    if err != nil { t.Fatal(err) }
    //err = ioutil.WriteFile(Tempdir+"/.pow/myapp/Procfile", []byte("web: bash -c 'while true; do echo hi; sleep 1; done'\n"), 0700)
    err = ioutil.WriteFile(Tempdir+"/.pow/app1/Procfile", []byte("web: socat TCP-LISTEN:$PORT,crlf,fork SYSTEM:\"echo HTTP/1.1 200 OK; echo Content-Type\\: text/plain; echo; echo hello\"\n"), 0700)
    if err != nil { t.Fatal(err) }
    b, err := SpawnBackend("app1")
    if err != nil { t.Fatal(err) }
    resp, err := http.Get("http://"+b.Address()+"/")
    if err != nil { t.Fatal(err) }
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil { t.Fatal(err) }
    if string(body) != "hello\r\n" {
      t.Fatal("body should have been 'hello\\r\\n', but was:",string(body),body)
    }

    b.Close()
}

func TestEnv(t *testing.T) {
    err := os.Mkdir(Tempdir+"/.pow/app2", 0700)
    if err != nil { t.Fatal(err) }
    //err = ioutil.WriteFile(Tempdir+"/.pow/myapp/Procfile", []byte("web: bash -c 'while true; do echo hi; sleep 1; done'\n"), 0700)
    err = ioutil.WriteFile(Tempdir+"/.pow/app2/.env", []byte("KEY1=VAL1\nKEY2=VAL2\n"), 0700)
    if err != nil { t.Fatal(err) }
    err = ioutil.WriteFile(Tempdir+"/.pow/app2/Procfile", []byte("web: socat TCP-LISTEN:$PORT,crlf,fork SYSTEM:\"echo HTTP/1.1 200 OK; echo Content-Type\\: text/plain; echo; echo $KEY1\\,$KEY2\"\n"), 0700)
    if err != nil { t.Fatal(err) }
    b, err := SpawnBackend("app2")
    if err != nil { t.Fatal(err) }
    resp, err := http.Get("http://"+b.Address()+"/")
    if err != nil { t.Fatal(err) }
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil { t.Fatal(err) }
    if string(body) != "VAL1,VAL2\r\n" {
      t.Fatal("body should have been 'VAL1,VAL2\\r\\n', but was:",string(body),body)
    }

    b.Close()
}


var Tempdir string

func TestMain(m *testing.M) {
  var err error
  Tempdir, err = ioutil.TempDir("", "gow-test")
  if err != nil {
    log.Fatalln(err)
  }
  os.Setenv("HOME", Tempdir)
  err = os.Mkdir(Tempdir+"/.pow", 0700)
  if err != nil {
    log.Fatalln(err)
  }

  os.Exit(m.Run())
}
