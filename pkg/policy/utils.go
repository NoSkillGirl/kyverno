package policy

import (
	"fmt"

	"github.com/jimlawless/whereami"
)

//Contains Check if strint is contained in a list of string
func containString(list []string, element string) bool {
	fmt.Printf("%s\n", whereami.WhereAmI())
	for _, e := range list {
		if e == element {
			return true
		}
	}
	return false
}
