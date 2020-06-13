package generate

import (
	"fmt"

	"github.com/jimlawless/whereami"
	kyverno "github.com/nirmata/kyverno/pkg/api/kyverno/v1"
	dclient "github.com/nirmata/kyverno/pkg/dclient"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func getResource(client *dclient.Client, resourceSpec kyverno.ResourceSpec) (*unstructured.Unstructured, error) {
	fmt.Printf("%s\n", whereami.WhereAmI())
	return client.GetResource(resourceSpec.Kind, resourceSpec.Namespace, resourceSpec.Name)
}
