package route

import (
	"time"
	"fmt"

	log "github.com/sirupsen/logrus"
)

// Staging is used to define the process when deploying a new version of 
// an upstream application
// it defines the stages that a migration runs through
// in the final stage - if everything works fine - the new version will
// have 100% of the traffic while the old version has 0%
// Per Stage requirements are defined which need to be fulfilled for a given
// timeframe for the staging to advance to the next stage and increase
// the load of the new verion
// if the requirements are not met staging will remove the stage and go back to
// the previous state, which for most cases should be 100% traffic on the old version
type Staging struct {
	stages []Stage
	backends []*Backend
	timeout time.Duration
	backoffs int
}

// Start the staging process 
func (s *Staging) Start() {

	// loop over all stages that are defined 
	// idx allows to go back to previous stage
	for idx := 1; idx < len(s.stages); idx++ {
		log.Debugf("Starting stage %d", idx)

		if err := s.stages[idx].Start(); err != nil {
			// return to previous route and start there again
			idx--
			if idx < 0 {
				panic(fmt.Errorf("Cannot have stage with index < 0"))
			}
			continue
		}

		// continue to the next stage
		// 
	}
	
}

// A Stage defines the requirements that need to be met for a given
// amount of time until staging can advance to the next stage
// it defines:
// - the distribution of traffic while the stage is active
// - the deployment strategy (e. g. Header-Based Routing, Showrouting, Canary)
// - the timeframe for which the requirements need to be met
// - the requirements in form of metrics and a threshhold for each metric 
// - the next stage
type Stage struct {
	// defines the metricName and its threshhold
	requirements map[string]float64
	// the timeframe for which the requirements need to be met
	timeframe time.Duration
	// the amount of failures which are allowd before the previous state is used
	allowedFailures int

	backends []*Backend
}

func (s *Stage) Start() error{
	return nil
}