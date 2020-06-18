package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type Feed struct {
  address string
  id string
}

var feeds []*Feed

func main() {
  r := mux.NewRouter()

  r.HandleFunc("/", test)
  log.Fatal(http.ListenAndServe(":8000", r))
}

func addFeed(w http.ResponseWriter, r *http.Request) {

}
