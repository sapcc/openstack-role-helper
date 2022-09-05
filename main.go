// Copyright 2022 SAP SE
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"net/http"

	"github.com/alecthomas/kong"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/utils/client"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/sapcc/go-bits/must"
)

// identityClient is the ServiceClient for Keystone v3.
var identityClient *gophercloud.ServiceClient

func main() {
	var cli cli //nolint:govet
	ctx := kong.Parse(&cli,
		kong.Name("openstack-role-helper"),
		kong.Description("Tool for performing mass role operations."),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{Compact: true}),
	)

	identityClient = must.Return(authenticate(&cli.openstackFlags, cli.Debug))

	switch ctx.Command() {
	case "list <role-names>":
		result := getRoleAssignments(cli.List.RoleNames...)
		printRoleAssignments(result)
	case "migrate <old-role-name> to <new-role-name>":
		migrateRole(cli.Migrate.OldRoleName.OldRoleName, cli.Migrate.OldRoleName.To.NewRoleName.NewRoleName)
	}
}

// authenticate authenticates against OpenStack and returns the necessary
// service clients.
func authenticate(osFlags *openstackFlags, debug bool) (identityClient *gophercloud.ServiceClient, err error) {
	// Update OpenStack environment variables, if value provided as flag.
	updateOpenStackEnvVars(osFlags)

	ao, err := clientconfig.AuthOptions(nil)
	if err != nil {
		return nil, fmt.Errorf("could not get auth variables: %s", err.Error())
	}

	provider, err := openstack.NewClient(ao.IdentityEndpoint)
	if err != nil {
		return nil, fmt.Errorf("cannot create an OpenStack client: %s", err.Error())
	}
	if debug {
		provider.HTTPClient = http.Client{
			Transport: &client.RoundTripper{
				Rt:     &http.Transport{},
				Logger: &client.DefaultLogger{},
			},
		}
	}

	err = openstack.Authenticate(provider, *ao)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to OpenStack: %s", err.Error())
	}

	identityClient, err = openstack.NewIdentityV3(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, fmt.Errorf("could not initialize identity client: %s", err.Error())
	}

	return identityClient, nil
}
