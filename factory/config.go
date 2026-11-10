// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0
//

/*
 * UDR Configuration Factory
 */

package factory

import (
	utilLogger "github.com/omec-project/util/logger"
)

const (
	UDR_EXPECTED_CONFIG_VERSION = "1.0.0"
)

type Config struct {
	Info          *Info              `yaml:"info"`
	Configuration *Configuration     `yaml:"configuration"`
	Logger        *utilLogger.Logger `yaml:"logger"`
	CfgLocation   string
}

type Info struct {
	Version     string `yaml:"version,omitempty"`
	Description string `yaml:"description,omitempty"`
}

const (
	UDR_DEFAULT_IPV4     = "127.0.0.4"
	UDR_DEFAULT_PORT     = "8000"
	UDR_DEFAULT_PORT_INT = 8000
)

type Configuration struct {
	Sbi      *Sbi     `yaml:"sbi"`
	Mongodb  *Mongodb `yaml:"mongodb"`
	NrfUri   string   `yaml:"nrfUri"`
	WebuiUri string   `yaml:"webuiUri"`
}

type Sbi struct {
	Tls             *Tls   `yaml:"tls,omitempty"`
	Scheme          string `yaml:"scheme"`
	RegisterIPv4    string `yaml:"registerIPv4,omitempty"`    // IPv4 address, host name, or env var name (POD_IP, POD_IPS) registered at the NRF.
	RegisterIPv6    string `yaml:"registerIPv6,omitempty"`    // IPv6 address, host name, or env var name registered at the NRF.
	BindingAddr     string `yaml:"bindingAddr,omitempty"`     // Listen address of any family; wins over bindingIPv4.
	BindingIPv4     string `yaml:"bindingIPv4,omitempty"`     // Legacy alias of bindingAddr.
	PreferredFamily string `yaml:"preferredFamily,omitempty"` // IPv4 (default) or IPv6, for the own URIs and peer selection.
	Port            int    `yaml:"port"`
}

type Tls struct {
	Log string `yaml:"log"`
	Pem string `yaml:"pem"`
	Key string `yaml:"key"`
}

type Mongodb struct {
	Name           string `yaml:"name,omitempty"`
	Url            string `yaml:"url,omitempty"`
	AuthKeysDbName string `yaml:"authKeysDbName"`
	AuthUrl        string `yaml:"authUrl"`
}

func (c *Config) GetVersion() string {
	if c.Info != nil && c.Info.Version != "" {
		return c.Info.Version
	}
	return ""
}
