package main

import (
	"log"
	"net/http"
  "strings"
  "os/exec"
  "os"
  "fmt"

	"github.com/gorilla/mux"
  "github.com/joho/godotenv"
)

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

  r := mux.NewRouter()

  r.HandleFunc("/add/", addFeed)
  log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), r))
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

  newFeed.initiateStream()

  feeds = append(feeds, newFeed)

  w.WriteHeader(http.StatusOK)
  w.Write([]byte("true;"))
}
