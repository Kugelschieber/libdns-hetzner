package hetzner

import (
	"github.com/libdns/libdns"
	"time"
)

// Zone is the zone type for Hetzner.
type Zone struct {
	ID string `json:"id"`
}

// Record is the record type for Hetzner implementing the libdns.Record interface.
type Record struct {
	ID     string `json:"id,omitempty"`
	ZoneID string `json:"zone_id,omitempty"`
	Type   string `json:"type"`
	Name   string `json:"name"`
	Data   string `json:"data"`
	TTL    int    `json:"ttl"`
}

// RR implements the libdns.Record interface.
func (r *Record) RR() libdns.RR {
	return libdns.RR{
		Name: r.Name,
		TTL:  time.Duration(r.TTL) * time.Second,
		Type: r.Type,
		Data: r.Data,
	}
}
