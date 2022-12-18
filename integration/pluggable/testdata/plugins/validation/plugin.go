/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	validation "github.com/osdi23p228/fabric/core/handlers/validation/api"
	"github.com/osdi23p228/fabric/core/handlers/validation/builtin"
	"github.com/osdi23p228/fabric/integration/pluggable"
)

// go build -buildmode=plugin -o plugin.so

// NewPluginFactory is the function ran by the plugin infrastructure to create a validation plugin factory.
func NewPluginFactory() validation.PluginFactory {
	pluggable.PublishValidationPluginActivation()
	return &builtin.DefaultValidationFactory{}
}
