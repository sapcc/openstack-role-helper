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
	"sort"
	"strings"

	"github.com/gophercloud/gophercloud/openstack/identity/v3/roles"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/olekukonko/tablewriter"
)

func getRole(name string) roles.Role {
	pages, err := roles.List(identityClient, roles.ListOpts{Name: name}).AllPages()
	must(err)
	extractedRoles, err := roles.ExtractRoles(pages)
	must(err)
	if len(extractedRoles) != 1 {
		must(fmt.Errorf("expected one Role in response, got: %d", len(extractedRoles)))
	}
	return extractedRoles[0]
}

type roleAssignment struct {
	Role  roles.AssignedRole `json:"role,omitempty"`
	Scope struct {
		roles.Scope
		InheritedTo string `json:"OS-INHERIT:inherited_to,omitempty"`
	} `json:"scope,omitempty"`
	User  roles.User  `json:"user,omitempty"`
	Group roles.Group `json:"group,omitempty"`

	Inherited bool `json:"-"` // this field is modified by getRoleAssignments()
	// All roles that are assigned to this particular user/group for this particular scope.
	assignedRoles []roles.AssignedRole `json:"-"`
}

func extractRoleAssignments(r pagination.Page) ([]roleAssignment, error) {
	var s struct {
		RoleAssignments []roleAssignment `json:"role_assignments"`
	}
	err := (r.(roles.RoleAssignmentPage)).ExtractInto(&s)
	return s.RoleAssignments, err
}

func getRoleAssignments(roleNames ...string) []roleAssignment {
	includeNames := true
	var assignments []roleAssignment
	for _, v := range roleNames {
		r := getRole(v)
		pages, err := roles.ListAssignments(identityClient, roles.ListAssignmentsOpts{
			RoleID:       r.ID,
			IncludeNames: &includeNames,
		}).AllPages()
		must(err)
		aList, err := extractRoleAssignments(pages)
		must(err)

		for _, a := range aList {
			if a.Scope.InheritedTo != "" {
				a.Inherited = true
			}
			assignments = append(assignments, a)
		}
	}

	// map[user/group]map[scope]roleAssignment
	uniqueAssignments := make(map[string]map[string]roleAssignment)
	for _, v := range assignments {
		userGroup := userOrGroup(v.User, v.Group)
		scope := projectOrDomain(v.Scope.Project, v.Scope.Domain)

		a, ok := uniqueAssignments[userGroup][scope]
		if !ok {
			if _, ok := uniqueAssignments[userGroup]; !ok {
				uniqueAssignments[userGroup] = make(map[string]roleAssignment)
			}
			a = v
		}
		a.assignedRoles = append(a.assignedRoles, v.Role)
		uniqueAssignments[userGroup][scope] = a
	}

	var result []roleAssignment
	for _, scopeMap := range uniqueAssignments {
		for _, v := range scopeMap {
			result = append(result, v)
		}
	}

	return result
}

func printRoleAssignments(data []roleAssignment) {
	sort.SliceStable(data, func(i, j int) bool {
		// sort by user and group
		return data[i].User.ID != "" && data[j].Group.ID != ""
	})
	sort.SliceStable(data, func(i, j int) bool {
		// sort by project and domain
		return data[i].Scope.Project.ID != "" && data[j].Scope.Domain.ID != ""
	})
	sort.SliceStable(data, func(i, j int) bool {
		// sort by roles
		iRoles := data[i].assignedRoles
		jRoles := data[j].assignedRoles
		if len(iRoles) < len(jRoles) {
			return true
		}

		iRoleNames := make([]string, 0, len(iRoles))
		for _, v := range iRoles {
			iRoleNames = append(iRoleNames, v.Name)
		}
		jRoleNames := make([]string, 0, len(jRoles))
		for _, v := range jRoles {
			jRoleNames = append(jRoleNames, v.Name)
		}

		return strings.Join(iRoleNames, ",") < strings.Join(jRoleNames, ",")
	})

	boolToStr := func(b bool) string {
		if b {
			return "True"
		}
		return ""
	}

	var rows [][]string
	rows = append(rows, []string{"role(s)", "user", "group", "project", "domain", "inherited"})
	for _, v := range data {
		roleNames := make([]string, 0, len(v.assignedRoles))
		for _, v := range v.assignedRoles {
			roleNames = append(roleNames, v.Name)
		}

		rows = append(rows, []string{
			strings.Join(roleNames, ","),
			nameAtScope(v.User.Name, v.User.Domain.Name),
			nameAtScope(v.Group.Name, v.Group.Domain.Name),
			nameAtScope(v.Scope.Project.Name, v.Scope.Project.Domain.Name),
			v.Scope.Domain.Name,
			boolToStr(v.Inherited),
		})
	}

	t := tablewriter.NewWriter(os.Stdout)
	t.SetHeader(rows[0])
	t.AppendBulk(rows[1:])
	t.Render()
}

func nameAtScope(name, scope string) string {
	if name == "" {
		return ""
	}
	return fmt.Sprintf("%s@%s", name, scope)
}

func userOrGroup(user roles.User, group roles.Group) string {
	if user.ID != "" {
		return nameAtScope(user.Name, user.Domain.Name)
	}
	return nameAtScope(group.Name, group.Domain.Name)
}

func projectOrDomain(project roles.Project, domain roles.Domain) string {
	if project.ID != "" {
		return nameAtScope(project.Name, project.Domain.Name)
	}
	return domain.Name
}
