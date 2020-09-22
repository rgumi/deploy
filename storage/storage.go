package storage

import "github.com/sirupsen/logrus"

var (
	logger = logrus.New()
	log    = logger.WithFields(logrus.Fields{
		"component": "storage",
	})
)

type Metric struct {
	TotalResponses    int                `json:",omitempty"`
	ResponseStatus200 int                `json:",omitempty"`
	ResponseStatus300 int                `json:",omitempty"`
	ResponseStatus400 int                `json:",omitempty"`
	ResponseStatus500 int                `json:",omitempty"`
	ResponseStatus600 int                `json:",omitempty"`
	ContentLength     float64            `json:",omitempty"`
	ResponseTime      float64            `json:",omitempty"`
	CustomMetrics     map[string]float64 `json:",omitempty"`
}
