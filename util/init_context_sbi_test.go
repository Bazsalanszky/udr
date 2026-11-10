// SPDX-FileCopyrightText: 2026 Balázs Toldi
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"testing"

	"github.com/omec-project/udr/context"
	"github.com/omec-project/udr/factory"
	"github.com/omec-project/util/sbiaddr"
)

const (
	podV4      = "10.42.0.4"
	podV6      = "fd00:42::4"
	svcName    = "udr"
	v4Wildcard = "0.0.0.0"
	defaultURI = "http://127.0.0.4:29504"
	podV4URI   = "http://10.42.0.4:29504"
	podV6URI   = "http://[fd00:42::4]:29504"
	svcNameURI = "http://udr:29504"
)

func TestInitUdrContext_SbiAddresses(t *testing.T) {
	cases := []struct {
		name         string
		sbi          factory.Sbi
		env          map[string]string
		wantRegister string // RegisterIPv4
		wantIPv6     string // RegisterIPv6
		wantBinding  string
		wantPref     sbiaddr.Preference
		wantURI      string
	}{
		{
			name:         "IPv4 literal, as before",
			sbi:          factory.Sbi{RegisterIPv4: "10.0.0.11", BindingIPv4: v4Wildcard},
			wantRegister: "10.0.0.11", wantBinding: v4Wildcard, wantURI: "http://10.0.0.11:29504",
		},
		{
			name:         "chart default: service name",
			sbi:          factory.Sbi{RegisterIPv4: svcName, BindingIPv4: svcName},
			wantRegister: svcName, wantBinding: svcName, wantURI: svcNameURI,
		},
		{
			name:         "nothing set: defaults",
			sbi:          factory.Sbi{},
			wantRegister: factory.UDR_DEFAULT_IPV4, wantBinding: sbiaddr.DefaultBindHost,
			wantURI: defaultURI,
		},
		{
			name:         "invalid registerIPv4 falls back to the default",
			sbi:          factory.Sbi{RegisterIPv4: "10.0.0.256"},
			wantRegister: factory.UDR_DEFAULT_IPV4, wantBinding: sbiaddr.DefaultBindHost,
			wantURI: defaultURI,
		},
		{
			name:     "IPv6 literal under registerIPv4 goes to RegisterIPv6",
			sbi:      factory.Sbi{RegisterIPv4: "FD00:42:0::4"},
			wantIPv6: podV6, wantBinding: sbiaddr.DefaultBindHost, wantURI: podV6URI,
			wantRegister: "",
		},
		{
			name:     "POD_IP holding IPv6 under registerIPv4 and bindingIPv4",
			sbi:      factory.Sbi{RegisterIPv4: sbiaddr.EnvPodIP, BindingIPv4: sbiaddr.EnvPodIP},
			env:      map[string]string{sbiaddr.EnvPodIP: podV6},
			wantIPv6: podV6, wantBinding: podV6, wantURI: podV6URI,
		},
		{
			name:         "POD_IP holding IPv4",
			sbi:          factory.Sbi{RegisterIPv4: sbiaddr.EnvPodIP, BindingIPv4: sbiaddr.EnvPodIP},
			env:          map[string]string{sbiaddr.EnvPodIP: podV4},
			wantRegister: podV4, wantBinding: podV4, wantURI: podV4URI,
		},
		{
			name:         "POD_IP unset falls back to the default",
			sbi:          factory.Sbi{RegisterIPv4: sbiaddr.EnvPodIP},
			wantRegister: factory.UDR_DEFAULT_IPV4, wantBinding: sbiaddr.DefaultBindHost,
			wantURI: defaultURI,
		},
		{
			name:         "POD_IPS dual stack",
			sbi:          factory.Sbi{RegisterIPv4: sbiaddr.EnvPodIPs},
			env:          map[string]string{sbiaddr.EnvPodIPs: podV6 + "," + podV4},
			wantRegister: podV4, wantIPv6: podV6, wantBinding: sbiaddr.DefaultBindHost,
			wantURI: podV4URI,
		},
		{
			name: "POD_IPS dual stack, preferredFamily IPv6",
			sbi: factory.Sbi{
				RegisterIPv4: sbiaddr.EnvPodIP, RegisterIPv6: sbiaddr.EnvPodIPs,
				PreferredFamily: "IPv6",
			},
			env:          map[string]string{sbiaddr.EnvPodIP: podV4, sbiaddr.EnvPodIPs: podV4 + "," + podV6},
			wantRegister: podV4, wantIPv6: podV6, wantBinding: sbiaddr.DefaultBindHost,
			wantPref: sbiaddr.PreferIPv6, wantURI: podV6URI,
		},
		{
			name:         "service name with registerIPv6",
			sbi:          factory.Sbi{RegisterIPv4: svcName, RegisterIPv6: podV6},
			wantRegister: svcName, wantIPv6: podV6, wantBinding: sbiaddr.DefaultBindHost,
			wantURI: svcNameURI,
		},
		{
			name:         "unknown preferredFamily keeps IPv4",
			sbi:          factory.Sbi{RegisterIPv4: podV4, RegisterIPv6: podV6, PreferredFamily: "ipv5"},
			wantRegister: podV4, wantIPv6: podV6, wantBinding: sbiaddr.DefaultBindHost,
			wantURI: podV4URI,
		},
		{
			name:         "bindingAddr wins over bindingIPv4",
			sbi:          factory.Sbi{RegisterIPv4: podV4, BindingAddr: "::", BindingIPv4: v4Wildcard},
			wantRegister: podV4, wantBinding: "::", wantURI: podV4URI,
		},
		{
			name:         "bracketed bindingIPv4 is unwrapped",
			sbi:          factory.Sbi{RegisterIPv4: podV4, BindingIPv4: "[::]"},
			wantRegister: podV4, wantBinding: "::", wantURI: podV4URI,
		},
		{
			name:         "bindingAddr POD_IPS cannot be bound and uses the wildcard",
			sbi:          factory.Sbi{RegisterIPv4: podV4, BindingAddr: sbiaddr.EnvPodIPs},
			env:          map[string]string{sbiaddr.EnvPodIPs: podV4 + "," + podV6},
			wantRegister: podV4, wantBinding: sbiaddr.DefaultBindHost, wantURI: podV4URI,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(sbiaddr.EnvPodIP, "")
			t.Setenv(sbiaddr.EnvPodIPs, "")
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			origUdrConfig := factory.UdrConfig
			t.Cleanup(func() { factory.UdrConfig = origUdrConfig })
			sbi := tc.sbi
			sbi.Scheme = "http"
			sbi.Port = 29504
			factory.UdrConfig = factory.Config{
				Info:          &factory.Info{Version: testVersion, Description: testDescription},
				Configuration: &factory.Configuration{Sbi: &sbi, NrfUri: testNrfUri},
			}
			// Stale values from an earlier initialisation must not survive.
			ctx := &context.UDRContext{RegisterIPv6: "fd00::dead", SbiPreference: sbiaddr.PreferIPv6}

			InitUdrContext(ctx)

			if ctx.RegisterIPv4 != tc.wantRegister {
				t.Errorf("RegisterIPv4 = %q, want %q", ctx.RegisterIPv4, tc.wantRegister)
			}
			if ctx.RegisterIPv6 != tc.wantIPv6 {
				t.Errorf("RegisterIPv6 = %q, want %q", ctx.RegisterIPv6, tc.wantIPv6)
			}
			if ctx.BindingIPv4 != tc.wantBinding {
				t.Errorf("BindingIPv4 = %q, want %q", ctx.BindingIPv4, tc.wantBinding)
			}
			if ctx.SbiPreference != tc.wantPref {
				t.Errorf("SbiPreference = %v, want %v", ctx.SbiPreference, tc.wantPref)
			}
			if got := ctx.GetSbiUri(); got != tc.wantURI {
				t.Errorf("GetSbiUri() = %q, want %q", got, tc.wantURI)
			}
		})
	}
}
