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
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/olekukonko/tablewriter"
)

// roleResult is the JSON response that is returned by OpenStack CLI for
// `role assignment list` sub-command.
type roleAssignmentResultFromOS struct {
	Domain    string `json:"Domain"`
	Group     string `json:"Group"`
	Inherited bool   `json:"Inherited"`
	Project   string `json:"Project"`
	Role      string `json:"Role"`
	System    string `json:"System"`
	User      string `json:"User"`
}

type roleAssignment struct {
	user    string
	group   string
	system  string
	domain  string
	project string
	roles   []string
}

func getRoleAssignments(openstackPath string, names bool, roles ...string) []roleAssignment {
	var roleAssignmentFromOS []roleAssignmentResultFromOS
	for _, r := range roles {
		args := []string{"role", "assignment", "list", "-f", "json", "--role", r}
		if names {
			args = append(args, "--names")
		}
		out, err := exec.Command(openstackPath, args...).CombinedOutput()
		must(err)

		var data []roleAssignmentResultFromOS
		err = json.Unmarshal(out, &data)
		must(err)
		roleAssignmentFromOS = append(roleAssignmentFromOS, data...)
	}

	// map[user/group]map[system/domain/project]roleAssignment
	assignments := make(map[string]map[string]roleAssignment)
	for _, v := range roleAssignmentFromOS {
		var user string
		switch {
		case v.User != "":
			user = v.User
		case v.Group != "":
			user = v.Group
		}
		var scope string
		switch {
		case v.System != "":
			scope = v.System
		case v.Domain != "":
			scope = v.Domain
		case v.Project != "":
			scope = v.Project
		}

		ra, exists := assignments[user][scope]
		if !exists {
			if _, ok := assignments[user]; !ok {
				assignments[user] = make(map[string]roleAssignment)
			}
			ra = roleAssignment{
				user:    v.User,
				group:   v.Group,
				system:  v.System,
				domain:  v.Domain,
				project: v.Project,
			}
		}
		ra.roles = append(ra.roles, v.Role)
		assignments[user][scope] = ra
	}

	var result []roleAssignment
	for _, scopeMap := range assignments {
		for _, v := range scopeMap {
			sort.Strings(v.roles)
			result = append(result, v)
		}
	}
	return result
}

func printRoleAssignments(data []roleAssignment) {
	sort.SliceStable(data, func(i, j int) bool {
		// sort by user and group
		return data[i].user != "" && data[j].group != ""
	})
	sort.SliceStable(data, func(i, j int) bool {
		// sort by project, domain, and system
		if data[i].project != "" && data[j].domain != "" {
			return true
		}
		return data[i].domain != "" && data[j].system != ""
	})
	sort.SliceStable(data, func(i, j int) bool {
		// sort by roles
		iRoles := data[i].roles
		jRoles := data[j].roles
		if len(iRoles) < len(jRoles) {
			return true
		}
		return strings.Join(iRoles, ",") < strings.Join(jRoles, ",")
	})

	var rows [][]string
	rows = append(rows, []string{"role(s)", "user", "group", "project", "domain", "system"})
	for _, v := range data {
		rows = append(rows, []string{
			strings.Join(v.roles, ","), v.user, v.group, v.project, v.domain, v.system,
		})
	}

	t := tablewriter.NewWriter(os.Stdout)
	t.SetHeader(rows[0])
	t.AppendBulk(rows[1:])
	t.Render()
}
