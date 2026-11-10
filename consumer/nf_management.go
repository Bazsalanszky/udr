// Copyright (C) 2026 Intel Corporation
// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"context"
	"strings"

	"github.com/omec-project/openapi/v2"
	"github.com/omec-project/openapi/v2/models"
	"github.com/omec-project/openapi/v2/nrfclient"
	openapiutils "github.com/omec-project/openapi/v2/utils"
	udrContext "github.com/omec-project/udr/context"
	"github.com/omec-project/udr/factory"
	"github.com/omec-project/udr/logger"
)

func newNrfClient(nrfUri string) *nrfclient.Client {
	return nrfclient.New(nrfUri, nrfclient.WithLogger(logger.ConsumerLog))
}

func getNfProfile(udrContext *udrContext.UDRContext, plmnConfig []models.PlmnId) (profile models.NFProfile, err error) {
	if udrContext == nil {
		return profile, openapi.ReportError("udr context has not been initialized. NF profile cannot be built")
	}
	profile = *models.NewNFProfileWithDefaults()
	profile.SetNfInstanceId(udrContext.NfId)
	profile.SetNfType(models.NFTYPE_UDR)
	profile.SetNfStatus(models.NFSTATUS_REGISTERED)
	if len(plmnConfig) > 0 {
		plmnCopy := make([]models.PlmnId, len(plmnConfig))
		copy(plmnCopy, plmnConfig)
		profile.SetPlmnList(plmnCopy)
	}

	version := factory.UdrConfig.Info.Version
	tmpVersion := strings.Split(version, ".")
	versionUri := "v" + tmpVersion[0]
	// RegisterIPv4 may hold a host name, so it must not go into an IPv4 field:
	// a host name goes into fqdn, and each address into the field of its family.
	addrs := udrContext.SbiAddrs()
	if addrs.Empty() {
		return profile, openapi.ReportError("UDR address is empty. NF profile cannot be built")
	}
	openapiutils.SetNFProfileAddresses(&profile, addrs.FQDN, addrs.IPv4, addrs.IPv6)
	nfServiceVersion := models.NewNFServiceVersion(versionUri, version)
	nfService := models.NewNFService("datarepository", models.SERVICENAME_NUDR_DR, []models.NFServiceVersion{*nfServiceVersion}, udrContext.UriScheme, models.NFSERVICESTATUS_REGISTERED)
	nfService.SetApiPrefix(udrContext.GetSbiUri())
	// One endpoint per family (TS 29.510 allows only one address per IpEndPoint), IPv4 first.
	nfService.SetIpEndPoints(openapiutils.NewIpEndPoints(addrs.IPv4, addrs.IPv6, int32(udrContext.SBIPort)))
	if addrs.FQDN != "" {
		nfService.SetFqdn(addrs.FQDN)
	}
	profile.SetNfServices([]models.NFService{*nfService})
	profile.SetNfServiceList(map[string]models.NFService{
		nfService.GetServiceInstanceId(): *nfService,
	})
	udrInfo := models.NewUdrInfo()
	udrInfo.SetSupportedDataSets([]models.DataSetId{
		models.DATASETID_SUBSCRIPTION,
	})
	profile.SetUdrInfo(*udrInfo)
	return profile, nil
}

// SendRegisterNFInstance and SendDeregisterNFInstance take the context of the registration service, which holds
// its registration mutex while they run and cancels the context to end them when a new configuration arrives or
// the NF terminates.
var SendRegisterNFInstance = func(ctx context.Context, plmnConfig []models.PlmnId) (prof *models.NFProfile, resourceNrfUri string, err error) {
	self := udrContext.UDR_Self()
	nfProfile, err := getNfProfile(self, plmnConfig)
	if err != nil {
		return models.NewNFProfileWithDefaults(), "", err
	}

	result, err := newNrfClient(self.NrfUri).Register(ctx, nfProfile)
	if err != nil {
		return models.NewNFProfileWithDefaults(), "", err
	}
	if result.Created {
		logger.ConsumerLog.Debugln("UDR NF profile registered to the NRF")
		if result.NfInstanceID != "" {
			self.NfId = result.NfInstanceID
		}
	} else {
		logger.ConsumerLog.Debugln("UDR NF profile updated with complete replacement")
	}
	return result.Profile, result.NrfURI, nil
}

var SendDeregisterNFInstance = func(ctx context.Context) error {
	logger.ConsumerLog.Infoln("send Deregister NFInstance")

	self := udrContext.UDR_Self()
	return newNrfClient(self.NrfUri).Deregister(ctx, self.NfId)
}

var SendUpdateNFInstance = func(ctx context.Context, patchItem []models.PatchItem) (receivedNfProfile *models.NFProfile, problemDetails *models.ProblemDetails, err error) {
	logger.ConsumerLog.Debugln("send Update NFInstance")

	self := udrContext.UDR_Self()
	return newNrfClient(self.NrfUri).Update(ctx, self.NfId, patchItem)
}
