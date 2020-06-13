package client

import (
	"fmt"
	"strings"

	openapi_v2 "github.com/googleapis/gnostic/OpenAPIv2"

	"github.com/jimlawless/whereami"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/dynamic/fake"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"
)

const (
	// CSRs CertificateSigningRequest
	CSRs string = "CertificateSigningRequest"
	// Secrets Secret
	Secrets string = "Secret"
	// ConfigMaps ConfigMap
	ConfigMaps string = "ConfigMap"
	// Namespaces Namespace
	Namespaces string = "Namespace"
)

//NewMockClient ---testing utilities
func NewMockClient(scheme *runtime.Scheme, objects ...runtime.Object) (*Client, error) {
	fmt.Printf("%s\n", whereami.WhereAmI())
	client := fake.NewSimpleDynamicClient(scheme, objects...)
	// the typed and dynamic client are initialized with similar resources
	kclient := kubernetesfake.NewSimpleClientset(objects...)
	return &Client{
		client:  client,
		kclient: kclient,
	}, nil

}

// NewFakeDiscoveryClient returns a fakediscovery client
func NewFakeDiscoveryClient(registeredResouces []schema.GroupVersionResource) *fakeDiscoveryClient {
	fmt.Printf("%s\n", whereami.WhereAmI())
	// Load some-preregistd resources
	res := []schema.GroupVersionResource{
		{Version: "v1", Resource: "configmaps"},
		{Version: "v1", Resource: "endpoints"},
		{Version: "v1", Resource: "namespaces"},
		{Version: "v1", Resource: "resourcequotas"},
		{Version: "v1", Resource: "secrets"},
		{Version: "v1", Resource: "serviceaccounts"},
		{Group: "apps", Version: "v1", Resource: "daemonsets"},
		{Group: "apps", Version: "v1", Resource: "deployments"},
		{Group: "apps", Version: "v1", Resource: "statefulsets"},
	}
	registeredResouces = append(registeredResouces, res...)
	return &fakeDiscoveryClient{registeredResouces: registeredResouces}
}

type fakeDiscoveryClient struct {
	registeredResouces []schema.GroupVersionResource
}

func (c *fakeDiscoveryClient) getGVR(resource string) schema.GroupVersionResource {
	fmt.Printf("%s\n", whereami.WhereAmI())
	for _, gvr := range c.registeredResouces {
		if gvr.Resource == resource {
			return gvr
		}
	}
	return schema.GroupVersionResource{}
}

func (c *fakeDiscoveryClient) GetServerVersion() (*version.Info, error) {
	fmt.Printf("%s\n", whereami.WhereAmI())
	return nil, nil
}

func (c *fakeDiscoveryClient) GetGVRFromKind(kind string) schema.GroupVersionResource {
	fmt.Printf("%s\n", whereami.WhereAmI())
	resource := strings.ToLower(kind) + "s"
	return c.getGVR(resource)
}

func (c *fakeDiscoveryClient) FindResource(kind string) (*meta.APIResource, schema.GroupVersionResource, error) {
	fmt.Printf("%s\n", whereami.WhereAmI())
	return nil, schema.GroupVersionResource{}, fmt.Errorf("Not implemented")
}

func (c *fakeDiscoveryClient) OpenAPISchema() (*openapi_v2.Document, error) {
	fmt.Printf("%s\n", whereami.WhereAmI())
	return nil, nil
}

func newUnstructured(apiVersion, kind, namespace, name string) *unstructured.Unstructured {
	fmt.Printf("%s\n", whereami.WhereAmI())
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": apiVersion,
			"kind":       kind,
			"metadata": map[string]interface{}{
				"namespace": namespace,
				"name":      name,
			},
		},
	}
}

func newUnstructuredWithSpec(apiVersion, kind, namespace, name string, spec map[string]interface{}) *unstructured.Unstructured {
	fmt.Printf("%s\n", whereami.WhereAmI())
	u := newUnstructured(apiVersion, kind, namespace, name)
	u.Object["spec"] = spec
	return u
}
