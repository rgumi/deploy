package route

import (
	"depoy/conditional"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

// SwitchOver is used to configure a switch-over from
// one backend to another. This can be used to gradually
// increase the load to a backend by updating the
// weights of the backends
type SwitchOver struct {
	From               *Backend                 `json:"from"`          // old version
	To                 *Backend                 `json:"to"`            // new version
	Conditions         []*conditional.Condition `json:"conditions"`    // conditions that all need to be met to change
	WeightChange       uint8                    `json:"weight_change"` // amount of change to the weights
	Timeout            time.Duration            `json:"timeout"`       // duration to wait before changing weights
	Route              *Route                   `json:"-"`             // route for which the switch is defined
	killChan           chan int                 `json:"-"`             // chan to stop the switchover process
	toRollbackWeight   uint8                    `json:"-"`
	fromRollbackWeight uint8                    `json:"-"`
	Rollback           bool                     `json:"rollback"`         // If Switchover is cancled or aborted, should the weights of backends be reset?
	AllowedFailures    int                      `json:"allowed_failures"` // amount of failures that are allowed before switchover is aborted
	failureCounter     int
}

// Stop the switchover process
func (s *SwitchOver) Stop() {
	s.killChan <- 1
}

// Start the switchover process
func (s *SwitchOver) Start() {

	if s.From.Weigth < 100 {
		panic(fmt.Errorf("Weight of Switchover.From cannot be less than 100"))
	}

	if s.To.Weigth > 0 {
		panic(fmt.Errorf("Weight of Switchover.From cannot be greater than 0"))
	}
	s.toRollbackWeight = s.To.Weigth
	s.fromRollbackWeight = s.From.Weigth

outer:
	for {
		select {
		case _ = <-s.killChan:
			log.Warnf("Killed SwitchOver %v", s)
			return

		default:
			time.Sleep(s.Timeout)

			metrics, err := s.Route.MetricsRepo.ReadRatesOfBackend(
				s.To.ID, time.Now().Add(-10*time.Second), time.Now())

			if err != nil {
				log.Warnf("Warning in Switchover (%v)", err)
				continue
			}

			for _, condition := range s.Conditions {
				status, err := condition.IsTrue(metrics)
				if err != nil {
					// TODO: Handle this somehow
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

					if s.failureCounter == s.AllowedFailures {
						// failed too often...

					}
					s.failureCounter++
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

			if s.From.Weigth == 0 {
				// switchover was successful, all traffic is forwarded to new backend

				log.Warnf("Switchover from %v to %v was successful", s.From, s.To)
				s.Stop()
			}
		}
	}
}
