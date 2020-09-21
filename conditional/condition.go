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
	Status      bool                                     `json:"status" yaml:"-"`              // Status if the condition active for long enough and is therefore true
	Metric      string                                   `json:"metric" yaml:"metric"`         // Name of the metric
	Operator    string                                   `json:"operator" yaml:"operator"`     // < > ==
	Threshold   float64                                  `json:"threshold" yaml:"threshold"`   // Threshhold that is checked
	ActiveFor   time.Duration                            `json:"active_for" yaml:"active_for"` // Duration for which the condition has to be met
	TriggerTime time.Time                                `json:"trigger_time" yaml:"-"`        // time the condition was first true
	IsTrue      func(m map[string]float64) (bool, error) `json:"-" yaml:"-"`                   // Condtional function to evaluate condition
}

func (c *Condition) Compile() func(m map[string]float64) (bool, error) {

	switch c.Operator {
	case "<":
		return func(m map[string]float64) (bool, error) {
			if _, found := m[c.Metric]; !found {
				return false, fmt.Errorf("Unknown metric")
			}
			if m[c.Metric] < c.Threshold {
				return true, nil
			}
			return false, nil
		}
	case "==":
		return func(m map[string]float64) (bool, error) {
			if _, found := m[c.Metric]; !found {
				return false, fmt.Errorf("Unknown metric")
			}
			if m[c.Metric] == c.Threshold {
				return true, nil
			}
			return false, nil
		}
	case ">":
		return func(m map[string]float64) (bool, error) {
			if _, found := m[c.Metric]; !found {
				return false, fmt.Errorf("Unknown metric")
			}
			if m[c.Metric] > c.Threshold {
				return true, nil
			}
			return false, nil
		}
	}
	return nil
}

// NewCondition returns a new condition for the given parameters
// Initializes correctly by setting up IsTrue to a conditional function
func NewCondition(metric, operator string, threshhold float64, activeFor time.Duration) *Condition {

	if metric == "" || operator == "" || activeFor == 0 {
		panic(fmt.Errorf("Parameters cannot be empty"))
	}

	for _, op := range allowedOperators {
		if op == operator {
			goto allowed
		}
	}
	// not allowed
	panic(("Operator not allowed. Only <, >, == allowed"))

allowed:

	cond := new(Condition)
	cond.Metric = metric
	cond.Operator = operator
	cond.ActiveFor = activeFor
	cond.Threshold = threshhold
	cond.IsTrue = cond.Compile()

	return cond
}

func (c *Condition) GetActiveFor() time.Duration {
	return time.Duration(c.ActiveFor) * time.Second
}
