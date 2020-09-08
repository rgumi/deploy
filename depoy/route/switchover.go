package route

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

// the metrics which are allowed for the condtions
var allowedOperators = []string{">", "==", "<"}

// Condition is used to evaluate the state
// of a backend and take action according to
// the values defined here
type Condition struct {
	Status      bool                                     `json:"status"`       // Status if the condition active for long enough and is therefore true
	Metric      string                                   `json:"metric"`       // Name of the metric
	Operator    string                                   `json:"operator"`     // < > ==
	Threshhold  float64                                  `json:"threshold"`    // Threshhold that is checked
	ActiveFor   time.Duration                            `json:"active_for"`   // Duration for which the condition has to be met
	triggerTime time.Time                                `json:"trigger_time"` // time the condition was first true
	IsTrue      func(m map[string]float64) (bool, error) `json:"-"`            // Condtional function to evaluate condition
}

func (c *Condition) compile() func(m map[string]float64) (bool, error) {

	switch c.Operator {
	case "<":
		return func(m map[string]float64) (bool, error) {
			if _, found := m[c.Metric]; !found {
				return false, fmt.Errorf("Unknown metric")
			}
			if m[c.Metric] < c.Threshhold {
				return true, nil
			}
			return false, nil
		}
	case "==":
		return func(m map[string]float64) (bool, error) {
			if _, found := m[c.Metric]; !found {
				return false, fmt.Errorf("Unknown metric")
			}
			if m[c.Metric] == c.Threshhold {
				return true, nil
			}
			return false, nil
		}
	case ">":
		return func(m map[string]float64) (bool, error) {
			if _, found := m[c.Metric]; !found {
				return false, fmt.Errorf("Unknown metric")
			}
			if m[c.Metric] > c.Threshhold {
				return true, nil
			}
			return false, nil
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
	From               *Backend      `json:"from"`          // old version
	To                 *Backend      `json:"to"`            // new version
	Conditions         []*Condition  `json:"conditions"`    // conditions that all need to be met to change
	WeightChange       uint8         `json:"weight_change"` // amount of change to the weights
	Timeout            time.Duration `json:"timeout"`       // duration to wait before changing weights
	Route              *Route        `json:"-"`             // route for which the switch is defined
	killChan           chan int      `json:"-"`             // chan to stop the switchover process
	toRollbackWeight   uint8         `json:"-"`
	fromRollbackWeight uint8         `json:"-"`
	Rollback           bool          `json:"rollback"`         // If Switchover is cancled or aborted, should the weights of backends be reset?
	AllowedFailures    int           `json:"allowed_failures"` // amount of failures that are allowed before switchover is aborted
}

// Stop the switchover process
func (s *SwitchOver) Stop() {
	s.killChan <- 1
}

// Start the switchover process
func (s *SwitchOver) Start() {

outer:
	for {
		select {
		case _ = <-s.killChan:
			log.Warnf("Killed SwitchOver %v", s)
			return

		default:
			time.Sleep(s.Timeout)

			metrics, err := s.Route.MetricsRepo.Storage.ReadRatesOfBackend(
				s.To.ID, time.Now().Add(-10*time.Second), time.Now())

			if err != nil {
				log.Error(err)
				continue
			}

			for _, condition := range s.Conditions {
				status, err := condition.IsTrue(metrics)
				if err != nil {
					// TODO: Handle this somehow
					panic(err)
				}
				if status && s.To.Active {

					if condition.triggerTime.IsZero() {

						condition.triggerTime = time.Now()

					} else {

						// check if condition was active for long enough
						if condition.triggerTime.Add(condition.ActiveFor).Before(time.Now()) {
							log.Warnf("Updating status of condition %v %v %v to true", condition.Metric, condition.Operator, condition.Threshhold)
							condition.Status = true
						}
					}

				} else {
					// condition is not met, therefore reset triggerTime
					condition.triggerTime = time.Time{}
				}
			}

			// check
			for _, condition := range s.Conditions {
				if !condition.Status {
					// if any condition is not true, continue
					log.Debugf("A condition is false. (%v)", s)
					continue outer
				}
			}

			// if all conditions are true, increase the weight of the new route
			s.From.UpdateWeight(s.From.Weigth - s.WeightChange)
			s.To.UpdateWeight(s.To.Weigth + s.WeightChange)

			// As both routes are part of the same route, both will be updated
			s.To.updateWeigth()

			// reset the conditions
			for _, condition := range s.Conditions {
				condition.triggerTime = time.Time{}
				condition.Status = false
			}

			if s.From.Weigth == 0 {
				// switchover was successful, all traffic is forwarded to new backend

				log.Warnf("Switchover from %v to %v was successful", s.From, s.To)
				s.Stop()
			}
		}
	}
}
