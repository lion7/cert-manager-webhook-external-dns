<p align="center">
  <img src="https://raw.githubusercontent.com/cert-manager/cert-manager/d53c0b9270f8cd90d908460d69502694e1838f5f/logo/logo-small.png" height="256" width="256" alt="cert-manager project logo" />
</p>

# cert-manager - ExternalDNS webhook

This repo allows [cert-manager](https://github.com/cert-manager/cert-manager) to use [ExternalDNS](https://github.com/kubernetes-sigs/external-dns) to handle ACME challenges. 

## Requirements 

### ExternalDNS

The default configuration of ExternalDNS needs altering for this integration to function:
- TXT records are not managed by ExternalDNS by default, it requires an extra flag 
- The DNSEndpoint CRD is not enabled by default 

If you are deploying with the [official Helm chart](https://artifacthub.io/packages/helm/external-dns/external-dns) you can accomplish this by including this in your values file:

```yaml
extraArgs: 
  - --managed-record-types=A      # ┐
  - --managed-record-types=AAAA   # ├ Default values
  - --managed-record-types=CNAME  # ┘
  - --managed-record-types=TXT    # ─ New value

sources:
  - service # ┬ Default values
  - ingress # ┘
  - crd     # ─ New value
```

### cert-manager

Any supported version of cert-manager supports DNS webhooks, for documentation on installing cert-manager see the [official documentation](https://cert-manager.io/docs/installation/)

## Installing

TODO

## Usage

To configure an issuer to use ExternalDNS you just specify the group and solver name within the Issuer or ClusterIssuer config:

```yaml
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: example-issuer
spec:
  acme:
   ...
    solvers:
    - dns01:
        webhook:
          groupName: external-dns.acme.cert-manager.io
          solverName: external-dns
```

## Running the test suite

All DNS providers **must** run the DNS01 provider conformance testing suite,
else they will have undetermined behaviour when used with cert-manager.

**It is essential that you configure and run the test suite when creating a
DNS01 webhook.**

A test file has been provided in [main_test.go](main_test.go).

You can run the test suite with:

```bash
make test
```

The test suite includes a mock DNS server with external-dns controller simulation to provide comprehensive testing without requiring external DNS provider credentials.
