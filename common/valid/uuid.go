package valid

import "github.com/google/uuid"

// IsUUID...
func IsUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

// IsUUIDSlice...
func IsUUIDSlice(ss []string) bool {
	for _, s := range ss {
		if !IsUUID(s) {
			return false
		}
	}
	return true
}
