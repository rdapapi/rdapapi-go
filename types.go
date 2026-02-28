package rdapapi

import (
	"math"
	"time"
)

// Meta contains metadata about the RDAP lookup.
type Meta struct {
	RDAPServer          string  `json:"rdap_server"`
	RawRDAPURL          string  `json:"raw_rdap_url"`
	Cached              bool    `json:"cached"`
	CacheExpires        string  `json:"cache_expires"`
	Followed            *bool   `json:"followed,omitempty"`
	RegistrarRDAPServer *string `json:"registrar_rdap_server,omitempty"`
	FollowError         *string `json:"follow_error,omitempty"`
}

// Dates contains registration dates.
type Dates struct {
	Registered *string `json:"registered"`
	Expires    *string `json:"expires"`
	Updated    *string `json:"updated"`
}

// RegisteredAt parses Registered into a time.Time.
// Returns the zero value and false if the field is nil or unparseable.
func (d Dates) RegisteredAt() (time.Time, bool) {
	return parseISO(d.Registered)
}

// ExpiresAt parses Expires into a time.Time.
// Returns the zero value and false if the field is nil or unparseable.
func (d Dates) ExpiresAt() (time.Time, bool) {
	return parseISO(d.Expires)
}

// UpdatedAt parses Updated into a time.Time.
// Returns the zero value and false if the field is nil or unparseable.
func (d Dates) UpdatedAt() (time.Time, bool) {
	return parseISO(d.Updated)
}

// ExpiresInDays returns the number of days until expiration.
// Returns -1 and false if the expiry date is nil or unparseable.
func (d Dates) ExpiresInDays() (int, bool) {
	t, ok := d.ExpiresAt()
	if !ok {
		return -1, false
	}
	days := int(math.Floor(time.Until(t).Hours() / 24))
	return days, true
}

func parseISO(s *string) (time.Time, bool) {
	if s == nil {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// Registrar contains domain registrar information.
type Registrar struct {
	Name       *string `json:"name"`
	IANAID     *string `json:"iana_id"`
	AbuseEmail *string `json:"abuse_email"`
	AbusePhone *string `json:"abuse_phone"`
	URL        *string `json:"url"`
}

// Contact contains contact entity information.
type Contact struct {
	Handle       *string `json:"handle"`
	Name         *string `json:"name"`
	Organization *string `json:"organization"`
	Email        *string `json:"email"`
	Phone        *string `json:"phone"`
	Address      *string `json:"address"`
	ContactURL   *string `json:"contact_url"`
	CountryCode  *string `json:"country_code"`
}

// Entities contains contact entities keyed by role.
type Entities struct {
	Registrant     *Contact `json:"registrant,omitempty"`
	Administrative *Contact `json:"administrative,omitempty"`
	Technical      *Contact `json:"technical,omitempty"`
	Billing        *Contact `json:"billing,omitempty"`
	Abuse          *Contact `json:"abuse,omitempty"`
}

// Remark contains a remark from the registry.
type Remark struct {
	Title       *string `json:"title"`
	Description string  `json:"description"`
}

// IpAddresses contains IP addresses for a nameserver.
type IpAddresses struct {
	V4 []string `json:"v4"`
	V6 []string `json:"v6"`
}

// PublicID contains a public identifier (e.g. ARIN OrgID, IANA Registrar ID).
type PublicID struct {
	Type       *string `json:"type"`
	Identifier *string `json:"identifier"`
}

// EntityAutnum contains an autonomous system number owned by an entity.
type EntityAutnum struct {
	Handle      *string `json:"handle"`
	Name        *string `json:"name"`
	StartAutnum *int    `json:"start_autnum"`
	EndAutnum   *int    `json:"end_autnum"`
}

// EntityNetwork contains an IP network block owned by an entity.
type EntityNetwork struct {
	Handle       *string  `json:"handle"`
	Name         *string  `json:"name"`
	StartAddress *string  `json:"start_address"`
	EndAddress   *string  `json:"end_address"`
	IPVersion    *string  `json:"ip_version"`
	CIDR         []string `json:"cidr"`
}

// DomainResponse is the response from a domain lookup.
type DomainResponse struct {
	Domain      string    `json:"domain"`
	UnicodeName *string   `json:"unicode_name"`
	Handle      *string   `json:"handle"`
	Status      []string  `json:"status"`
	Registrar   Registrar `json:"registrar"`
	Dates       Dates     `json:"dates"`
	Nameservers []string  `json:"nameservers"`
	DNSSEC      bool      `json:"dnssec"`
	Entities    Entities  `json:"entities"`
	Meta        Meta      `json:"meta"`
}

// IpResponse is the response from an IP address lookup.
type IpResponse struct {
	Handle       *string  `json:"handle"`
	Name         *string  `json:"name"`
	Type         *string  `json:"type"`
	StartAddress *string  `json:"start_address"`
	EndAddress   *string  `json:"end_address"`
	IPVersion    *string  `json:"ip_version"`
	ParentHandle *string  `json:"parent_handle"`
	Country      *string  `json:"country"`
	Status       []string `json:"status"`
	Dates        Dates    `json:"dates"`
	Entities     Entities `json:"entities"`
	CIDR         []string `json:"cidr"`
	Remarks      []Remark `json:"remarks"`
	Port43       *string  `json:"port43"`
	Meta         Meta     `json:"meta"`
}

// AsnResponse is the response from an ASN lookup.
type AsnResponse struct {
	Handle      *string  `json:"handle"`
	Name        *string  `json:"name"`
	Type        *string  `json:"type"`
	StartAutnum *int     `json:"start_autnum"`
	EndAutnum   *int     `json:"end_autnum"`
	Status      []string `json:"status"`
	Dates       Dates    `json:"dates"`
	Entities    Entities `json:"entities"`
	Remarks     []Remark `json:"remarks"`
	Port43      *string  `json:"port43"`
	Meta        Meta     `json:"meta"`
}

// NameserverResponse is the response from a nameserver lookup.
type NameserverResponse struct {
	LDHName     string      `json:"ldh_name"`
	UnicodeName *string     `json:"unicode_name"`
	Handle      *string     `json:"handle"`
	IPAddresses IpAddresses `json:"ip_addresses"`
	Status      []string    `json:"status"`
	Dates       Dates       `json:"dates"`
	Entities    Entities    `json:"entities"`
	Meta        Meta        `json:"meta"`
}

// EntityResponse is the response from an entity lookup.
type EntityResponse struct {
	Handle       *string         `json:"handle"`
	Name         *string         `json:"name"`
	Organization *string         `json:"organization"`
	Email        *string         `json:"email"`
	Phone        *string         `json:"phone"`
	Address      *string         `json:"address"`
	ContactURL   *string         `json:"contact_url"`
	CountryCode  *string         `json:"country_code"`
	Roles        []string        `json:"roles"`
	Status       []string        `json:"status"`
	Dates        Dates           `json:"dates"`
	Remarks      []Remark        `json:"remarks"`
	Port43       *string         `json:"port43"`
	PublicIDs    []PublicID      `json:"public_ids"`
	Entities     Entities        `json:"entities"`
	Autnums      []EntityAutnum  `json:"autnums"`
	Networks     []EntityNetwork `json:"networks"`
	Meta         Meta            `json:"meta"`
}

// BulkDomainResult is a single result within a bulk domain lookup response.
type BulkDomainResult struct {
	Domain  string          `json:"domain"`
	Status  string          `json:"status"`
	Data    *DomainResponse `json:"data,omitempty"`
	Error   *string         `json:"error,omitempty"`
	Message *string         `json:"message,omitempty"`
	RawMeta *Meta           `json:"meta,omitempty"`
}

// BulkDomainSummary contains summary counts for a bulk domain lookup.
type BulkDomainSummary struct {
	Total      int `json:"total"`
	Successful int `json:"successful"`
	Failed     int `json:"failed"`
}

// BulkDomainResponse is the response from a bulk domain lookup.
type BulkDomainResponse struct {
	Results []BulkDomainResult `json:"results"`
	Summary BulkDomainSummary  `json:"summary"`
}
