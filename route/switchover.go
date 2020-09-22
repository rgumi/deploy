package route

import (
	"fmt"
	"time"

	"github.com/rgumi/depoy/conditional"
)

var counter int

// SwitchOver is used to configure a switch-over from
// one backend to another. This can be used to gradually
// increase the load to a backend by updating the
// weights of the backends
type SwitchOver struct {
	ID                 int                      `json:"id"`
	From               *Backend                 `json:"from"`
	To                 *Backend                 `json:"to"`
	Status             string                   `json:"status"`
	Conditions         []*conditional.Condition `json:"conditions"`       // conditions that all need to be met to change
	WeightChange       uint8                    `json:"weight_change"`    // amount of change to the weights
	Timeout            time.Duration            `json:"timeout"`          // duration to wait before changing weights
	Route              *Route                   `json:"-"`                // route for which the switch is defined
	Rollback           bool                     `json:"rollback"`         // If Switchover is cancled or aborted, should the weights of backends be reset?
	AllowedFailures    int                      `json:"allowed_failures"` // amount of failures that are allowed before switchover is aborted
	toRollbackWeight   uint8
	fromRollbackWeight uint8
	failureCounter     int
	killChan           chan int // chan to stop the switchover process
}

func NewSwitchOver(
	from, to *Backend,
	route *Route,
	conditions []*conditional.Condition,
	timeout time.Duration,
	weightChange uint8) (*SwitchOver, error) {

	if from.ID == to.ID {
		return nil, fmt.Errorf("from and to cannot be the same entity")
	}
	if from.Weigth < 100 {
		return nil, fmt.Errorf("Weight of Switchover.From cannot be less than 100")
	}
	if to.Weigth > 0 {
		return nil, fmt.Errorf("Weight of Switchover.To must be 0")
	}

	for _, cond := range conditions {
		cond.IsTrue = cond.Compile()
	}

	counter++

	return &SwitchOver{
		ID:           counter,
		From:         from,
		To:           to,
		Status:       "Registered",
		Conditions:   conditions,
		Timeout:      timeout,
		WeightChange: weightChange,
		Route:        route,
		Rollback:     false,
		killChan:     make(chan int, 1),
	}, nil
}

// Stop the switchover process
func (s *SwitchOver) Stop() {

	if s.Status == "Running" {
		s.Status = "Stopped"
	}

	if s.Rollback {
		s.From.UpdateWeight(s.fromRollbackWeight)
		s.To.UpdateWeight(s.toRollbackWeight)
		s.To.updateWeigth()
	}

	s.killChan <- 1
}

// Start the switchover process
func (s *SwitchOver) Start() {
	s.toRollbackWeight = s.To.Weigth
	s.fromRollbackWeight = s.From.Weigth
	s.Status = "Running"
outer:
	for {
		select {
		case _ = <-s.killChan:
			log.Warnf("Killed SwitchOver %v of Route %v", s.ID, s.Route.Name)
			return

		default:
			time.Sleep(s.Timeout)

			metrics, err := s.Route.MetricsRepo.ReadRatesOfBackend(
				s.To.ID, time.Now().Add(-10*time.Second), time.Now())

			if err != nil {
				log.Debugf("Warning in Switchover (%v)", err)
				continue
			}

			for _, condition := range s.Conditions {
				status, err := condition.IsTrue(metrics)
				if err != nil {
					// should never happen
					panic(err)
				}

				if status && s.To.Active {

					if condition.TriggerTime.IsZero() {

						condition.TriggerTime = time.Now()

					} else {

						// check if condition was active for long enough
						if condition.TriggerTime.Add(condition.GetActiveFor()).Before(time.Now()) {
							log.Warnf("Updating status of condition %v %v %v to true", condition.Metric, condition.Operator, condition.Threshold)
							condition.Status = true
						}
					}

				} else {
					// condition is not met, therefore reset triggerTime
					condition.TriggerTime = time.Time{}
					s.failureCounter++

					if s.failureCounter == s.AllowedFailures {
						// failed too often...
						s.Status = "Failed"
						s.Stop()

					}
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
				condition.TriggerTime = time.Time{}
				condition.Status = false
			}

			if s.From.Weigth == 0 || s.To.Weigth == 100 {
				// switchover was successful, all traffic is forwarded to new backend

				log.Warnf("Switchover from %v to %v was successful", s.From.ID, s.To.ID)
				s.Status = "Success"
				s.Stop()
			}
		}
	}
}
