package main

import (
	"fmt"
)

// what do we do in here?
// start a server that ds-host will forward connections to.
// In here we do:
// - create and delete sandbox data dirs
// - create and manage app-spaces and app code
// - perform the operations on app-space data that do not require custom code
//   ..like crud on DB

// For now the main function is to manage app-spaces and app code. Meaning:
// - create new app-space dirs with appropriate permissions
// - create new app code dir, with appropriate permissions, (owner, group?)
// -

func main() {
	fmt.Println("Hello from ds-trusted")
}
