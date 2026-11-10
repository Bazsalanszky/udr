// SPDX-FileCopyrightText: 2026 Balázs Toldi
//
// SPDX-License-Identifier: Apache-2.0

package datarepository

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	udrContext "github.com/omec-project/udr/context"
)

const ueIdParam = "ueId"

// TestEeSubscriptionLocationHeaderOnIPv6 checks that a UDR registered with
// an IPv6 address hands out Location URIs a client can parse: with the former
// "%s://%s:%d" formatting the host came out unbracketed
// ("http://fd00::4:8000/..."), which url.Parse reads as host "fd00::4:8000"
// and no port.
func TestEeSubscriptionLocationHeaderOnIPv6(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetUDRContextForHandlerTests()
	udrSelf := udrContext.UDR_Self()
	udrSelf.RegisterIPv4 = ""
	udrSelf.RegisterIPv6 = "fd00::4"
	t.Cleanup(func() {
		udrSelf.RegisterIPv6 = ""
		resetUDRContextForHandlerTests()
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost,
		"/subscription-data/imsi-001010000000002/context-data/ee-subscriptions",
		bytes.NewBufferString(`{}`),
	)
	request.Header.Set("Content-Type", contentTypeJSON)
	c, _ := gin.CreateTestContext(recorder)
	c.Request = request
	c.Params = gin.Params{{Key: ueIdParam, Value: "imsi-001010000000002"}}

	HTTPCreateEeSubscriptions(c)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d with body %s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}
	location := recorder.Header().Get("Location")
	const wantPrefix = "http://[fd00::4]:8000/nudr-dr/v2/subscription-data/imsi-001010000000002/context-data/ee-subscriptions/"
	if !strings.HasPrefix(location, wantPrefix) {
		t.Fatalf("Location = %q, want prefix %q", location, wantPrefix)
	}
	u, err := url.Parse(location)
	if err != nil {
		t.Fatalf("url.Parse(%q): %v", location, err)
	}
	if u.Hostname() != "fd00::4" || u.Port() != "8000" {
		t.Fatalf("Location host %q port %q, want fd00::4 and 8000", u.Hostname(), u.Port())
	}
}
