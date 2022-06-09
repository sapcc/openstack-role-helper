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
	"os"
	"os/exec"

	"github.com/alecthomas/kong"
)

type cli struct {
	List    listCmd    `cmd:"" help:"List role assignments."`
	Migrate migrateCmd `cmd:"" help:"Migrate a role assignment for a user/group on system/domain/project, i.e. add a new role and remove an existing role."`
}

type listCmd struct {
	Roles []string `arg:"" help:"Roles (name or ID)."`
}

type migrateCmd struct {
	OldRole struct {
		// Note: var name needs to be same as enclosing struct
		OldRole string `arg:"" help:"Role (name or ID)."`
		To      struct {
			NewRole struct {
				// Note: var name needs to be same as enclosing struct
				NewRole string `arg:"" help:"Role (name or ID)."`
			} `arg:""`
		} `cmd:""`
	} `arg:""`
}

func main() {
	openstackCmdPath := getExecutablePath("openstack")

	var cli cli //nolint:govet
	ctx := kong.Parse(&cli,
		kong.Name("openstack-role-helper"),
		kong.Description("Wrapper around OpenStack CLI for performing mass role operations."),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{Compact: true}),
	)

	switch ctx.Command() {
	case "list <roles>":
		result := getRoleAssignments(openstackCmdPath, true, cli.List.Roles...)
		printRoleAssignments(result)
	case "migrate <old-role> to <new-role>":
		migrateRole(openstackCmdPath, cli.Migrate.OldRole.OldRole, cli.Migrate.OldRole.To.NewRole.NewRole)
	}
}

func must(err error) {
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		os.Exit(1)
	}
}

// exec.Command() already uses LookPath() in case an executable name is
// provided instead of a path, but we do this manually for two reasons:
// 1. To terminate the program early in case the executable path could not be found.
// 2. To save multiple LookPath() calls for the same executable.
func getExecutablePath(fileName string) string {
	path, err := exec.LookPath(fileName)
	must(err)
	return path
}
