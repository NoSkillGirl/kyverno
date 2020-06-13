package generate

import (
	"fmt"

	"github.com/jimlawless/whereami"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func manageLabels(unstr *unstructured.Unstructured, triggerResource unstructured.Unstructured) {
	fmt.Printf("%s\n", whereami.WhereAmI())
	// add managedBY label if not defined
	labels := unstr.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}

	// handle managedBy label
	managedBy(labels)
	// handle generatedBy label
	generatedBy(labels, triggerResource)

	// update the labels
	unstr.SetLabels(labels)
}

func managedBy(labels map[string]string) {
	fmt.Printf("%s\n", whereami.WhereAmI())
	// ManagedBy label
	key := "app.kubernetes.io/managed-by"
	value := "kyverno"
	val, ok := labels[key]
	if ok {
		if val != value {
			log.Log.Info(fmt.Sprintf("resource managed by %s, kyverno wont over-ride the label", val))
			return
		}
	}
	if !ok {
		// add label
		labels[key] = value
	}
}

func generatedBy(labels map[string]string, triggerResource unstructured.Unstructured) {
	fmt.Printf("%s\n", whereami.WhereAmI())
	key := "kyverno.io/generated-by"
	value := fmt.Sprintf("%s-%s-%s", triggerResource.GetKind(), triggerResource.GetNamespace(), triggerResource.GetName())
	val, ok := labels[key]
	if ok {
		if val != value {
			log.Log.Info(fmt.Sprintf("resource generated by %s, kyverno wont over-ride the label", val))
			return
		}
	}
	if !ok {
		// add label
		labels[key] = value
	}
}
