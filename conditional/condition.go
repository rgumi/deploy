package conditional

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
	// Status if the condition active for long enough and is therefore true
	Status bool `json:"status" yaml:"-"`
	// Name of the metric
	Metric string `json:"metric" yaml:"metric"`
	// allowed operators: < > ==
	Operator string `json:"operator" yaml:"operator"`
	// Threshhold that is checked
	Threshold float64 `json:"threshold" yaml:"threshold"`
	// Duration for which the condition has to be met
	ActiveFor time.Duration `json:"active_for" yaml:"activeFor" default:"5s"`
	// Duration for which an active alert needs to be inactive to be resolved
	ResolveIn time.Duration `json:"resolve_in,omitempty" yaml:"resolveIn,omitempty"`
	// time the condition was first true
	TriggerTime time.Time `json:"-" yaml:"-"`
	// Condtional function to evaluate condition
	IsTrue func(m map[string]float64) bool `json:"-" yaml:"-"`
}

func (c *Condition) Compile() func(m map[string]float64) {

	switch c.Operator {
	case "<":
		c.IsTrue = func(m map[string]float64) bool {
			if value, found := m[c.Metric]; found && value < c.Threshold {
				return true
			}
			return false
		}

	case "==":
		c.IsTrue = func(m map[string]float64) bool {
			if value, found := m[c.Metric]; found && value == c.Threshold {
				return true
			}
			return false
		}

	case ">":
		c.IsTrue = func(m map[string]float64) bool {
			if value, found := m[c.Metric]; found && value > c.Threshold {
				return true
			}
			return false
		}
	}
	return nil
}

// NewCondition returns a new condition for the given parameters
// Initializes correctly by setting up IsTrue to a conditional function
func NewCondition(metric, operator string, threshhold float64, activeFor, resolveIn time.Duration) *Condition {

	if metric == "" || operator == "" || activeFor == 0 {
		panic(fmt.Errorf("Parameters cannot be empty"))
	}
	for _, op := range allowedOperators {
		if op == operator {
			goto allowed
		}
	}
	// not allowed
	panic(fmt.Errorf("Operator not allowed. Only <, >, == allowed"))

allowed:

	cond := new(Condition)
	cond.Metric = metric
	cond.Operator = operator
	cond.ActiveFor = activeFor
	cond.ResolveIn = resolveIn
	cond.Threshold = threshhold
	cond.Compile()

	return cond
}

func (c *Condition) GetActiveFor() time.Duration {
	return c.ActiveFor
}
