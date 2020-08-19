package route

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

func Test_Route(t *testing.T) {

	route, err := New("/", "", 5, NewCanary(10))
	if err != nil {
		t.Errorf(err.Error())
	}
	route.Targets = []Target{
		Target{
			Addr: "http://localhost:7070",
		},
		Target{
			Addr: "http://localhost:9090",
		},
	}

	route.Init()
	exitLoop := false
	go func() {
		for i := 0; i < 10; i++ {
			if exitLoop {
				fmt.Println("exiting loop")
				return
			}
			r, _ := http.NewRequest("GET", "http://example.com/", nil)
			r.Header.Add("Server", "GoTCP")
			route.RequestChan <- r
			time.Sleep(100 * time.Millisecond)
		}
	}()

	time.Sleep(5 * time.Second)
	route.Strategy.Update(8)
	go func() {
		for i := 0; i < 10; i++ {
			if exitLoop {
				fmt.Println("exiting loop")
				return
			}
			r, _ := http.NewRequest("GET", "http://example.com/", nil)
			r.Header.Add("Server", "GoTCP")
			route.RequestChan <- r
			time.Sleep(100 * time.Millisecond)
		}
	}()

	exit := make(chan int)

	go func() {
		time.Sleep(20 * time.Second)
		route.Shutdown()
		exitLoop = true
		exit <- 1
		fmt.Println("Finished exit")
		return
	}()

	for {
		select {
		case got := <-route.ResponseChan:
			fmt.Println(got)
		case got := <-route.MetricChan:
			fmt.Println(got)

		case _ = <-exit:
			fmt.Println("exiting select")
			return
		}
	}
}
