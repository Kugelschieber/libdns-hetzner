package hetzner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Client is a Hetzner DNS API client.
type Client struct {
	accessToken string
}

// NewClient returns a new Client.
func NewClient(accessToken string) *Client {
	return &Client{
		accessToken: accessToken,
	}
}

func (c *Client) GetZoneID(ctx context.Context, zone string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://dns.hetzner.com/api/v1/zones?name=%s", url.QueryEscape(zone)), nil)
	data, err := c.doRequest(req)

	if err != nil {
		return "", err
	}

	result := struct {
		Zones []Zone `json:"zones"`
	}{}

	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}

	if len(result.Zones) > 1 {
		return "", errors.New("zone is ambiguous")
	}

	return result.Zones[0].ID, nil
}

func (c *Client) GetAllRecords(ctx context.Context, zone string) ([]Record, error) {
	zoneID, err := c.GetZoneID(ctx, zone)

	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://dns.hetzner.com/api/v1/records?zone_id=%s", zoneID), nil)
	data, err := c.doRequest(req)

	if err != nil {
		return nil, err
	}

	result := struct {
		Records []Record `json:"records"`
	}{}

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	records := make([]Record, 0, len(result.Records))

	for _, r := range result.Records {
		records = append(records, Record{
			ID:   r.ID,
			Type: r.Type,
			Name: r.Name,
			Data: r.Data,
			TTL:  r.TTL,
		})
	}

	return records, nil
}

func (c *Client) CreateRecord(ctx context.Context, zone string, r Record) (Record, error) {
	zoneID, err := c.GetZoneID(ctx, zone)

	if err != nil {
		return Record{}, err
	}

	reqData := Record{
		ZoneID: zoneID,
		Type:   r.Type,
		Name:   c.normalizeRecordName(r.Name, zone),
		Data:   r.Data,
		TTL:    r.TTL,
	}

	reqBuffer, err := json.Marshal(reqData)

	if err != nil {
		return Record{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://dns.hetzner.com/api/v1/records", bytes.NewBuffer(reqBuffer))
	data, err := c.doRequest(req)

	if err != nil {
		return Record{}, err
	}

	result := struct {
		Record Record `json:"record"`
	}{}

	if err := json.Unmarshal(data, &result); err != nil {
		return Record{}, err
	}

	return Record{
		ID:   result.Record.ID,
		Type: result.Record.Type,
		Name: result.Record.Name,
		Data: result.Record.Data,
		TTL:  result.Record.TTL,
	}, nil
}

func (c *Client) DeleteRecord(ctx context.Context, record Record) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", fmt.Sprintf("https://dns.hetzner.com/api/v1/records/%s", record.ID), nil)

	if err != nil {
		return err
	}

	if _, err = c.doRequest(req); err != nil {
		return err
	}

	return nil
}

func (c *Client) UpdateRecord(ctx context.Context, zone string, r Record) (Record, error) {
	zoneID, err := c.GetZoneID(ctx, zone)

	if err != nil {
		return Record{}, err
	}

	reqData := Record{
		ZoneID: zoneID,
		Type:   r.Type,
		Name:   c.normalizeRecordName(r.Name, zone),
		Data:   r.Data,
		TTL:    r.TTL,
	}

	reqBuffer, err := json.Marshal(reqData)

	if err != nil {
		return Record{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", fmt.Sprintf("https://dns.hetzner.com/api/v1/records/%s", r.ID), bytes.NewBuffer(reqBuffer))

	if err != nil {
		return Record{}, err
	}

	data, err := c.doRequest(req)

	if err != nil {
		return Record{}, err
	}

	result := struct {
		Record Record `json:"record"`
	}{}

	if err := json.Unmarshal(data, &result); err != nil {
		return Record{}, err
	}

	return Record{
		ID:   result.Record.ID,
		Type: result.Record.Type,
		Name: result.Record.Name,
		Data: result.Record.Data,
		TTL:  result.Record.TTL,
	}, nil
}

func (c *Client) CreateOrUpdateRecord(ctx context.Context, zone string, r Record) (Record, error) {
	if len(r.ID) == 0 {
		return c.CreateRecord(ctx, zone, r)
	}

	return c.UpdateRecord(ctx, zone, r)
}

func (c *Client) doRequest(request *http.Request) ([]byte, error) {
	request.Header.Add("Auth-API-Token", c.accessToken)
	client := &http.Client{}
	response, err := client.Do(request)

	if err != nil {
		return nil, err
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("%s (%d)", http.StatusText(response.StatusCode), response.StatusCode)
	}

	defer func() {
		_ = response.Body.Close()
	}()
	data, err := io.ReadAll(response.Body)

	if err != nil {
		return nil, err
	}

	return data, nil
}

func (c *Client) normalizeRecordName(recordName string, zone string) string {
	// TODO
	// Workaround for https://github.com/caddy-dns/hetzner/issues/3
	// Can be removed after https://github.com/libdns/libdns/issues/12
	normalized := unFQDN(recordName)
	normalized = strings.TrimSuffix(normalized, unFQDN(zone))

	if normalized == "" {
		normalized = "@"
	}

	return unFQDN(normalized)
}
