package metrics

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

var metricRepo *MetricsRepository
var metricChan chan<- Metrics
var backend1 = uuid.New()

var promMetricString = `
# Finally a summary, which has a complex representation, too:
# HELP rpc_duration_seconds A summary of the RPC duration in seconds.
# TYPE rpc_duration_seconds summary
rpc_duration_seconds{quantile="0.01"} 3102
rpc_duration_seconds{quantile="0.05"} 3272
rpc_duration_seconds{quantile="0.5"} 4773
rpc_duration_seconds{quantile="0.9"} 9001
rpc_duration_seconds{quantile="0.99"} 76656
rpc_duration_seconds_sum 1.7560473e+07
rpc_duration_seconds_count 2693
`

var testURL string
var testStorage *TestStorage

func init() {
	// start test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.String(), "/metrics") {
			w.Write([]byte(promMetricString))

		} else {
			w.WriteHeader(404)
		}
	}))
	testURL = ts.URL + "/metrics"

	testStorage = &TestStorage{
		data: testdataAlarm,
	}
}

var testdataAlarm = map[string]float64{
	"rpc_duration_seconds_sum":   3,
	"rpc_duration_seconds_count": 3001,
}

var testdataResolved = map[string]float64{
	"rpc_duration_seconds_sum":   1,
	"rpc_duration_seconds_count": 2001,
}

type TestStorage struct {
	data map[string]float64
}

func (t *TestStorage) Write(id uuid.UUID, data map[string]float64) {
	fmt.Printf("%v: %v\n", id, data)
}
func (t *TestStorage) Read(uuid.UUID) map[string]float64 {
	return t.data
}

func Test_NewMetricsRepository(t *testing.T) {
	metricChan, metricRepo = NewMetricsRepository(testStorage, 2*time.Second)

	go metricRepo.Listen()
	metrics1 := Metrics{
		BackendID:           backend1,
		clientFailResponses: 1234,
	}

	metricChan <- metrics1

	metricRepo.Stop()
}

func Test_NewBackend(t *testing.T) {

	go metricRepo.Listen()

	_, err := metricRepo.RegisterBackend(
		backend1,
		testURL,
		map[string]float64{
			"rpc_duration_seconds_sum":   2,
			"rpc_duration_seconds_count": 3000,
		},
	)
	if err != nil {
		t.Error("RegisterBackend should not have returned an error")
	}

	_, err = metricRepo.RegisterBackend(
		backend1,
		testURL,
		map[string]float64{
			"rpc_duration_seconds_sum":   2,
			"rpc_duration_seconds_count": 3000,
		},
	)
	if err == nil {
		t.Error("RegisterBackend should have returned an error")
	}

	time.Sleep(5 * time.Second)
	metricRepo.Stop()
}

func Test_RemoveBackend_Success(t *testing.T) {

	metricRepo.RemoveBackend(backend1)

	actual := len(metricRepo.Backends)
	expected := 0
	if actual != expected {
		t.Errorf("Should be 0 due to remove. Got: %d", actual)
	}

	err := metricRepo.RemoveBackend(backend1)
	if err == nil {
		t.Error("The instance should not have been found!")
	}
}

func Test_RemoveBackend_Failure(t *testing.T) {

	err := metricRepo.RemoveBackend(uuid.New())
	if err == nil {
		t.Error("The instance should not have been found!")
	}
}

func Test_Monitor(t *testing.T) {
	alertChan, err := metricRepo.RegisterBackend(
		backend1,
		testURL,
		map[string]float64{
			"rpc_duration_seconds_sum":   2,
			"rpc_duration_seconds_count": 3000,
		},
	)
	if err != nil {
		t.Error("RegisterBackend should not have returned an error")
	}

	go metricRepo.Monitor(backend1, time.Duration(2)*time.Second)

	go func() {
		for {
			select {
			case alert := <-alertChan:
				fmt.Println(alert)

				// resolve alarm manually to check if its working correctly
				time.Sleep(1 * time.Second)
				testStorage.data = testdataResolved
			}

		}
	}()

	time.Sleep(20 * time.Second)
	metricRepo.Stop()
}
