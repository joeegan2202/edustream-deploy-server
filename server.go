package main

import (
	"log"
	"net/http"
  "strings"
  "os/exec"
  "fmt"

	"github.com/gorilla/mux"
)

type Feed struct {
  address string
  id string
  streamCmd *exec.Cmd
}

var feeds []*Feed

func main() {
  r := mux.NewRouter()

  r.HandleFunc("/add/", addFeed)
  r.PathPrefix("/stream/").Handler(http.StripPrefix("/stream/", new(StreamServer)))
  log.Fatal(http.ListenAndServe(":8000", r))
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

type StreamServer struct {}

func (s *StreamServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  w.Header().Set("Access-Control-Allow-Origin", "*")

  //ip := r.Header.Get("X-FORWARDED-FOR")
  //if ip != "18.222.231.117" && r.RemoteAddr != "18.222.231.117" {
  //  w.WriteHeader(http.StatusUnauthorized)
  //  w.Write([]byte("false;Wrong IP to get stream"))
  //  return
  //}

  id := strings.Split(r.URL.Path, "/")[0]

  for _, f := range feeds {
    if f.id == id {
      http.StripPrefix(id, http.FileServer(http.Dir(fmt.Sprintf("./streams/%s", id)))).ServeHTTP(w, r)
      return
    }
  }

  w.WriteHeader(http.StatusBadRequest)
  w.Write([]byte(`{"status": false, "err": "No session for stream found"}`))
}
