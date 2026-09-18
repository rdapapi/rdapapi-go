package rdapapi

import (
	"math"
	"time"
)

// Protocol is the wire protocol an answer came from. Unknown values from the
// API pass through unchanged rather than failing to decode.
type Protocol string

const (
	ProtocolRDAP  Protocol = "rdap"
	ProtocolWHOIS Protocol = "whois"
)

// Meta contains metadata about the lookup: where the answer came from, and how
// it was served.
type Meta struct {
	// Server is the hostname of the upstream that answered. It matches the
	// Server of that TLD's /tlds entry, so it can be passed straight to
	// WithServer. Occasionally nil on an older cached record.
	Server *string `json:"server"`
	// Source is which protocol answered: ProtocolRDAP everywhere except a
	// domain lookup whose TLD has no RDAP server, which is read over WHOIS.
	Source Protocol `json:"source"`
	// RDAPServer is the base URL of the RDAP server, empty when Source is
	// ProtocolWHOIS.
	//
	// Deprecated: use Server for the host that answered, or RawRDAPURL for the
	// exact endpoint queried.
	RDAPServer string `json:"rdap_server"`
	// RawRDAPURL is the direct URL to the raw RDAP response. Empty when Source
	// is ProtocolWHOIS, which has no URL form, or when a stored snapshot
	// answered.
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
//
// Every field is nullable: the registry may never have had the value, or may
// have withheld it. Redacted says which, where the server declares it.
type Contact struct {
	Handle       *string `json:"handle"`
	Name         *string `json:"name"`
	Organization *string `json:"organization"`
	Email        *string `json:"email"`
	// Phone is E.164, with a ";ext=42" suffix where the registry supplies an
	// extension.
	Phone   *string `json:"phone"`
	Address *string `json:"address"`
	// ContactURL carries the web form a server publishes where it withholds
	// Email.
	ContactURL  *string `json:"contact_url"`
	CountryCode *string `json:"country_code"`
}

// Entities contains contact entities keyed by role.
type Entities struct {
	Registrant     *Contact `json:"registrant,omitempty"`
	Administrative *Contact `json:"administrative,omitempty"`
	Technical      *Contact `json:"technical,omitempty"`
	Billing        *Contact `json:"billing,omitempty"`
	Abuse          *Contact `json:"abuse,omitempty"`
}

// RedactionMethod is how an upstream server said it withheld a value:
// RedactionRemoval deleted it, RedactionEmptyValue blanked it,
// RedactionPartialValue truncated it, and RedactionReplacementValue published a
// substitute in its place. A method we do not recognise passes through
// unchanged.
type RedactionMethod string

const (
	RedactionRemoval          RedactionMethod = "removal"
	RedactionEmptyValue       RedactionMethod = "emptyValue"
	RedactionPartialValue     RedactionMethod = "partialValue"
	RedactionReplacementValue RedactionMethod = "replacementValue"
)

// Redaction reports which fields the upstream server declared it withheld, and
// by what method. It mirrors the shape of the record it describes, so a claim
// about Entities.Registrant.Name sits at Redacted.Entities["registrant"]["name"].
//
// Nil when the server declared nothing, which is not evidence that nothing was
// withheld: most servers declare nothing at all.
type Redaction struct {
	Handle *RedactionMethod `json:"handle,omitempty"`
	// Registrar holds claims about the top-level registrar object, keyed by
	// field name. Domain lookups only.
	Registrar map[string]RedactionMethod `json:"registrar,omitempty"`
	// Entities holds claims keyed by contact role, then by field within that
	// contact.
	Entities map[string]map[string]RedactionMethod `json:"entities,omitempty"`
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
	// DNSSEC reports whether the delegation is signed. Nil where the registry
	// publishes no DNSSEC status, as .tr, .gg and .nc do not.
	DNSSEC   *bool      `json:"dnssec"`
	Entities Entities   `json:"entities"`
	Redacted *Redaction `json:"redacted,omitempty"`
	Meta     Meta       `json:"meta"`
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
	// Geofeed is the URL of the RFC 8805 geofeed this network publishes,
	// returned as published, never fetched, and never inherited from a parent
	// network. Nil when there is none.
	Geofeed  *string    `json:"geofeed"`
	Remarks  []Remark   `json:"remarks"`
	Port43   *string    `json:"port43"`
	Redacted *Redaction `json:"redacted,omitempty"`
	Meta     Meta       `json:"meta"`
}

// AsnResponse is the response from an ASN lookup.
type AsnResponse struct {
	Handle      *string `json:"handle"`
	Name        *string `json:"name"`
	Type        *string `json:"type"`
	StartAutnum *int    `json:"start_autnum"`
	EndAutnum   *int    `json:"end_autnum"`
	// Country is the ISO 3166-1 alpha-2 code derived from the contact
	// entities' address: regional registries carry no top-level country on
	// autnum records. Nil when no contact supplies one.
	Country  *string    `json:"country"`
	Status   []string   `json:"status"`
	Dates    Dates      `json:"dates"`
	Entities Entities   `json:"entities"`
	Remarks  []Remark   `json:"remarks"`
	Port43   *string    `json:"port43"`
	Redacted *Redaction `json:"redacted,omitempty"`
	Meta     Meta       `json:"meta"`
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
	Redacted    *Redaction  `json:"redacted,omitempty"`
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
	Redacted     *Redaction      `json:"redacted,omitempty"`
	Meta         Meta            `json:"meta"`
}

// BulkDomainResult is a single result within a bulk domain lookup response.
type BulkDomainResult struct {
	Domain string `json:"domain"`
	// Status is "success" or "error". A failing domain does not fail the
	// request, so check it on every entry.
	Status  string          `json:"status"`
	Data    *DomainResponse `json:"data,omitempty"`
	Error   *string         `json:"error,omitempty"`
	Message *string         `json:"message,omitempty"`
	// RawMeta is the entry-level meta. On a successful entry it is merged into
	// Data.Meta and left nil. On a failed one it holds the partial meta —
	// Server and Source alone — naming the upstream that was tried, and is nil
	// when the entry failed before an upstream was chosen, as invalid_domain
	// does.
	RawMeta *Meta `json:"meta,omitempty"`
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

// AvailabilityLevel is a qualitative bucket describing how often a field is
// populated in a TLD's RDAP responses.
type AvailabilityLevel string

const (
	AvailabilityAlways    AvailabilityLevel = "always"
	AvailabilityUsually   AvailabilityLevel = "usually"
	AvailabilitySometimes AvailabilityLevel = "sometimes"
	AvailabilityNever     AvailabilityLevel = "never"
)

// FieldAvailability reports how often each common domain field is populated in
// a TLD's RDAP responses.
type FieldAvailability struct {
	Registrar    AvailabilityLevel `json:"registrar"`
	RegisteredAt AvailabilityLevel `json:"registered_at"`
	ExpiresAt    AvailabilityLevel `json:"expires_at"`
	Nameservers  AvailabilityLevel `json:"nameservers"`
	Status       AvailabilityLevel `json:"status"`
}

// TldEntry is a single TLD entry from the /tlds catalog.
type TldEntry struct {
	TLD string `json:"tld"`
	// Protocol is which protocol answers for this TLD. ProtocolWHOIS marks the
	// ccTLDs IANA lists no RDAP server for.
	Protocol       Protocol `json:"protocol"`
	SupportedSince string   `json:"supported_since"`
	// Server is the hostname of the upstream that answers for this TLD. It is
	// what WithServer filters on, and what a lookup's Meta.Server returns.
	Server string `json:"server"`
	// RDAPServerHost is nil when Protocol is ProtocolWHOIS.
	//
	// Deprecated: superseded by Server.
	RDAPServerHost *string `json:"rdap_server_host"`
	// RDAPServerURL is the full URL of the upstream RDAP server. Nil when
	// Protocol is ProtocolWHOIS, which has no URL form.
	RDAPServerURL *string `json:"rdap_server_url"`
	// FieldAvailability is nil when we do not yet have enough observations for
	// this TLD, or when Protocol is ProtocolWHOIS: the measurement is taken
	// from RDAP responses.
	FieldAvailability *FieldAvailability `json:"field_availability"`
}

// TldThresholds lists the percentage cutoffs used to pick each availability
// label.
type TldThresholds struct {
	Always    float64 `json:"always"`
	Usually   float64 `json:"usually"`
	Sometimes float64 `json:"sometimes"`
}

// TldListMeta is the metadata envelope for GET /tlds.
type TldListMeta struct {
	ComputedAt string        `json:"computed_at"`
	Count      int           `json:"count"`
	Coverage   float64       `json:"coverage"`
	Thresholds TldThresholds `json:"thresholds"`
}

// TldMeta is the metadata envelope for GET /tlds/{tld}.
type TldMeta struct {
	ComputedAt string        `json:"computed_at"`
	Thresholds TldThresholds `json:"thresholds"`
}

// TldListResponse is the response from GET /tlds.
type TldListResponse struct {
	Data []TldEntry  `json:"data"`
	Meta TldListMeta `json:"meta"`
	// ETag is the value of the server's ETag header. Pass back via
	// WithIfNoneMatch to skip unchanged transfers on a later call.
	ETag string `json:"-"`
}

// TldResponse is the response from GET /tlds/{tld}.
type TldResponse struct {
	Data TldEntry `json:"data"`
	Meta TldMeta  `json:"meta"`
	// ETag is the value of the server's ETag header. Pass back via
	// WithIfNoneMatch to skip unchanged transfers on a later call.
	ETag string `json:"-"`
}

// PingResponse is the response from a health check.
type PingResponse struct {
	Status string `json:"status"`
}
