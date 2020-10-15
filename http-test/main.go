package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

var (
	counter1, counter2 int
	addr               = flag.String("addr", ":7070", "defines the addr of the server")
	promAddr           = flag.String("prom", ":7071", "defines the addr of the prom server")
	timeout            = flag.Int("timeout", 0, "defines the timeout of the handle before sending a response")
	statusCode         = flag.Int("statusCode", 500, "defines the status code that is returned on testing handle")
)

func LogRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		before := time.Now()

		defer func() {
			delta := time.Since(before)

			log.Infof("%s \"%s %s %s\" %v",
				r.RemoteAddr, r.Method, r.URL.Path,
				r.Proto, delta,
			)
		}()
		handler.ServeHTTP(w, r)
	})
}

func hello(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()
	time.Sleep(time.Duration(*timeout) * time.Millisecond)
	counter1++
	w.WriteHeader(200)

	b := make([]byte, 2900)
	rand.Read(b)
	w.Write(b)
}

func returnStatusCode(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()
	w.WriteHeader(*statusCode)
}

func world(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()
	time.Sleep(time.Duration(*timeout) * time.Millisecond)
	counter2++

	b, _ := ioutil.ReadAll(r.Body)
	fmt.Println(string(b))

	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Backend", *addr)
	w.WriteHeader(200)
	w.Write([]byte("{\"test\":\"World\"}"))
}

func main() {

	flag.Parse()

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		if err := http.ListenAndServe(*promAddr, nil); err != nil {
			log.Fatal(err)
		}
	}()

	fmt.Printf("Addr: %v. Timeout %v. PromAddr: %v\n", *addr, *timeout, *promAddr)
	router1 := httprouter.New()
	router1.GET("/", world)
	router1.GET("/hello/", hello)
	router1.GET("/world/", world)
	router1.GET("/status/", returnStatusCode)

	server1 := http.Server{
		Addr:         *addr,
		Handler:      LogRequest(router1),
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
		IdleTimeout:  time.Second * 30,
	}

	log.Fatal(server1.ListenAndServe())
}
