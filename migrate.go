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
	"os/exec"
)

func migrateRole(openstackPath, oldRole, newRole string) {
	// Step 1. Get role assignments.
	assignments := getRoleAssignments(openstackPath, false, oldRole, newRole)

	// Step 2. Only add newRole, if it doesn't exist for a user/group.
	var roleAddList []roleAssignment
	for _, v := range assignments {
		exist := false
		for _, r := range v.roles {
			if r == newRole {
				exist = true
			}
		}
		if !exist {
			roleAddList = append(roleAddList, v)
		}
	}
	if len(roleAddList) > 0 {
		fmt.Printf("Role %q will be added to the following existing role assignments\n", newRole)
		printRoleAssignments(roleAddList)

		fmt.Println()
		getUserConfirmation()

		for _, v := range roleAddList {
			args := buildRoleMigrateArgs("add", newRole, v)
			_, err := exec.Command(openstackPath, args...).CombinedOutput()
			must(err)
		}
	}

	// Step 3. Only remove oldRole, if newRole exists.
	// Get fresh listing from OpenStack.
	assignments = getRoleAssignments(openstackPath, false, oldRole, newRole)
	var roleRemoveList []roleAssignment
	for _, v := range assignments {
		exist := false
		for _, r := range v.roles {
			if r == newRole {
				exist = true
			}
		}
		if exist {
			roleRemoveList = append(roleRemoveList, v)
		}
	}
	if len(roleRemoveList) > 0 {
		fmt.Printf("Role %q will be removed from the following existing role assignments\n", oldRole)
		printRoleAssignments(roleRemoveList)

		fmt.Println()
		getUserConfirmation()

		for _, v := range roleAddList {
			args := buildRoleMigrateArgs("remove", oldRole, v)
			_, err := exec.Command(openstackPath, args...).CombinedOutput()
			must(err)
		}
	}
}

func getUserConfirmation() {
	yes := "YES"
	fmt.Printf("Type %q to continue: ", yes)
	var input string
	fmt.Scanln(&input)
	if input != yes {
		must(fmt.Errorf("expected %q, got %q", yes, input))
	}
}

func buildRoleMigrateArgs(subcommand, role string, v roleAssignment) []string {
	args := []string{"role", subcommand, role}

	switch {
	case v.user != "":
		args = append(args, "--user", v.user)
	case v.group != "":
		args = append(args, "--group", v.group)
	default:
		// This probably won't happen but just in case.
		must(fmt.Errorf("user/group not found in %+v", v))
	}

	switch {
	case v.system != "":
		args = append(args, "--system", v.system)
	case v.domain != "":
		args = append(args, "--domain", v.domain)
	case v.project != "":
		args = append(args, "--project", v.project)
	default:
		// This probably won't happen but just in case.
		must(fmt.Errorf("system/domain/project not found in %+v", v))
	}

	return args
}
