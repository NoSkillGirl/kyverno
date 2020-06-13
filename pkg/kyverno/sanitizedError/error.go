package sanitizedError

import (
	"fmt"

	"github.com/jimlawless/whereami"
)

type customError struct {
	message string
}

func (c customError) Error() string {
	fmt.Printf("%s\n", whereami.WhereAmI())
	return c.message
}

func New(message string) error {
	fmt.Printf("%s\n", whereami.WhereAmI())
	return customError{message: message}
}

func IsErrorSanitized(err error) bool {
	fmt.Printf("%s\n", whereami.WhereAmI())
	if _, ok := err.(customError); !ok {
		return false
	}
	return true
}
