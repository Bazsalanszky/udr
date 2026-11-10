// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0
//

package util

import (
	"cmp"
	"strings"

	"github.com/google/uuid"
	"github.com/omec-project/openapi/v2/models"
	"github.com/omec-project/udr/context"
	"github.com/omec-project/udr/factory"
	"github.com/omec-project/udr/logger"
	"github.com/omec-project/util/sbiaddr"
)

func InitUdrContext(context *context.UDRContext) {
	config := factory.UdrConfig
	logger.UtilLog.Infof("udrconfig Info: Version[%s] Description[%s]", config.Info.Version, config.Info.Description)
	configuration := config.Configuration
	context.NfId = uuid.New().String()
	context.RegisterIPv4 = factory.UDR_DEFAULT_IPV4 // default localhost
	context.RegisterIPv6 = ""
	context.SbiPreference = sbiaddr.PreferIPv4
	context.SBIPort = factory.UDR_DEFAULT_PORT_INT // default port
	if sbi := configuration.Sbi; sbi != nil {
		context.UriScheme = models.UriScheme(sbi.Scheme)
		setRegisterAddrs(context, sbi)
		if sbi.Port != 0 {
			context.SBIPort = sbi.Port
		}
		if tls := sbi.Tls; tls != nil {
			if tls.Key != "" {
				context.Key = tls.Key
			}
			if tls.Pem != "" {
				context.PEM = tls.Pem
			}
		}
		var err error
		if context.SbiPreference, err = sbiaddr.ParsePreference(sbi.PreferredFamily); err != nil {
			logger.UtilLog.Warnf("SBI preferredFamily: %v; using IPv4", err)
		}
		setBindingAddr(context, sbi)
	}
	if configuration.NrfUri != "" {
		context.NrfUri = configuration.NrfUri
	} else {
		logger.UtilLog.Warnln("NRF Uri is empty. Using localhost as NRF IPv4 address")
		context.NrfUri = sbiaddr.URI(string(context.UriScheme), "127.0.0.1", 29510)
	}
}

// setRegisterAddrs resolves registerIPv4 and registerIPv6 (each an address, a
// host name or the name of an environment variable such as POD_IP or POD_IPS)
// into the addresses the UDR registers at the NRF. RegisterIPv4 keeps holding
// the IPv4 address or the host name, as before; an IPv6 address goes to
// RegisterIPv6, whichever key it came from.
func setRegisterAddrs(context *context.UDRContext, sbi *factory.Sbi) {
	addrs, err := sbiaddr.ResolveRegister(sbi.RegisterIPv4, sbi.RegisterIPv6)
	if err != nil {
		logger.UtilLog.Warnf("SBI register address: %v", err)
	}
	if addrs.Empty() {
		addrs.IPv4 = factory.UDR_DEFAULT_IPV4
	}
	if addrs.FQDN != "" && addrs.IPv4 != "" {
		logger.UtilLog.Warnf("SBI register address: registering FQDN %s, not IPv4 %s", addrs.FQDN, addrs.IPv4)
	}
	context.RegisterIPv4 = cmp.Or(addrs.FQDN, addrs.IPv4)
	context.RegisterIPv6 = addrs.IPv6
}

// setBindingAddr sets the SBI listen host from bindingAddr, or from its legacy
// alias bindingIPv4. Either may name an environment variable; the default is
// 0.0.0.0, which also accepts IPv6 connections.
func setBindingAddr(context *context.UDRContext, sbi *factory.Sbi) {
	if strings.TrimSpace(sbi.BindingAddr) == "" && strings.TrimSpace(sbi.BindingIPv4) == "" {
		logger.UtilLog.Warnln("SBI binding address is not set. Using the 0.0.0.0 address as default")
	}
	var err error
	if context.BindingIPv4, err = sbiaddr.BindHost(sbi.BindingAddr, sbi.BindingIPv4); err != nil {
		logger.UtilLog.Warnf("SBI binding address: %v", err)
	}
}
