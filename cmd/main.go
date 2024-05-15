package main

import (
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	mutator "github.com/rotemjac/k8s-mutation-hook-service/pkg/mutate"
)

func handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello %q", html.EscapeString(r.URL.Path))
}

func handleMutate(w http.ResponseWriter, r *http.Request) {
	log.Println("Received")
	// read the body / request
	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		sendError(err, w)
		return
	}

	// mutate the request
	mutated, err := mutator.Mutate(body, true)
	if err != nil {
		sendError(err, w)
		return
	}

	// and write it back
	log.Println("Returned")
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(mutated)
}

func sendError(err error, w http.ResponseWriter) {
	log.Println(err)
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintf(w, "%s", err)
}

func main() {
	log.Println("Starting server ...")

	mux := http.NewServeMux()

	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/mutate", handleMutate)

	s := &http.Server{
		Addr:           ":8443",
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1048576
	}

	log.Fatal(s.ListenAndServeTLS("/etc/webhook-secret/tls.crt", "/etc/webhook-secret/tls.key"))
}
