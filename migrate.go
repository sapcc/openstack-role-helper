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
	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/roles"
)

func migrateRole(oldRoleName, newRoleName string) {
	// Step 1. Get IDs for the user given role name.
	oldRole := getRole(oldRoleName)
	newRole := getRole(newRoleName)

	// Step 2. Get role assignments.
	assignments := getRoleAssignments(oldRoleName, newRoleName)

	// Step 2. Find which user/group don't have the newRole and add the newRole to them.
	var roleAddList []roleAssignment
	for _, v := range assignments {
		exist := false
		for _, r := range v.assignedRoles {
			if r.ID == newRole.ID {
				exist = true
			}
		}
		if !exist {
			roleAddList = append(roleAddList, v)
		}
	}
	if len(roleAddList) > 0 {
		fmt.Printf("Role %q will be added to the following role assignments:\n", newRoleName)
		printRoleAssignments(roleAddList)

		getUserConfirmation()
		fmt.Println()

		for _, v := range roleAddList {
			err := assignRole(newRole, v)
			userGroup := userOrGroup(v.User, v.Group)
			projectDomain := projectOrDomain(v.Scope.Project, v.Scope.Domain)
			if err != nil {
				fmt.Printf("ERROR: could not assign role %q to %q on %q: %s\n",
					newRoleName, userGroup, projectDomain, err.Error())
			} else {
				fmt.Printf("INFO: successfully assigned role %q to %q on %q\n",
					newRoleName, userGroup, projectDomain)
			}
		}

		fmt.Println(strings.Repeat("=", 79))
		fmt.Println()
	}

	// Step 3. Remove oldRole from those user/group where both oldRole and newRole exists.
	assignments = getRoleAssignments(oldRoleName, newRoleName) // get up-to-date assignments list from Keystone
	var roleRemoveList []roleAssignment
	for _, v := range assignments {
		foundOld := false
		foundNew := false
		for _, r := range v.assignedRoles {
			switch r.ID {
			case oldRole.ID:
				foundOld = true
			case newRole.ID:
				foundNew = true
			}
		}
		if foundOld && foundNew {
			roleRemoveList = append(roleRemoveList, v)
		}
	}
	if len(roleRemoveList) > 0 {
		fmt.Printf("Role %q will be removed from the following role assignments:\n", oldRoleName)
		printRoleAssignments(roleRemoveList)

		getUserConfirmation()
		fmt.Println()

		for _, v := range roleRemoveList {
			err := unassignRole(oldRole, v)
			userGroup := userOrGroup(v.User, v.Group)
			projectDomain := projectOrDomain(v.Scope.Project, v.Scope.Domain)
			if err != nil {
				fmt.Printf("ERROR: could not unassign role %q from %q on %q: %s\n",
					oldRoleName, userGroup, projectDomain, err.Error())
			} else {
				fmt.Printf("INFO: successfully unassigned role %q from %q on %q\n",
					oldRoleName, userGroup, projectDomain)
			}
		}
	}
}

func assignRole(role roles.Role, assignment roleAssignment) error {
	url, err := buildAssignURL(role, assignment)
	if err != nil {
		return err
	}

	resp, err := identityClient.Put(url, nil, nil, &gophercloud.RequestOpts{
		OkCodes: []int{204},
	})
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return err
	}

	return nil
}

func unassignRole(role roles.Role, assignment roleAssignment) error {
	url, err := buildAssignURL(role, assignment)
	if err != nil {
		return err
	}

	resp, err := identityClient.Delete(url, &gophercloud.RequestOpts{
		OkCodes: []int{204},
	})
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return err
	}

	return nil
}

///////////////////////////////////////////////////////////////////////////////
// Helper functions

const (
	rolePath                = "roles"
	osInheritancePath       = "OS-INHERIT"
	inheritedToProjectsPath = "inherited_to_projects"
)

func assignURL(targetType, targetID, actorType, actorID, roleID string) string {
	return identityClient.ServiceURL(targetType, targetID, actorType, actorID, rolePath, roleID)
}

func assignWithInheritanceURL(targetType, targetID, actorType, actorID, roleID string) string {
	return identityClient.ServiceURL(osInheritancePath, targetType, targetID, actorType, actorID, rolePath, roleID, inheritedToProjectsPath)
}

func buildAssignURL(role roles.Role, assignment roleAssignment) (string, error) {
	opts := roles.AssignOpts{
		GroupID: assignment.Group.ID,
	}
	if opts.GroupID == "" {
		opts.UserID = assignment.User.ID
	}
	opts.DomainID = assignment.Scope.Domain.ID
	if opts.DomainID == "" {
		opts.ProjectID = assignment.Scope.Project.ID
	}

	// Check xor conditions
	_, err := gophercloud.BuildRequestBody(opts, "")
	if err != nil {
		return "", err
	}

	// Get corresponding URL
	var targetID string
	var targetType string
	if opts.ProjectID != "" {
		targetID = opts.ProjectID
		targetType = "projects"
	} else {
		targetID = opts.DomainID
		targetType = "domains"
	}

	var actorID string
	var actorType string
	if opts.UserID != "" {
		actorID = opts.UserID
		actorType = "users"
	} else {
		actorID = opts.GroupID
		actorType = "groups"
	}

	urlFunc := assignURL
	if assignment.Inherited {
		urlFunc = assignWithInheritanceURL
	}
	return urlFunc(targetType, targetID, actorType, actorID, role.ID), nil
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
