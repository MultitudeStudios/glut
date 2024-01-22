package sqlutil

import "fmt"

// Like...
func Like(value string) string {
	return fmt.Sprintf("%%%s%%", value)
}
