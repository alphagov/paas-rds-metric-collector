package utils

// SliceContainsString ...
func SliceContainsString(s []string, v string) bool {
	for _, value := range s {
		if value == v {
			return true
		}
	}

	return false
}
