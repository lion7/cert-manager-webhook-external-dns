package testing

import (
	"context"
	"testing"
	"time"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/external-dns/controller"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
	"sigs.k8s.io/external-dns/source"
	"sigs.k8s.io/yaml"

	"github.com/lion7/cert-manager-webhook-external-dns/internal/scheme"
	contextutil "github.com/lion7/cert-manager-webhook-external-dns/internal/util/context"

	_ "embed"
)

// Solver wraps a provided cert-manager webhook solver, adding an extra step to
// the Initialize method to ensure the external-dns CRDs are installed.
//
// It will also start the external-dns control loop once the underlying Solver
// is initialized.
type Solver struct {
	webhook.Solver
	*Registry
}

func NewTestSolver(t *testing.T, solver webhook.Solver, dnsAddr string, domains ...string) *Solver {
	return &Solver{
		Solver:   solver,
		Registry: NewTestRegistry(t, dnsAddr, domains...),
	}
}

func (s *Solver) Initialize(config *rest.Config, stopCh <-chan struct{}) error {
	// Ensure the CRDs are installed
	if err := s.installCRDs(config); err != nil {
		return err
	}

	// Initialize the underlying solver
	if err := s.Solver.Initialize(config, stopCh); err != nil {
		return err
	}

	// Create scheme
	runtimeScheme := scheme.NewScheme()

	// Create config for interacting with the CRD
	// https://github.com/kubernetes-sigs/external-dns/blob/55a54b9e3ac858ce4818e5ececd0b151f20c136c/source/crd.go#L99-L101
	configCopy := *config
	configCopy.ContentConfig.GroupVersion = &scheme.ExternalDNSGroupVersion
	configCopy.APIPath = "/apis"
	configCopy.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: serializer.NewCodecFactory(runtimeScheme)}

	// Create client for interacting with the external-dns group/version
	restClient, err := rest.UnversionedRESTClientFor(&configCopy)
	if err != nil {
		return err
	}

	// Create source for new config
	crdSource, err := source.NewCRDSource(restClient, "", "DNSEndpoint", "", labels.Everything(), runtimeScheme, false)
	if err != nil {
		return err
	}

	// Setup the external-dns controller
	externalDNS := controller.Controller{
		Source:             crdSource,
		Registry:           s.Registry,
		Policy:             &plan.SyncPolicy{},
		DomainFilter:       s.Registry.domains,
		ManagedRecordTypes: []string{endpoint.RecordTypeA, endpoint.RecordTypeAAAA, endpoint.RecordTypeCNAME, endpoint.RecordTypeTXT},
	}

	// Run external-dns in the background, stopping via context cancellation
	// when the webhook is stopped
	ctx := contextutil.StopChannelContext(stopCh)

	// Start external-dns, we cant use externalDNS.Run as this can perform a
	// fatal log that ends the test when shutting down
	go wait.UntilWithContext(ctx, func(ctx context.Context) {
		_ = externalDNS.RunOnce(ctx)
	}, time.Second)

	// We cant capture externalDNS errors in a nice way, so we will just have
	// to rely on failing tests to tell us something is wrong
	return nil
}

//go:embed dnsendpoint.customresourcedefinitions.yaml
var dnsEndpointManifest []byte

func (s *Solver) installCRDs(config *rest.Config) error {
	var manifest apiextensionsv1.CustomResourceDefinition
	if err := yaml.Unmarshal(dnsEndpointManifest, &manifest); err != nil {
		return err
	}

	_, err := envtest.InstallCRDs(config, envtest.CRDInstallOptions{
		CRDs: []*apiextensionsv1.CustomResourceDefinition{&manifest},
	})

	return err
}
