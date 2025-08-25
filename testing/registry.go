package testing

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/miekg/dns"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
)

// Registry implements an sigs.k8s.io/external-dns/registry.Registry by hosting
// a local DNS server. This allows us to test directly against ourselves over
// localhost.
type Registry struct {
	mu      sync.RWMutex
	records map[endpoint.EndpointKey]*endpoint.Endpoint
	domains *endpoint.DomainFilter
}

func NewTestRegistry(t *testing.T, addr string, domains ...string) *Registry {
	domainFilter := endpoint.NewDomainFilter(domains)
	registry := &Registry{
		domains: &domainFilter,
		records: make(map[endpoint.EndpointKey]*endpoint.Endpoint),
	}

	server := &dns.Server{
		Addr:    addr,
		Net:     "udp",
		Handler: dns.HandlerFunc(registry.handleDNSRequest),
	}

	go func() {
		_ = server.ListenAndServe()
	}()

	t.Cleanup(func() {
		_ = server.Shutdown()
	})

	return registry
}

func (r *Registry) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	records := make([]*endpoint.Endpoint, 0, len(r.records))
	for _, record := range r.records {
		records = append(records, record.DeepCopy())
	}

	return records, nil
}

func (r *Registry) ApplyChanges(ctx context.Context, changes *plan.Changes) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, change := range changes.Create {
		r.records[change.Key()] = change.DeepCopy()
	}

	for _, change := range changes.UpdateNew {
		r.records[change.Key()] = change.DeepCopy()
	}

	for _, change := range changes.Delete {
		delete(r.records, change.Key())
	}

	return nil
}

func (r *Registry) AdjustEndpoints(endpoints []*endpoint.Endpoint) ([]*endpoint.Endpoint, error) {
	return endpoints, nil
}

func (r *Registry) GetDomainFilter() endpoint.DomainFilterInterface {
	return r.domains
}

func (r *Registry) OwnerID() string {
	return ""
}

func (r *Registry) handleDNSRequest(w dns.ResponseWriter, req *dns.Msg) {
	msg := new(dns.Msg)
	msg.SetReply(req)
	if req.Opcode == dns.OpcodeQuery {
		for _, q := range msg.Question {
			if err := r.addDNSAnswer(q, msg, req); err != nil {
				msg.SetRcode(req, dns.RcodeServerFailure)
				break
			}
		}
	}
	_ = w.WriteMsg(msg)
}

func (r *Registry) addDNSAnswer(q dns.Question, msg *dns.Msg, req *dns.Msg) error {
	switch q.Qtype {
	// Always return loopback for any A query
	case dns.TypeA:
		rr, err := dns.NewRR(fmt.Sprintf("%s 5 IN A 127.0.0.1", q.Name))
		if err != nil {
			return err
		}
		msg.Answer = append(msg.Answer, rr)
		return nil

	// TXT records are the only important record for ACME dns-01 challenges
	case dns.TypeTXT:
		records, err := r.Records(context.TODO())
		if err != nil {
			return err
		}

		found := false
		for _, record := range records {
			if record.DNSName == strings.TrimSuffix(q.Name, ".") && record.RecordType == endpoint.RecordTypeTXT {
				for _, target := range record.Targets {
					rr, err := dns.NewRR(fmt.Sprintf("%s 5 IN TXT %s", q.Name, target))
					if err != nil {
						return err
					}

					msg.Answer = append(msg.Answer, rr)
					found = true
				}
			}
		}

		if !found {
			msg.SetRcode(req, dns.RcodeNameError)
			return nil
		}

		return nil

	// NS and SOA are for authoritative lookups, return obviously invalid data
	case dns.TypeNS:
		rr, err := dns.NewRR(fmt.Sprintf("%s 5 IN NS ns.external-dns-acme-webook.invalid.", q.Name))
		if err != nil {
			return err
		}
		msg.Answer = append(msg.Answer, rr)
		return nil
	case dns.TypeSOA:
		rr, err := dns.NewRR(fmt.Sprintf("%s 5 IN SOA %s 20 5 5 5 5", "ns.external-dns-acme-webook.invalid.", "ns.external-dns-acme-webook.invalid."))
		if err != nil {
			return err
		}
		msg.Answer = append(msg.Answer, rr)
		return nil
	default:
		return fmt.Errorf("unimplemented record type %v", q.Qtype)
	}
}
