/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package encoder_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/osdi23p228/fabric/bccsp/factory"
)

func TestEncoder(t *testing.T) {
	factory.InitFactories(nil)

	RegisterFailHandler(Fail)
	RunSpecs(t, "Encoder Suite")
}
