package rdapapi

import (
	"encoding/json"
	"testing"
)

func TestDomainResponseUnmarshal(t *testing.T) {
	raw := `{
		"domain": "example.com",
		"unicode_name": "example.com",
		"handle": "D12345-LRMS",
		"status": ["client transfer prohibited", "server delete prohibited"],
		"registrar": {
			"name": "Example Registrar",
			"iana_id": "9999",
			"abuse_email": "abuse@example.com",
			"abuse_phone": "+1.5551234567",
			"url": "https://example-registrar.com"
		},
		"dates": {
			"registered": "1995-08-14T04:00:00Z",
			"expires": "2025-08-13T04:00:00Z",
			"updated": "2024-08-14T07:01:44Z"
		},
		"nameservers": ["ns1.example.com", "ns2.example.com"],
		"dnssec": true,
		"entities": {
			"registrant": {
				"handle": "C-001",
				"name": "John Doe",
				"organization": "Example Inc.",
				"email": "john@example.com",
				"phone": "+1.5559876543",
				"address": "123 Main St",
				"contact_url": "https://example.com/contact",
				"country_code": "US"
			}
		},
		"meta": {
			"rdap_server": "https://rdap.example.com",
			"raw_rdap_url": "https://rdap.example.com/domain/example.com",
			"cached": false,
			"cache_expires": "2024-08-14T08:00:00Z",
			"followed": true,
			"registrar_rdap_server": "https://rdap.registrar.com"
		}
	}`

	var resp DomainResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if resp.Domain != "example.com" {
		t.Errorf("Domain = %q, want %q", resp.Domain, "example.com")
	}
	assertStringPtr(t, "UnicodeName", resp.UnicodeName, "example.com")
	assertStringPtr(t, "Handle", resp.Handle, "D12345-LRMS")
	if len(resp.Status) != 2 {
		t.Errorf("Status len = %d, want 2", len(resp.Status))
	}
	assertStringPtr(t, "Registrar.Name", resp.Registrar.Name, "Example Registrar")
	assertStringPtr(t, "Registrar.IANAID", resp.Registrar.IANAID, "9999")
	assertStringPtr(t, "Registrar.AbuseEmail", resp.Registrar.AbuseEmail, "abuse@example.com")
	assertStringPtr(t, "Registrar.AbusePhone", resp.Registrar.AbusePhone, "+1.5551234567")
	assertStringPtr(t, "Registrar.URL", resp.Registrar.URL, "https://example-registrar.com")
	assertStringPtr(t, "Dates.Registered", resp.Dates.Registered, "1995-08-14T04:00:00Z")
	assertStringPtr(t, "Dates.Expires", resp.Dates.Expires, "2025-08-13T04:00:00Z")
	assertStringPtr(t, "Dates.Updated", resp.Dates.Updated, "2024-08-14T07:01:44Z")
	if len(resp.Nameservers) != 2 {
		t.Errorf("Nameservers len = %d, want 2", len(resp.Nameservers))
	}
	if !resp.DNSSEC {
		t.Error("DNSSEC = false, want true")
	}
	if resp.Entities.Registrant == nil {
		t.Fatal("Entities.Registrant is nil")
	}
	assertStringPtr(t, "Registrant.Name", resp.Entities.Registrant.Name, "John Doe")
	assertStringPtr(t, "Registrant.Organization", resp.Entities.Registrant.Organization, "Example Inc.")
	assertStringPtr(t, "Registrant.CountryCode", resp.Entities.Registrant.CountryCode, "US")
	if resp.Meta.RDAPServer != "https://rdap.example.com" {
		t.Errorf("Meta.RDAPServer = %q, want %q", resp.Meta.RDAPServer, "https://rdap.example.com")
	}
	if resp.Meta.Cached {
		t.Error("Meta.Cached = true, want false")
	}
	if resp.Meta.Followed == nil || !*resp.Meta.Followed {
		t.Error("Meta.Followed should be true")
	}
	assertStringPtr(t, "Meta.RegistrarRDAPServer", resp.Meta.RegistrarRDAPServer, "https://rdap.registrar.com")
}

func TestDomainResponseNullableFields(t *testing.T) {
	raw := `{
		"domain": "example.com",
		"status": [],
		"registrar": {},
		"dates": {"registered": null, "expires": null, "updated": null},
		"nameservers": [],
		"dnssec": false,
		"entities": {},
		"meta": {"rdap_server": "", "raw_rdap_url": "", "cached": false, "cache_expires": ""}
	}`

	var resp DomainResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if resp.UnicodeName != nil {
		t.Error("UnicodeName should be nil")
	}
	if resp.Handle != nil {
		t.Error("Handle should be nil")
	}
	if resp.Dates.Registered != nil {
		t.Error("Dates.Registered should be nil")
	}
	if resp.Entities.Registrant != nil {
		t.Error("Entities.Registrant should be nil")
	}
	if resp.Meta.Followed != nil {
		t.Error("Meta.Followed should be nil")
	}
}

func TestIpResponseUnmarshal(t *testing.T) {
	raw := `{
		"handle": "NET-8-8-8-0-1",
		"name": "LVLT-GOGL-8-8-8",
		"type": "ALLOCATION",
		"start_address": "8.8.8.0",
		"end_address": "8.8.8.255",
		"ip_version": "v4",
		"parent_handle": "NET-8-0-0-0-1",
		"country": "US",
		"status": ["active"],
		"dates": {"registered": "2014-03-14T00:00:00Z", "expires": null, "updated": "2014-03-14T00:00:00Z"},
		"entities": {
			"abuse": {
				"handle": "ABUSE-001",
				"email": "abuse@google.com"
			}
		},
		"cidr": ["8.8.8.0/24"],
		"remarks": [{"title": "Note", "description": "For Google DNS"}],
		"port43": "whois.arin.net",
		"meta": {"rdap_server": "https://rdap.arin.net", "raw_rdap_url": "https://rdap.arin.net/ip/8.8.8.8", "cached": true, "cache_expires": "2024-08-14T08:00:00Z"}
	}`

	var resp IpResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	assertStringPtr(t, "Handle", resp.Handle, "NET-8-8-8-0-1")
	assertStringPtr(t, "Name", resp.Name, "LVLT-GOGL-8-8-8")
	assertStringPtr(t, "Type", resp.Type, "ALLOCATION")
	assertStringPtr(t, "StartAddress", resp.StartAddress, "8.8.8.0")
	assertStringPtr(t, "EndAddress", resp.EndAddress, "8.8.8.255")
	assertStringPtr(t, "IPVersion", resp.IPVersion, "v4")
	assertStringPtr(t, "ParentHandle", resp.ParentHandle, "NET-8-0-0-0-1")
	assertStringPtr(t, "Country", resp.Country, "US")
	if len(resp.CIDR) != 1 || resp.CIDR[0] != "8.8.8.0/24" {
		t.Errorf("CIDR = %v, want [8.8.8.0/24]", resp.CIDR)
	}
	if len(resp.Remarks) != 1 {
		t.Fatalf("Remarks len = %d, want 1", len(resp.Remarks))
	}
	assertStringPtr(t, "Remarks[0].Title", resp.Remarks[0].Title, "Note")
	if resp.Remarks[0].Description != "For Google DNS" {
		t.Errorf("Remarks[0].Description = %q, want %q", resp.Remarks[0].Description, "For Google DNS")
	}
	assertStringPtr(t, "Port43", resp.Port43, "whois.arin.net")
	if !resp.Meta.Cached {
		t.Error("Meta.Cached = false, want true")
	}
	if resp.Entities.Abuse == nil {
		t.Fatal("Entities.Abuse is nil")
	}
	assertStringPtr(t, "Entities.Abuse.Email", resp.Entities.Abuse.Email, "abuse@google.com")
}

func TestAsnResponseUnmarshal(t *testing.T) {
	raw := `{
		"handle": "AS15169",
		"name": "GOOGLE",
		"type": "DIRECT ALLOCATION",
		"start_autnum": 15169,
		"end_autnum": 15169,
		"status": ["active"],
		"dates": {"registered": "2000-03-10T00:00:00Z", "expires": null, "updated": "2012-02-24T00:00:00Z"},
		"entities": {},
		"remarks": [],
		"port43": "whois.arin.net",
		"meta": {"rdap_server": "https://rdap.arin.net", "raw_rdap_url": "https://rdap.arin.net/autnum/15169", "cached": false, "cache_expires": ""}
	}`

	var resp AsnResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	assertStringPtr(t, "Handle", resp.Handle, "AS15169")
	assertStringPtr(t, "Name", resp.Name, "GOOGLE")
	assertStringPtr(t, "Type", resp.Type, "DIRECT ALLOCATION")
	if resp.StartAutnum == nil || *resp.StartAutnum != 15169 {
		t.Errorf("StartAutnum = %v, want 15169", resp.StartAutnum)
	}
	if resp.EndAutnum == nil || *resp.EndAutnum != 15169 {
		t.Errorf("EndAutnum = %v, want 15169", resp.EndAutnum)
	}
	assertStringPtr(t, "Port43", resp.Port43, "whois.arin.net")
}

func TestNameserverResponseUnmarshal(t *testing.T) {
	raw := `{
		"ldh_name": "ns1.google.com",
		"unicode_name": "ns1.google.com",
		"handle": "NS-001",
		"ip_addresses": {"v4": ["216.239.32.10"], "v6": ["2001:4860:4802:32::a"]},
		"status": ["active"],
		"dates": {"registered": null, "expires": null, "updated": null},
		"entities": {},
		"meta": {"rdap_server": "https://rdap.verisign.com", "raw_rdap_url": "https://rdap.verisign.com/com/v1/nameserver/ns1.google.com", "cached": false, "cache_expires": ""}
	}`

	var resp NameserverResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if resp.LDHName != "ns1.google.com" {
		t.Errorf("LDHName = %q, want %q", resp.LDHName, "ns1.google.com")
	}
	assertStringPtr(t, "UnicodeName", resp.UnicodeName, "ns1.google.com")
	assertStringPtr(t, "Handle", resp.Handle, "NS-001")
	if len(resp.IPAddresses.V4) != 1 || resp.IPAddresses.V4[0] != "216.239.32.10" {
		t.Errorf("IPAddresses.V4 = %v, want [216.239.32.10]", resp.IPAddresses.V4)
	}
	if len(resp.IPAddresses.V6) != 1 || resp.IPAddresses.V6[0] != "2001:4860:4802:32::a" {
		t.Errorf("IPAddresses.V6 = %v, want [2001:4860:4802:32::a]", resp.IPAddresses.V6)
	}
}

func TestEntityResponseUnmarshal(t *testing.T) {
	raw := `{
		"handle": "GOGL",
		"name": "Google LLC",
		"organization": "Google LLC",
		"email": "arin-contact@google.com",
		"phone": "+1-650-253-0000",
		"address": "1600 Amphitheatre Parkway",
		"contact_url": "https://google.com",
		"country_code": "US",
		"roles": ["registrant"],
		"status": ["active"],
		"dates": {"registered": "2000-03-30T00:00:00Z", "expires": null, "updated": "2024-01-01T00:00:00Z"},
		"remarks": [{"title": "Registration", "description": "First registered in 2000"}],
		"port43": "whois.arin.net",
		"public_ids": [{"type": "ARIN OrgID", "identifier": "GOGL"}],
		"entities": {},
		"autnums": [{"handle": "AS15169", "name": "GOOGLE", "start_autnum": 15169, "end_autnum": 15169}],
		"networks": [{"handle": "NET-8-8-8-0-1", "name": "LVLT-GOGL-8-8-8", "start_address": "8.8.8.0", "end_address": "8.8.8.255", "ip_version": "v4", "cidr": ["8.8.8.0/24"]}],
		"meta": {"rdap_server": "https://rdap.arin.net", "raw_rdap_url": "https://rdap.arin.net/entity/GOGL", "cached": false, "cache_expires": ""}
	}`

	var resp EntityResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	assertStringPtr(t, "Handle", resp.Handle, "GOGL")
	assertStringPtr(t, "Name", resp.Name, "Google LLC")
	assertStringPtr(t, "Organization", resp.Organization, "Google LLC")
	assertStringPtr(t, "Email", resp.Email, "arin-contact@google.com")
	assertStringPtr(t, "Phone", resp.Phone, "+1-650-253-0000")
	assertStringPtr(t, "Address", resp.Address, "1600 Amphitheatre Parkway")
	assertStringPtr(t, "ContactURL", resp.ContactURL, "https://google.com")
	assertStringPtr(t, "CountryCode", resp.CountryCode, "US")
	if len(resp.Roles) != 1 || resp.Roles[0] != "registrant" {
		t.Errorf("Roles = %v, want [registrant]", resp.Roles)
	}
	if len(resp.PublicIDs) != 1 {
		t.Fatalf("PublicIDs len = %d, want 1", len(resp.PublicIDs))
	}
	assertStringPtr(t, "PublicIDs[0].Type", resp.PublicIDs[0].Type, "ARIN OrgID")
	assertStringPtr(t, "PublicIDs[0].Identifier", resp.PublicIDs[0].Identifier, "GOGL")
	if len(resp.Autnums) != 1 {
		t.Fatalf("Autnums len = %d, want 1", len(resp.Autnums))
	}
	assertStringPtr(t, "Autnums[0].Handle", resp.Autnums[0].Handle, "AS15169")
	if resp.Autnums[0].StartAutnum == nil || *resp.Autnums[0].StartAutnum != 15169 {
		t.Errorf("Autnums[0].StartAutnum = %v, want 15169", resp.Autnums[0].StartAutnum)
	}
	if len(resp.Networks) != 1 {
		t.Fatalf("Networks len = %d, want 1", len(resp.Networks))
	}
	assertStringPtr(t, "Networks[0].Handle", resp.Networks[0].Handle, "NET-8-8-8-0-1")
	assertStringPtr(t, "Networks[0].StartAddress", resp.Networks[0].StartAddress, "8.8.8.0")
	assertStringPtr(t, "Networks[0].IPVersion", resp.Networks[0].IPVersion, "v4")
	if len(resp.Networks[0].CIDR) != 1 || resp.Networks[0].CIDR[0] != "8.8.8.0/24" {
		t.Errorf("Networks[0].CIDR = %v, want [8.8.8.0/24]", resp.Networks[0].CIDR)
	}
}

func TestBulkDomainResponseUnmarshal(t *testing.T) {
	raw := `{
		"results": [
			{
				"domain": "example.com",
				"status": "success",
				"data": {
					"domain": "example.com",
					"status": ["active"],
					"registrar": {"name": "Test Registrar"},
					"dates": {},
					"nameservers": [],
					"dnssec": false,
					"entities": {},
					"meta": {"rdap_server": "", "raw_rdap_url": "", "cached": false, "cache_expires": ""}
				}
			},
			{
				"domain": "nope.example",
				"status": "error",
				"error": "not_found",
				"message": "No RDAP data found"
			}
		],
		"summary": {"total": 2, "successful": 1, "failed": 1}
	}`

	var resp BulkDomainResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(resp.Results) != 2 {
		t.Fatalf("Results len = %d, want 2", len(resp.Results))
	}
	if resp.Results[0].Domain != "example.com" {
		t.Errorf("Results[0].Domain = %q, want %q", resp.Results[0].Domain, "example.com")
	}
	if resp.Results[0].Status != "success" {
		t.Errorf("Results[0].Status = %q, want %q", resp.Results[0].Status, "success")
	}
	if resp.Results[0].Data == nil {
		t.Fatal("Results[0].Data is nil")
	}
	if resp.Results[1].Status != "error" {
		t.Errorf("Results[1].Status = %q, want %q", resp.Results[1].Status, "error")
	}
	assertStringPtr(t, "Results[1].Error", resp.Results[1].Error, "not_found")
	assertStringPtr(t, "Results[1].Message", resp.Results[1].Message, "No RDAP data found")
	if resp.Summary.Total != 2 {
		t.Errorf("Summary.Total = %d, want 2", resp.Summary.Total)
	}
	if resp.Summary.Successful != 1 {
		t.Errorf("Summary.Successful = %d, want 1", resp.Summary.Successful)
	}
	if resp.Summary.Failed != 1 {
		t.Errorf("Summary.Failed = %d, want 1", resp.Summary.Failed)
	}
}

func TestMarshalRoundTrip(t *testing.T) {
	s := func(v string) *string { return &v }
	b := func(v bool) *bool { return &v }

	original := &DomainResponse{
		Domain:      "test.com",
		UnicodeName: s("test.com"),
		Handle:      s("D-123"),
		Status:      []string{"active"},
		Registrar:   Registrar{Name: s("Test Reg")},
		Dates:       Dates{Registered: s("2020-01-01T00:00:00Z")},
		Nameservers: []string{"ns1.test.com"},
		DNSSEC:      true,
		Entities:    Entities{},
		Meta: Meta{
			RDAPServer:   "https://rdap.test.com",
			RawRDAPURL:   "https://rdap.test.com/domain/test.com",
			Cached:       true,
			CacheExpires: "2024-01-01T00:00:00Z",
			Followed:     b(false),
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded DomainResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Domain != original.Domain {
		t.Errorf("Domain = %q, want %q", decoded.Domain, original.Domain)
	}
	if decoded.DNSSEC != original.DNSSEC {
		t.Errorf("DNSSEC = %v, want %v", decoded.DNSSEC, original.DNSSEC)
	}
	if decoded.Meta.Cached != original.Meta.Cached {
		t.Errorf("Meta.Cached = %v, want %v", decoded.Meta.Cached, original.Meta.Cached)
	}
	if decoded.Meta.Followed == nil || *decoded.Meta.Followed != false {
		t.Error("Meta.Followed should be false after round-trip")
	}
}

// assertStringPtr is a helper for checking *string values.
func assertStringPtr(t *testing.T, field string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Errorf("%s is nil, want %q", field, want)
		return
	}
	if *got != want {
		t.Errorf("%s = %q, want %q", field, *got, want)
	}
}
