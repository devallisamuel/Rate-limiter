package limiter

import ("time"; "fmt")

type Limiter struct {
	limit      int
	windowSize time.Duration
	counter    int
	startTime  time.Time
}

func New(limit int, windowSize time.Duration) *Limiter {
	return &Limiter{
		limit,
		windowSize,
		0,
		time.Now(),
	}
}

func (param *Limiter) Allow() bool {
	hasTimeElapsed := hasElapsed(*param)

	fmt.Println(!hasTimeElapsed )

	if hasTimeElapsed {
		param.counter = 0
		param.startTime = time.Now()
	} 
	if param.counter < param.limit {
		param.counter++
		return true
	}
	return false;
}

func hasElapsed(param Limiter) bool {
	return time.Since(param.startTime) > param.windowSize 

}
