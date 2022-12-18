/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msptesttools

import (
	"github.com/osdi23p228/fabric/bccsp/factory"
	"github.com/osdi23p228/fabric/core/config/configtest"
	"github.com/osdi23p228/fabric/msp"
	"github.com/osdi23p228/fabric/msp/mgmt"
)

// LoadTestMSPSetup sets up the local MSP
// and a chain MSP for the default chain
func LoadMSPSetupForTesting() error {
	dir := configtest.GetDevMspDir()
	conf, err := msp.GetLocalMspConfig(dir, nil, "SampleOrg")
	if err != nil {
		return err
	}

	err = mgmt.GetLocalMSP(factory.GetDefault()).Setup(conf)
	if err != nil {
		return err
	}

	err = mgmt.GetManagerForChain("testchannelid").Setup([]msp.MSP{mgmt.GetLocalMSP(factory.GetDefault())})
	if err != nil {
		return err
	}

	return nil
}

// Loads the development local MSP for use in testing.  Not valid for production/runtime context
func LoadDevMsp() error {
	mspDir := configtest.GetDevMspDir()
	return mgmt.LoadLocalMsp(mspDir, nil, "SampleOrg")
}
