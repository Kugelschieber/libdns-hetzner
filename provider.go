package hetzner

import (
	"context"
	"strings"
	"sync"

	"github.com/libdns/libdns"
)

// Provider implements the libdns interfaces for Hetzner.
type Provider struct {
	// AuthAPIToken is the Hetzner Auth API token - see https://dns.hetzner.com/api-docs#section/Authentication/Auth-API-Token
	AuthAPIToken string `json:"auth_api_token"`

	client *Client
	once   sync.Once
}

// New returns a new libdns provider for Hetzner.
func New(token string) *Provider {
	return &Provider{
		AuthAPIToken: token,
		client:       NewClient(token),
	}
}

// GetRecords  implements the libdns.RecordGetter interface.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	records, err := p.client.GetAllRecords(ctx, unFQDN(zone))

	if err != nil {
		return nil, err
	}

	return []libdns.Record(records), nil
}

// AppendRecords implements the libdns.RecordAppender interface.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	var appendedRecords []libdns.Record

	for _, r := range records {
		newRecord, err := p.client.CreateRecord(ctx, unFQDN(zone), r)

		if err != nil {
			return nil, err
		}

		appendedRecords = append(appendedRecords, newRecord)
	}

	return appendedRecords, nil
}

// DeleteRecords implements the libdns.RecordDeleter interface.
func (p *Provider) DeleteRecords(ctx context.Context, _ string, records []libdns.Record) ([]libdns.Record, error) {
	for _, r := range records {
		err := p.client.DeleteRecord(ctx, r)

		if err != nil {
			return nil, err
		}
	}

	return records, nil
}

// SetRecords implements the libdns.RecordSetter interface.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	var setRecords []libdns.Record

	for _, r := range records {
		setRecord, err := p.client.CreateOrUpdateRecord(ctx, unFQDN(zone), r)

		if err != nil {
			return setRecords, err
		}

		setRecords = append(setRecords, setRecord)
	}

	return setRecords, nil
}

// getClient initializes the client for the provider.
func (p *Provider) getClient() *Client {
	p.once.Do(func() {
		if p.AuthAPIToken == "" {
			panic("hetzner: api token missing")
		}

		p.client = NewClient(p.AuthAPIToken)
	})

	return p.client
}

// unFQDN trims any trailing "." from fqdn. Hetzner's API does not use FQDNs.
func unFQDN(fqdn string) string {
	return strings.TrimSuffix(fqdn, ".")
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
