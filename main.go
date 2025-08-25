package main

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"
	logf "github.com/cert-manager/cert-manager/pkg/logs"
	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/external-dns/endpoint"
	externaldns "sigs.k8s.io/external-dns/endpoint"

	"github.com/lion7/cert-manager-webhook-external-dns/internal/scheme"
	contextutil "github.com/lion7/cert-manager-webhook-external-dns/internal/util/context"
)

var (
	// ProviderName is the presented name of the provider
	ProviderName = "external-dns"

	// GroupName is the Kubernetes group name that will be forwarded to this
	// extension-apiserver.
	GroupName = "external-dns.acme.cert-manager.io"

	// RequestTimeout is the timeout for each request
	RequestTimeout = time.Second * 5
)

func main() {
	cmd.RunWebhookServer(GroupName,
		&externalDNSProviderSolver{},
	)
}

// externalDNSProviderSolver implements the provider-specific logic needed to
// 'present' an ACME challenge TXT record for your own DNS provider.
// To do so, it must implement the `github.com/cert-manager/cert-manager/pkg/acme/webhook.Solver`
// interface.
type externalDNSProviderSolver struct {
	client client.Client
	ctx    context.Context
}

// Name is used as the name for this DNS solver when referencing it on the ACME
// Issuer resource.
// This should be unique **within the group name**, i.e. you can have two
// solvers configured with the same Name() **so long as they do not co-exist
// within a single webhook deployment**.
// For example, `cloudflare` may be used as the name of a solver.
func (c *externalDNSProviderSolver) Name() string {
	return ProviderName
}

// Present is responsible for actually presenting the DNS record with the
// DNS provider.
// This method should tolerate being called multiple times with the same value.
// cert-manager itself will later perform a self check to ensure that the
// solver has correctly configured the DNS provider.
func (c *externalDNSProviderSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	// Create child context that times out
	ctx, cancel := context.WithTimeout(c.ctx, RequestTimeout)
	defer cancel()

	// Fail early if the challenge provided contains bad config
	providerSpecific, err := loadProviderSpecificConfig(ch.Config)
	if err != nil {
		return fmt.Errorf("could not load provider specific config: %w", err)
	}

	// Create the DNSEndpoint object, we just define the metadata here so the
	// actual update can happen in the controllerutil.CreateOrPatch call below
	dnsEndpoint := externaldns.DNSEndpoint{
		ObjectMeta: metav1.ObjectMeta{
			Name:      generateName(ch),
			Namespace: ch.ResourceNamespace,
		},
	}

	result, err := controllerutil.CreateOrPatch(ctx, c.client, &dnsEndpoint, func() error {
		ep := endpoint.NewEndpoint(ch.ResolvedFQDN, endpoint.RecordTypeTXT, ch.Key).WithSetIdentifier(dnsEndpoint.Name)
		ep.ProviderSpecific = providerSpecific
		dnsEndpoint.Spec.Endpoints = []*endpoint.Endpoint{ep}
		return nil
	})

	if err != nil {
		return fmt.Errorf("could not create/patch DNSEndpoint object: %w", err)
	}

	//exhaustive:ignore
	switch result {
	case controllerutil.OperationResultCreated:
		logf.Log.Info("created DNSEndpoint object", "request", ch.UID, "namespace", dnsEndpoint.Namespace, "name", dnsEndpoint.Name)
	case controllerutil.OperationResultUpdated:
		logf.Log.Info("updated DNSEndpoint object", "request", ch.UID, "namespace", dnsEndpoint.Namespace, "name", dnsEndpoint.Name)
	case controllerutil.OperationResultNone:
		// Object already existed and was up to date
	}

	return nil
}

// CleanUp should delete the relevant TXT record from the DNS provider console.
// If multiple TXT records exist with the same record name (e.g.
// _acme-challenge.example.com) then **only** the record with the same `key`
// value provided on the ChallengeRequest should be cleaned up.
// This is in order to facilitate multiple DNS validations for the same domain
// concurrently.
func (c *externalDNSProviderSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	// Create child context that times out
	ctx, cancel := context.WithTimeout(c.ctx, RequestTimeout)
	defer cancel()

	// Create object with just the namespace and name for the Delete method
	dnsEndpoint := externaldns.DNSEndpoint{
		ObjectMeta: metav1.ObjectMeta{
			Name:      generateName(ch),
			Namespace: ch.ResourceNamespace,
		},
	}

	// Delete the object, we do not care if the object does not exist
	err := c.client.Delete(ctx, &dnsEndpoint)
	if client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("could not delete DNSEndpoint object: %w", err)
	}

	return nil
}

// Initialize will be called when the webhook first starts.
// This method can be used to instantiate the webhook, i.e. initialising
// connections or warming up caches.
// Typically, the kubeClientConfig parameter is used to build a Kubernetes
// client that can be used to fetch resources from the Kubernetes API, e.g.
// Secret resources containing credentials used to authenticate with DNS
// provider accounts.
// The stopCh can be used to handle early termination of the webhook, in cases
// where a SIGTERM or similar signal is sent to the webhook process.
func (c *externalDNSProviderSolver) Initialize(config *rest.Config, stopCh <-chan struct{}) error {
	cli, err := client.New(config, client.Options{Scheme: scheme.NewScheme()})
	if err != nil {
		return fmt.Errorf("could not create kubernetes client: %w", err)
	}

	c.client = cli
	c.ctx = contextutil.StopChannelContext(stopCh)

	return nil
}

// generateName returns a short, consistent Kubernetes object name for a given
// challenge
func generateName(ch *v1alpha1.ChallengeRequest) string {
	// The the name always starts with the domain name for readability, we
	// remove any wildcards as they are invalid characters for a Kubernetes
	// object name. We do not care about collisions in the prefix as a suffix
	// with a hash is appended.
	prefix := strings.TrimPrefix(
		strings.TrimSuffix(ch.DNSName, "."), "*.") + "-"

	// Generate a hash off the key for a consistent suffix
	hash := fnv.New32()
	fmt.Fprint(hash, ch.Key)
	return prefix + rand.SafeEncodeString(fmt.Sprint(hash.Sum32()))
}

// loadProviderSpecificConfig is a small helper function that decodes JSON
// configuration into the typed config struct.
func loadProviderSpecificConfig(cfgJSON *extapi.JSON) (endpoint.ProviderSpecific, error) {
	type providerSpecificConfig struct {
		ProviderSpecific endpoint.ProviderSpecific `json:"providerSpecific,omitempty"`
	}

	if cfgJSON == nil {
		return nil, nil
	}

	var providerSpecific providerSpecificConfig
	if err := json.Unmarshal(cfgJSON.Raw, &providerSpecific); err != nil {
		return nil, fmt.Errorf("error decoding solver config: %v", err)
	}

	return providerSpecific.ProviderSpecific, nil
}
