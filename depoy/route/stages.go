package route

import "time"

type Stage struct {
	from           *Backend
	to             *Backend
	activeTime     time.Duration
	allowedErrors  int
	increaseWeight int
	decreaseWeight int
}
