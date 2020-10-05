package route

import (
	"fmt"
	"time"

	"github.com/rgumi/depoy/conditional"
	log "github.com/sirupsen/logrus"
)

var counter int
var granularity = 10 * time.Second

// Switchover is used to configure a switch-over from
// one backend to another. This can be used to gradually
// increase the load to a backend by updating the
// weights of the backends
type Switchover struct {
	ID                 int                      `json:"id"`
	From               *Backend                 `json:"from"`
	To                 *Backend                 `json:"to"`
	Status             string                   `json:"status"`
	Conditions         []*conditional.Condition `json:"conditions"`    // conditions that all need to be met to change
	WeightChange       uint8                    `json:"weight_change"` // amount of change to the weights
	Timeout            time.Duration            `json:"-"`             // duration to wait before changing weights
	Route              *Route                   `json:"-"`             // route for which the switch is defined
	Rollback           bool                     `json:"-"`             // If Switchover is cancled or aborted, should the weights of backends be reset?
	AllowedFailures    int                      `json:"-"`             // amount of failures that are allowed before switchover is aborted
	FailureCounter     int                      `json:"-"`
	toRollbackWeight   uint8
	fromRollbackWeight uint8
	killChan           chan int // chan to stop the switchover process
}

func NewSwitchover(
	from, to *Backend,
	route *Route,
	conditions []*conditional.Condition,
	timeout time.Duration,
	allowedFailures int,
	weightChange uint8, rollback bool) (*Switchover, error) {

	if from.ID == to.ID {
		return nil, fmt.Errorf("from and to cannot be the same entity")
	}
	if from.Weigth < to.Weigth {
		return nil, fmt.Errorf("Weight of Switchover.From must be larger then Switchover.To")
	}

	for _, cond := range conditions {
		cond.Compile()
	}

	counter++
	return &Switchover{
		ID:              counter,
		From:            from,
		To:              to,
		Status:          "Registered",
		Conditions:      conditions,
		Timeout:         timeout,
		WeightChange:    weightChange,
		AllowedFailures: allowedFailures,
		Route:           route,
		Rollback:        rollback,
		killChan:        make(chan int, 1),
	}, nil
}

// Stop the switchover process
func (s *Switchover) Stop() {

	if s.Status == "Running" {
		s.Status = "Stopped"
	}

	if s.Rollback && s.Status == "Failed" {
		log.Warnf("Switchover from %v to %v failed", s.From.ID, s.To.ID)
		s.From.UpdateWeight(s.fromRollbackWeight)
		s.To.UpdateWeight(s.toRollbackWeight)
		s.To.updateWeigth()
	}
	s.killChan <- 1
}

// Start the switchover process
func (s *Switchover) Start() {
	s.toRollbackWeight = s.To.Weigth
	s.fromRollbackWeight = s.From.Weigth
	s.Status = "Running"
outer:
	for {
		select {
		case _ = <-s.killChan:
			log.Warnf("Killed SwitchOver %v of Route %v", s.ID, s.Route.Name)
			return

		case _ = <-time.After(s.Timeout):
			metrics, err := s.Route.MetricsRepo.ReadRatesOfBackend(
				s.To.ID, time.Now().Add(-s.Timeout), time.Now())
			if err != nil {
				log.Trace(err)
				continue
			}
			// begin cycle => check each condition if true
			for _, condition := range s.Conditions {
				status := condition.IsTrue(metrics)
				if status && s.To.Active {
					if condition.TriggerTime.IsZero() {
						condition.TriggerTime = time.Now()
					} else {
						// check if condition was active for long enough
						if condition.TriggerTime.Add(condition.GetActiveFor()).Before(time.Now()) {
							log.Debugf("Updating status of condition %v %v %v to true", condition.Metric, condition.Operator, condition.Threshold)
							condition.Status = true
						}
					}
				} else {
					// condition is not met, therefore reset triggerTime
					condition.TriggerTime = time.Time{}
				}
			}
			// end of cycle, check conditions
			for _, condition := range s.Conditions {
				if !condition.Status {
					// if any condition is not true, cycle is failed
					log.Debugf("A condition is false. (%v)", s)
					s.FailureCounter++
					if s.FailureCounter > s.AllowedFailures {
						// failed too often...
						s.Status = "Failed"
						s.Stop()
					}
					// continue cycle
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
			if s.From.Weigth <= 0 || s.To.Weigth >= 100 {
				// switchover was successful, all traffic is forwarded to new backend
				log.Infof("Switchover from %v to %v was successful", s.From.ID, s.To.ID)
				s.Status = "Success"
				s.Stop()
			}
		}
	}
}
