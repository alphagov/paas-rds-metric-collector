package utils

import (
	"time"
)

// SliceContainsString ...
func SliceContainsString(s []string, v string) bool {
	for _, value := range s {
		if value == v {
			return true
		}
	}

	return false
}

func WithTimeout(timeout time.Duration, payload func()) bool {
	ch := make(chan bool, 1)
	timeoutHappened := false
	defer close(ch)

	go func() {
		payload()
		if !timeoutHappened {
			ch <- true
		}
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-ch:
		return false
	case <-timer.C:
		timeoutHappened = true
	}
	return true
}
