package metrics

// Metrics contains diffrent metrics which can be collected
type Metrics struct {
	totalRequests    int
	sserrRequests    int
	cserrRequests    int
	successRequests  int
	redirectRequests int
}
