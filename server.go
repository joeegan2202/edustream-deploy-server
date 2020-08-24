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
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

var key *rsa.PrivateKey

type Feed struct {
	address   string
	id        string
	streamCmd *exec.Cmd
}

var feeds []*Feed
var hlsTime string

func main() {
	if godotenv.Load("config.env") != nil {
		log.Fatal("Failed to get configuration while loading port number!")
	}

	port := os.Getenv("PORT")
	hlsTime = os.Getenv("HLS_TIME")

	setupKey()

	r := mux.NewRouter()

	r.HandleFunc("/add/", addFeed)
	r.HandleFunc("/stop/", stopFeed)
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
		return
	}

	feeds = append(feeds, newFeed)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("true;"))

	sendStatus(id, 1)
}

func stopFeed(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	query := r.URL.Query()

	if query["id"] == nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("false;Incorrect parameters to stop feed"))
		return
	}

	id := query["id"][0]

	fmt.Printf("Stopping feed with ID: %s\n", id)

	for _, f := range feeds {
		if f.id == id {
			f.streamCmd.Process.Kill()
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("true;Feed stopped"))
			return
		}
	}

	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("No feed found with id!"))
}

type IngestServer struct{}

func (i *IngestServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	chunk := make([]byte, 2048)
	bytesRead, err := io.ReadAtLeast(r.Body, chunk, 100)

	log.Println(bytesRead)

	if err != nil {
		fmt.Printf("Error while trying to read chunk of body data! %s\n", err.Error())
		return
	}

	fmt.Println(r.URL.Path)

	hasher := sha256.New()

	hasher.Write(chunk[:bytesRead])

	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hasher.Sum(nil))

	if err != nil {
		fmt.Printf("Error trying to sign data chunk! %s\n", err.Error())
		return
	}

	client := new(http.Client)

	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("https://api.edustream.live/ingest/%s?signature=%x", r.URL.Path, signature), io.MultiReader(bytes.NewReader(chunk[:bytesRead]), r.Body))

	if err != nil {
		fmt.Printf("Error trying to create ingest request! %s\n", err.Error())
		return
	}

	res, err := client.Do(req)

	if err != nil {
		log.Printf("Error sending request! %s\n", err.Error())
		return
	}

	data, err := ioutil.ReadAll(res.Body)
	fmt.Printf("Response received from %s: %s\n", req.URL.RequestURI(), string(data))
}

func sendStatus(cameraId string, status int8) {
	hasher := sha256.New()

	now := time.Now().Format(time.RFC3339Nano)

	hasher.Write([]byte(now))

	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hasher.Sum(nil))

	if err != nil {
		log.Printf("Error signing timestamp! %s\n", err.Error())
		return
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.edustream.live/status/?signature=%x&cameraId=%s&status=%d", signature, cameraId, status), strings.NewReader(now))

	if err != nil {
		log.Printf("Error creating request! %s\n", err.Error())
		return
	}

	client := new(http.Client)

	res, err := client.Do(req)

	if err != nil {
		log.Printf("Error sending request! %s\n", err.Error())
		return
	}

	data, _ := ioutil.ReadAll(res.Body)
	log.Printf("Response returned: %s\n", string(data))
}
