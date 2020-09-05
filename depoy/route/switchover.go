package route

import (
	"fmt"
	"time"
)

// the metrics which are allowed for the condtions
var allowedOperators = []string{">", "==", "<"}

// Condition is used to evaluate the state
// of a backend and take action according to
// the values defined here
type Condition struct {
	Status      bool                            // Status if the condition active for long enough and is therefore true
	Metric      string                          // Name of the metric
	Operator    string                          // < > ==
	Threshhold  float64                         // Threshhold that is checked
	ActiveFor   time.Duration                   // Duration for which the condition has to be met
	triggerTime time.Time                       // time the condition was first true
	IsTrue      func(m map[string]float64) bool // Condtional function to evaluate condition
}

func (c *Condition) compile() func(m map[string]float64) bool {

	switch c.Operator {
	case "<":
		return func(m map[string]float64) bool {
			if m[c.Metric] < c.Threshhold {
				return true
			}
			return false
		}
	case "==":
		return func(m map[string]float64) bool {
			if m[c.Metric] == c.Threshhold {
				return true
			}
			return false
		}
	case ">":
		return func(m map[string]float64) bool {
			if m[c.Metric] > c.Threshhold {
				return true
			}
			return false
		}
	}
	return nil
}

// NewCondition returns a new condition for the given parameters
// Initializes correctly by setting up IsTrue to a conditional function
func NewCondition(metric, operator string, threshhold float64, activeFor time.Duration) (*Condition, error) {

	if metric == "" || operator == "" || activeFor == 0 {
		return nil, fmt.Errorf("Parameters cannot be empty")
	}

	for _, op := range allowedOperators {
		if op == operator {
			goto allowed
		}
	}
	// not allowed
	return nil, fmt.Errorf("Operator not allowed. Only <, >, == allowed")

allowed:

	cond := new(Condition)
	cond.Metric = metric
	cond.Operator = operator
	cond.ActiveFor = activeFor
	cond.Threshhold = threshhold
	cond.IsTrue = cond.compile()

	return cond, nil
}

// SwitchOver is used to configure a switch-over from
// one backend to another. This can be used to gradually
// increase the load to a backend by updating the
// weights of the backends
type SwitchOver struct {
	From         *Backend      // old version
	To           *Backend      // new version
	Conditions   []Condition   // conditions that all need to be met to change
	WeightChange int           // amount of change to the weights
	Wait         time.Duration // duration to wait before changing weights
	Route        *Route        // route for which the switch is defined
	killChan     chan int      // chan to stop the switchover process
}

// Stop the switchover process
func (s *SwitchOver) Stop() {
	s.killChan <- 1
}

// Start the switchover process
func (s *SwitchOver) Start() {

	for {
		select {
		case _ = <-s.killChan:
			return

		default:
			/*
				// s.Route.MetricsRepo.GetMetric(backend)
				// loop over conditions and check if all are met
				for _, condition := range s.Conditions {


						if condition.IsTrue() {
							if condition.triggerTime != time.Time {

							}
						}


					// check
				}
			*/
			time.Sleep(s.Wait)
		}
	}
}
