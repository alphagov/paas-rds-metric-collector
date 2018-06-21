package utils

import (
	"math/rand"
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

func RandomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyz")
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
