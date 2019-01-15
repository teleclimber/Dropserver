package timetrack

import (
	"fmt"
	"time"
)

// Track measures the time it takes for a thing to happen
func Track(start time.Time, name string) {
	elapsed := time.Since(start)
	fmt.Printf("%s took %s\n", name, elapsed)
}
