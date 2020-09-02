package storage

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
)

var (
	id1   = uuid.New()
	id2   = uuid.New()
	id3   = uuid.New()
	local *LocalStorage
)

func Test_Write(t *testing.T) {
	local = NewLocalStorage()

	local.Write("TestRoute", id1, map[string]float64{
		"error_rate":      0,
		"http_status_500": 0,
	}, 100, 200, 0)
	local.Write("TestRoute", id1, map[string]float64{
		"error_rate":      0,
		"http_status_500": 0,
	}, 100, 200, 0)
	local.Write("TestRoute", id1, map[string]float64{
		"error_rate":      0,
		"http_status_500": 0,
	}, 400, 200, 0)

	local.Write("TestRoute", id2, map[string]float64{
		"error_rate":      2,
		"http_status_500": 3,
	}, 1000, 200, 0)

	local.Write("TestRoute", id3, map[string]float64{
		"error_rate":      2,
		"http_status_500": 3,
	}, 100, 200, 0)
}

func Test_Read(t *testing.T) {
	fmt.Println(local.ReadRoute("TestRoute"))
	fmt.Println(local.ReadBackend(id1))
}
