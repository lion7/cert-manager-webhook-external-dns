package scheme

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	externaldns "sigs.k8s.io/external-dns/endpoint"
)

var (
	// ExternalDNS allows the CRD to have a configurable group/version/kind.
	// We only support the default value
	ExternalDNSGroupVersion = schema.GroupVersion{
		Group:   "externaldns.k8s.io",
		Version: "v1alpha1",
	}
)

func NewScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	// ExternalDNS does not have a method to just add to the scheme, so we add
	// the types we care about.
	scheme.AddKnownTypes(ExternalDNSGroupVersion,
		&externaldns.DNSEndpoint{},
		&externaldns.DNSEndpointList{},
	)
	metav1.AddToGroupVersion(scheme, ExternalDNSGroupVersion)

	return scheme
}
