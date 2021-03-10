package stdinput

import (
	"bufio"
	"fmt"
	"os"
)

// StdInput is a placeholder struct for interface
type StdInput struct{}

//ReadLine reads a line of text from stdin.
func (s *StdInput) ReadLine(label string) string {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print(label)
	// Scans a line from Stdin(Console)
	scanner.Scan()
	// Holds the string that scanned
	text := scanner.Text()

	return text
}
