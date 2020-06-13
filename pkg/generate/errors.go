package generate

import (
	"fmt"

	"github.com/jimlawless/whereami"
)

// ParseFailed stores the resource that failed to parse
type ParseFailed struct {
	spec       interface{}
	parseError error
}

func (e *ParseFailed) Error() string {
	fmt.Printf("%s\n", whereami.WhereAmI())
	return fmt.Sprintf("failed to parse the resource spec %v: %v", e.spec, e.parseError.Error())
}

//NewParseFailed returns a new ParseFailed error
func NewParseFailed(spec interface{}, err error) *ParseFailed {
	fmt.Printf("%s\n", whereami.WhereAmI())
	return &ParseFailed{spec: spec, parseError: err}
}

//Violation stores the rule that violated
type Violation struct {
	rule string
	err  error
}

func (e *Violation) Error() string {
	fmt.Printf("%s\n", whereami.WhereAmI())
	return fmt.Sprintf("creating Violation; error %s", e.err)
}

//NewViolation returns a new Violation error
func NewViolation(rule string, err error) *Violation {
	fmt.Printf("%s\n", whereami.WhereAmI())
	return &Violation{rule: rule, err: err}
}

// NotFound stores the resource that was not found
type NotFound struct {
	kind      string
	namespace string
	name      string
}

func (e *NotFound) Error() string {
	fmt.Printf("%s\n", whereami.WhereAmI())
	return fmt.Sprintf("resource %s/%s/%s not present", e.kind, e.namespace, e.name)
}

//NewNotFound returns a new NotFound error
func NewNotFound(kind, namespace, name string) *NotFound {
	fmt.Printf("%s\n", whereami.WhereAmI())
	return &NotFound{kind: kind, namespace: namespace, name: name}
}

//ConfigNotFound stores the config information
type ConfigNotFound struct {
	config    interface{}
	kind      string
	namespace string
	name      string
}

func (e *ConfigNotFound) Error() string {
	fmt.Printf("%s\n", whereami.WhereAmI())
	return fmt.Sprintf("configuration %v, not present in resource %s/%s/%s", e.config, e.kind, e.namespace, e.name)
}

//NewConfigNotFound returns a new NewConfigNotFound error
func NewConfigNotFound(config interface{}, kind, namespace, name string) *ConfigNotFound {
	fmt.Printf("%s\n", whereami.WhereAmI())
	return &ConfigNotFound{config: config, kind: kind, namespace: namespace, name: name}
}
