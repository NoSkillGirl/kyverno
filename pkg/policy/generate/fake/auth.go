package fake

import (
	"fmt"
	"github.com/jimlawless/whereami"
)

//FakeAuth providers implementation for testing, retuning true for all operations
type FakeAuth struct {
}

//NewFakeAuth returns a new instance of Fake Auth that returns true for each operation
func NewFakeAuth() *FakeAuth {
	fmt.Printf("%s\n", whereami.WhereAmI())
	a := FakeAuth{}
	return &a
}

// CanICreate returns 'true'
func (a *FakeAuth) CanICreate(kind, namespace string) (bool, error) {
	fmt.Printf("%s\n", whereami.WhereAmI())
	return true, nil
}

// CanIUpdate returns 'true'
func (a *FakeAuth) CanIUpdate(kind, namespace string) (bool, error) {
	fmt.Printf("%s\n", whereami.WhereAmI())
	return true, nil
}

// CanIDelete returns 'true'
func (a *FakeAuth) CanIDelete(kind, namespace string) (bool, error) {
	fmt.Printf("%s\n", whereami.WhereAmI())
	return true, nil
}

// CanIGet returns 'true'
func (a *FakeAuth) CanIGet(kind, namespace string) (bool, error) {
	fmt.Printf("%s\n", whereami.WhereAmI())
	return true, nil
}
