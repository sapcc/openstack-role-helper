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
	"encoding/json"
	"fmt"
	"os/exec"
)

func migrateRole(oldRole, newRole string) {
	// Step 1. Get IDs for the user give roles (since the user could have provided role
	// names instead of IDs).
	oldRoleID := getRoleID(oldRole)
	newRoleID := getRoleID(newRole)

	// Step 2. Get role assignments.
	assignments := getRoleAssignments(false, oldRoleID, newRoleID)

	// Step 2. Find which user/group don't have the newRole and add the newRole to them.
	var roleAddList []roleAssignment
	for _, v := range assignments {
		exist := false
		for _, r := range v.roles {
			if r == newRoleID {
				exist = true
			}
		}
		if !exist {
			roleAddList = append(roleAddList, v)
		}
	}
	if len(roleAddList) > 0 {
		fmt.Printf("Role \"%s = %s\" will be added to the following role assignments:\n", newRole, newRoleID)
		printRoleAssignments(roleAddList)

		fmt.Println()
		getUserConfirmation()
		fmt.Println()

		for _, v := range roleAddList {
			args := buildRoleMigrateArgs("add", newRoleID, v)
			out, err := exec.Command(openstackCmdPath, args...).CombinedOutput()
			fmt.Println(string(out))
			must(err)
		}
	}

	// Step 3. Remove oldRole from those user/group where both oldRole and newRole exists.
	// Get fresh listing from OpenStack.
	assignments = getRoleAssignments(false, oldRoleID, newRoleID)
	var roleRemoveList []roleAssignment
	for _, v := range assignments {
		foundOld := false
		foundNew := false
		for _, r := range v.roles {
			if r == oldRoleID {
				foundOld = true
			}
			if r == newRoleID {
				foundNew = true
			}
		}
		if foundOld && foundNew {
			roleRemoveList = append(roleRemoveList, v)
		}
	}
	if len(roleRemoveList) > 0 {
		fmt.Printf("Role \"%s = %s\" will be removed from the following role assignments:\n", oldRole, oldRoleID)
		printRoleAssignments(roleRemoveList)

		fmt.Println()
		getUserConfirmation()
		fmt.Println()

		for _, v := range roleRemoveList {
			args := buildRoleMigrateArgs("remove", oldRoleID, v)
			out, err := exec.Command(openstackCmdPath, args...).CombinedOutput()
			fmt.Println(string(out))
			must(err)
		}
	}
}

func getRoleID(name string) string {
	args := []string{"role", "show", name, "-f", "json"}
	out, err := exec.Command(openstackCmdPath, args...).CombinedOutput()
	must(err)

	var data struct {
		ID string `json:"id"`
	}
	err = json.Unmarshal(out, &data)
	must(err)
	if data.ID == "" {
		must(fmt.Errorf("could not find ID for role: %q", name))
	}

	return data.ID
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
