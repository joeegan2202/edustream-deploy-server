package main

import (
	"bytes"
	"crypto"
  "crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

var key *rsa.PrivateKey

type Feed struct {
  address string
  id string
  streamCmd *exec.Cmd
}

var feeds []*Feed

func main() {
  if godotenv.Load("config.env") != nil {
    log.Fatal("Failed to get configuration while loading port number!")
  }

  port := os.Getenv("PORT")

  setupKey()

  r := mux.NewRouter()

  r.HandleFunc("/add/", addFeed)
  r.PathPrefix("/ingest/").Handler(http.StripPrefix("/ingest/", new(IngestServer))) // The actual file server for streams
  log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), r))
}

func setupKey() {
  file, err := os.OpenFile("key.pem", os.O_RDWR, 0755)

  if err != nil {
    log.Fatalf("Couldn't open keyfile! %s\n", err.Error())
  }

  data, err := ioutil.ReadAll(file)

  if err != nil {
    log.Fatalf("Couldn't read from keyfile! %s\n", err.Error())
  }

  key, err = x509.ParsePKCS1PrivateKey(data)
}

func addFeed(w http.ResponseWriter, r *http.Request) {
  w.Header().Set("Access-Control-Allow-Origin", "*")

  //ip := r.Header.Get("X-FORWARDED-FOR")
  //if ip != "18.222.231.117" && r.RemoteAddr != "18.222.231.117" {
  //  w.WriteHeader(http.StatusUnauthorized)
  //  w.Write([]byte("false;Wrong IP to get stream"))
  //  return
  //}

  query := r.URL.Query()

  if query["id"] == nil || query["address"] == nil {
    w.WriteHeader(http.StatusBadRequest)
    w.Write([]byte("false;Incorrect parameters to start camera"))
    return
  }

  id := query["id"][0]
  address := query["address"][0]

  fmt.Printf("Starting stream with ID: %s, and address: %s\n", id, address)

  for _, f := range feeds {
    if f.id == id {
      w.WriteHeader(http.StatusOK)
      w.Write([]byte("true;Already started"))
      return
    }
  }

  newFeed := new(Feed)
  newFeed.id = strings.ReplaceAll(id, "\"", "\\\"")
  newFeed.address = strings.ReplaceAll(address, "\"", "\\\"")

  if err := newFeed.initiateStream(); err != nil {
    fmt.Println(err.Error())
  }

  feeds = append(feeds, newFeed)

  w.WriteHeader(http.StatusOK)
  w.Write([]byte("true;"))
}

type IngestServer struct {}

func (i *IngestServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  chunk := make([]byte, 2048)
  _, err := io.ReadAtLeast(r.Body, chunk, 100)

  if err != nil {
    fmt.Printf("Error while trying to read chunk of body data! %s\n", err.Error())
    return
  }

  fmt.Println(r.URL.RawPath)

  hasher := sha256.New()

  hasher.Write(chunk)

  signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hasher.Sum(nil))

  if err != nil {
    fmt.Printf("Error trying to sign data chunk! %s\n", err.Error())
    return
  }

  client := new(http.Client)

  req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("https://api.edustream.live/%s?signature=%x", r.URL.RawPath, signature), io.MultiReader(bytes.NewReader(chunk), r.Body))

  if err != nil {
    fmt.Printf("Error trying to create ingest request! %s\n", err.Error())
    return
  }

  client.Do(req)
}
