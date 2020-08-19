package route

import "sync/atomic"

type Strategy interface {
	GetTargetIndex() int
	Update(int)
	SetTargets(int)
}

type RoundRobin struct {
	max  int
	next uint32
}

func NewRoundRobin(lenTargets int) *RoundRobin {
	return &RoundRobin{
		max:  lenTargets,
		next: 0,
	}
}

func (r *RoundRobin) GetTargetIndex() int {
	n := atomic.AddUint32(&r.next, 1)
	return (int(n) - 1) % r.max
}

func (r *RoundRobin) Update(prob int) {
	return
}
func (r *RoundRobin) SetTargets(lenTargets int) {
	r.max = lenTargets
}

type Canary struct {
	max       int
	next      uint32
	firstProb int
}

func NewCanary(firstProb int) *Canary {
	return &Canary{
		max:       10,
		next:      0,
		firstProb: firstProb,
	}
}

func (c *Canary) GetTargetIndex() int {
	n := atomic.AddUint32(&c.next, 1)
	if (int(n)-1)%c.max < c.firstProb {
		return 0
	}
	return 1
}

func (c *Canary) Update(prob int) {
	c.firstProb = prob
}

func (c *Canary) SetTargets(lenTargets int) {
	return
}
