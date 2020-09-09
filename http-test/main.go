package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	counter1, counter2 int
	timeout            *int
)

func hello(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()
	time.Sleep(time.Duration(*timeout) * time.Millisecond)
	counter1++
	w.WriteHeader(200)

	b := make([]byte, 2900)
	rand.Read(b)
	w.Write(b)

	fmt.Printf("%d/%d\n", counter1, counter2)
}

func world(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()
	time.Sleep(time.Duration(*timeout) * time.Millisecond)
	counter2++

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write([]byte("{\"test\":\"World\"}"))

	fmt.Printf("%d/%d\n", counter1, counter2)
}

func main() {
	addr := flag.String("addr", ":7070", "defines the addr of the server")
	promAddr := flag.String("prom", ":7071", "defines the addr of the prom server")
	timeout = flag.Int("timeout", 0, "defines the timeout of the handle before sending a response")
	flag.Parse()

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		if err := http.ListenAndServe(*promAddr, nil); err != nil {
			log.Fatal(err)
		}
	}()

	fmt.Printf("Addr: %v. Timeout %v. PromAddr: %v\n", *addr, *timeout, *promAddr)
	router1 := httprouter.New()
	router1.GET("/", hello)
	router1.GET("/hello/", hello)
	router1.GET("/world/", world)

	server1 := http.Server{
		Addr:         *addr,
		Handler:      router1,
		ReadTimeout:  time.Second * 2,
		WriteTimeout: time.Second * 2,
		IdleTimeout:  time.Second * 20,
	}

	log.Fatal(server1.ListenAndServe())
}