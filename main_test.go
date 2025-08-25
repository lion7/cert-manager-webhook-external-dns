package main

import (
	"net"
	"testing"
	"time"

	acmetest "github.com/cert-manager/cert-manager/test/acme"
	"github.com/go-logr/logr/testr"
	inttesting "github.com/lion7/cert-manager-webhook-external-dns/testing"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const DNSPort = "59351"

func TestRunsSuite(t *testing.T) {
	log.SetLogger(testr.New(t))

	// Create a solver and registry that runs a background external-dns
	solver := inttesting.NewTestSolver(t, &externalDNSProviderSolver{}, net.JoinHostPort("", DNSPort), "example.com.")

	// Uncomment the below fixture when implementing your custom DNS provider
	fixture := acmetest.NewFixture(solver,
		acmetest.SetResolvedZone("example.com."),
		acmetest.SetAllowAmbientCredentials(false),
		acmetest.SetConfig(map[string]any{}),
		acmetest.SetDNSServer(net.JoinHostPort("127.0.0.1", DNSPort)),
		acmetest.SetUseAuthoritative(false),
		acmetest.SetPropagationLimit(time.Second*30),
	)

	// TODO: Uncomment RunConformance and delete RunBasic and RunExtended once https://github.com/cert-manager/cert-manager/pull/4835 is merged
	fixture.RunBasic(t)
	fixture.RunExtended(t)
	// fixture.RunConformance(t)
}
