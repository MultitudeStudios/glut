package sqlutil

import "fmt"

// Exists...
func Exists(sql string) string {
	return fmt.Sprintf("SELECT EXISTS (%s)", sql)
}
