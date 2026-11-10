// SPDX-FileCopyrightText: 2026 Balázs Toldi
//
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/omec-project/openapi/v2/models"
	openapiutils "github.com/omec-project/openapi/v2/utils"
	udrContext "github.com/omec-project/udr/context"
	"github.com/omec-project/udr/factory"
	"github.com/omec-project/util/sbiaddr"
)

const (
	profileV4      = "10.0.0.4"
	profileV6      = "fd00::4"
	profileFQDN    = "udr"
	profileV6URI   = "http://[fd00::4]:29504"
	profileVersion = "1.0.0"
	profileNfId    = "test-udr"
)

type wantEndPoint struct {
	ipv4, ipv6 string
}

func TestGetNfProfileAddresses(t *testing.T) {
	cases := []struct {
		name         string
		registerIPv4 string
		registerIPv6 string
		preference   sbiaddr.Preference
		wantFqdn     string
		wantIPv4     []string
		wantIPv6     []string
		wantEPs      []wantEndPoint
		wantPrefix   string
	}{
		{
			name: "IPv4", registerIPv4: profileV4,
			wantIPv4: []string{profileV4}, wantEPs: []wantEndPoint{{ipv4: profileV4}},
			wantPrefix: "http://10.0.0.4:29504",
		},
		{
			name: "FQDN goes to fqdn, not to an IPv4 field", registerIPv4: profileFQDN,
			wantFqdn: profileFQDN, wantEPs: []wantEndPoint{{}},
			wantPrefix: "http://udr:29504",
		},
		{
			name: "IPv6", registerIPv6: profileV6,
			wantIPv6: []string{profileV6}, wantEPs: []wantEndPoint{{ipv6: profileV6}},
			wantPrefix: profileV6URI,
		},
		{
			name: "IPv6 in RegisterIPv4", registerIPv4: profileV6,
			wantIPv6: []string{profileV6}, wantEPs: []wantEndPoint{{ipv6: profileV6}},
			wantPrefix: profileV6URI,
		},
		{
			name: "dual stack: two endpoints, IPv4 first", registerIPv4: profileV4, registerIPv6: profileV6,
			wantIPv4: []string{profileV4}, wantIPv6: []string{profileV6},
			wantEPs:    []wantEndPoint{{ipv4: profileV4}, {ipv6: profileV6}},
			wantPrefix: "http://10.0.0.4:29504",
		},
		{
			name: "dual stack, IPv6 preferred", registerIPv4: profileV4, registerIPv6: profileV6,
			preference: sbiaddr.PreferIPv6,
			wantIPv4:   []string{profileV4}, wantIPv6: []string{profileV6},
			wantEPs:    []wantEndPoint{{ipv4: profileV4}, {ipv6: profileV6}},
			wantPrefix: profileV6URI,
		},
		{
			name: "FQDN and IPv6", registerIPv4: profileFQDN, registerIPv6: profileV6,
			wantFqdn: profileFQDN, wantIPv6: []string{profileV6}, wantEPs: []wantEndPoint{{ipv6: profileV6}},
			wantPrefix: "http://udr:29504",
		},
	}
	origConfig := factory.UdrConfig
	t.Cleanup(func() { factory.UdrConfig = origConfig })
	factory.UdrConfig = factory.Config{Info: &factory.Info{Version: profileVersion}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prof, err := getNfProfile(&udrContext.UDRContext{
				NfId:          profileNfId,
				UriScheme:     models.URISCHEME_HTTP,
				RegisterIPv4:  tc.registerIPv4,
				RegisterIPv6:  tc.registerIPv6,
				SBIPort:       29504,
				SbiPreference: tc.preference,
			}, nil)
			if err != nil {
				t.Fatalf("getNfProfile returned error: %v", err)
			}
			if got := prof.GetFqdn(); got != tc.wantFqdn {
				t.Errorf("profile fqdn = %q, want %q", got, tc.wantFqdn)
			}
			if got := prof.GetIpv4Addresses(); !reflect.DeepEqual(got, tc.wantIPv4) {
				t.Errorf("profile ipv4Addresses = %v, want %v", got, tc.wantIPv4)
			}
			if got := prof.GetIpv6Addresses(); !reflect.DeepEqual(got, tc.wantIPv6) {
				t.Errorf("profile ipv6Addresses = %v, want %v", got, tc.wantIPv6)
			}

			services := prof.GetNfServices()
			listed := prof.GetNfServiceList()
			if len(services) != 1 || len(listed) != 1 {
				t.Fatalf("got %d nfServices and %d nfServiceList entries, want 1 each", len(services), len(listed))
			}
			for _, svc := range []models.NFService{services[0], listed[services[0].GetServiceInstanceId()]} {
				checkService(t, svc, tc.wantFqdn, tc.wantPrefix, tc.wantEPs)
			}

			// A consumer using the shared resolver reaches the UDR at its
			// apiPrefix, whatever the family.
			discovered := profileForDiscovery(t, prof)
			if got := openapiutils.SearchNFServiceURI(&discovered, models.SERVICENAME_NUDR_DR,
				models.NFSERVICESTATUS_REGISTERED, tc.preference == sbiaddr.PreferIPv6); got != tc.wantPrefix {
				t.Errorf("SearchNFServiceURI = %q, want %q", got, tc.wantPrefix)
			}
		})
	}
}

func checkService(t *testing.T, svc models.NFService, wantFqdn, wantPrefix string, wantEPs []wantEndPoint) {
	t.Helper()
	if got := svc.GetFqdn(); got != wantFqdn {
		t.Errorf("service fqdn = %q, want %q", got, wantFqdn)
	}
	if got := svc.GetApiPrefix(); got != wantPrefix {
		t.Errorf("service apiPrefix = %q, want %q", got, wantPrefix)
	}
	endPoints := svc.GetIpEndPoints()
	if len(endPoints) != len(wantEPs) {
		t.Fatalf("got %d ipEndPoints, want %d: %+v", len(endPoints), len(wantEPs), endPoints)
	}
	for i, ep := range endPoints {
		if ep.GetIpv4Address() != wantEPs[i].ipv4 || ep.GetIpv6Address() != wantEPs[i].ipv6 {
			t.Errorf("ipEndPoints[%d] = v4 %q v6 %q, want v4 %q v6 %q",
				i, ep.GetIpv4Address(), ep.GetIpv6Address(), wantEPs[i].ipv4, wantEPs[i].ipv6)
		}
		if ep.GetPort() != 29504 || ep.GetTransport() != models.TRANSPORTPROTOCOL_TCP {
			t.Errorf("ipEndPoints[%d] = port %d transport %q, want 29504 TCP", i, ep.GetPort(), ep.GetTransport())
		}
	}
}

// profileForDiscovery round-trips the registered profile through JSON, as the
// NRF hands it to a consumer.
func profileForDiscovery(t *testing.T, prof models.NFProfile) models.NFProfileDiscovery {
	t.Helper()
	raw, err := json.Marshal(prof)
	if err != nil {
		t.Fatalf("marshal profile: %v", err)
	}
	var discovered models.NFProfileDiscovery
	if err := json.Unmarshal(raw, &discovered); err != nil {
		t.Fatalf("unmarshal profile: %v", err)
	}
	return discovered
}

func TestGetNfProfileRejectsAnEmptyAddress(t *testing.T) {
	origConfig := factory.UdrConfig
	t.Cleanup(func() { factory.UdrConfig = origConfig })
	factory.UdrConfig = factory.Config{Info: &factory.Info{Version: profileVersion}}

	for _, register := range []string{"", "not an address"} {
		_, err := getNfProfile(&udrContext.UDRContext{
			NfId: profileNfId, UriScheme: models.URISCHEME_HTTP, RegisterIPv4: register, SBIPort: 29504,
		}, nil)
		if err == nil {
			t.Errorf("getNfProfile with RegisterIPv4 %q: got no error", register)
		}
	}
}
