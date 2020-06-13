package variables

import (
	"fmt"
	"regexp"

	"github.com/jimlawless/whereami"
)

//IsVariable returns true if the element contains a 'valid' variable {{}}
func IsVariable(element string) bool {
	fmt.Printf("%s\n", whereami.WhereAmI())
	validRegex := regexp.MustCompile(variableRegex)
	groups := validRegex.FindAllStringSubmatch(element, -1)
	return len(groups) != 0
}
