// SPDX-FileCopyrightText: 2026 Balázs Toldi
//
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/omec-project/openapi/v2/models"
	"github.com/omec-project/util/sbiaddr"
)

const (
	testV4    = "10.0.0.4"
	testV6    = "fd00::4"
	testFQDN  = "udr"
	testV6URI = "http://[fd00::4]:29504"
)

func TestGetSbiUri(t *testing.T) {
	cases := []struct {
		name         string
		registerIPv4 string
		registerIPv6 string
		preference   sbiaddr.Preference
		wantURI      string
		wantHost     string
	}{
		{"IPv4", testV4, "", sbiaddr.PreferIPv4, "http://10.0.0.4:29504", testV4},
		{"FQDN", testFQDN, "", sbiaddr.PreferIPv4, "http://udr:29504", testFQDN},
		{"IPv6 only", "", testV6, sbiaddr.PreferIPv4, testV6URI, testV6},
		{"IPv6 in RegisterIPv4", testV6, "", sbiaddr.PreferIPv4, testV6URI, testV6},
		{"dual, IPv4 preferred", testV4, testV6, sbiaddr.PreferIPv4, "http://10.0.0.4:29504", testV4},
		{"dual, IPv6 preferred", testV4, testV6, sbiaddr.PreferIPv6, testV6URI, testV6},
		{"FQDN and IPv6, IPv6 preferred", testFQDN, testV6, sbiaddr.PreferIPv6, "http://udr:29504", testFQDN},
		{"nothing", "", "", sbiaddr.PreferIPv4, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &UDRContext{
				UriScheme:     models.URISCHEME_HTTP,
				RegisterIPv4:  tc.registerIPv4,
				RegisterIPv6:  tc.registerIPv6,
				SBIPort:       29504,
				SbiPreference: tc.preference,
			}
			if got := ctx.GetSbiUri(); got != tc.wantURI {
				t.Fatalf("GetSbiUri() = %q, want %q", got, tc.wantURI)
			}
			if tc.wantURI == "" {
				return
			}
			groupURI := ctx.GetIPv4GroupUri(NUDR_DR)
			if groupURI != tc.wantURI+"/nudr-dr/v2" {
				t.Fatalf("GetIPv4GroupUri(NUDR_DR) = %q, want %q", groupURI, tc.wantURI+"/nudr-dr/v2")
			}
			// The Location headers are built from this URI, so it must parse
			// back into the registered host and port.
			u, err := url.Parse(groupURI + "/subscription-data/subs-to-notify/1")
			if err != nil {
				t.Fatalf("url.Parse(%q): %v", groupURI, err)
			}
			if u.Hostname() != tc.wantHost || u.Port() != "29504" {
				t.Fatalf("url.Parse(%q) host %q port %q, want %q and 29504", groupURI, u.Hostname(), u.Port(), tc.wantHost)
			}
		})
	}
}

// TestGetIPv4GroupUriKeepsLegacyFormat checks that IPv4 and host-name
// configurations produce the same Location URIs as the former
// fmt.Sprintf("%s://%s:%d%s") formatting.
func TestGetIPv4GroupUriKeepsLegacyFormat(t *testing.T) {
	for _, host := range []string{"127.0.0.4", testV4, testFQDN, "udr.omec.svc.cluster.local"} {
		for _, scheme := range []models.UriScheme{models.URISCHEME_HTTP, models.URISCHEME_HTTPS} {
			for _, port := range []int{8000, 29504} {
				ctx := &UDRContext{UriScheme: scheme, RegisterIPv4: host, SBIPort: port}
				want := fmt.Sprintf("%s://%s:%d%s", scheme, host, port, "/nudr-dr/v2")
				if got := ctx.GetIPv4GroupUri(NUDR_DR); got != want {
					t.Errorf("GetIPv4GroupUri() = %q, want %q", got, want)
				}
			}
		}
	}
}
