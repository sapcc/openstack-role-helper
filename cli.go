package main

import "os"

type cli struct {
	Debug bool `short:"d" help:"Enable debug mode (will print API requests and responses)."`
	openstackFlags

	List    listCmd    `cmd:"" help:"List role assignments."`
	Migrate migrateCmd `cmd:"" help:"Migrate a role assignment for a user/group on a project/domain, i.e. add a new role and remove an existing role. Note: inherited role assignments are skipped."`
}

type openstackFlags struct {
	OSAuthURL           string `help:"Authentication URL."`
	OSUsername          string `help:"Username."`
	OSPassword          string `help:"User's Password."`
	OSUserDomainID      string `help:"User's domain ID."`
	OSUserDomainName    string `help:"User's domain name."`
	OSProjectID         string `help:"Project ID to scope to."`
	OSProjectName       string `help:"Project name to scope to."`
	OSProjectDomainID   string `help:"Domain ID containing project to scope to."`
	OSProjectDomainName string `help:"Domain name containing project to scope to."`
}

type listCmd struct {
	RoleNames []string `arg:"" help:"Role name(s)."`
}

type migrateCmd struct {
	OldRoleName struct {
		// Note: var name needs to be same as enclosing struct
		OldRoleName string `arg:""`
		To          struct {
			NewRoleName struct {
				// Note: var name needs to be same as enclosing struct
				NewRoleName string `arg:""`
			} `arg:""`
		} `cmd:""`
	} `arg:""`
}

///////////////////////////////////////////////////////////////////////////////
// Helper Functions

func setenvIfVal(key, val string) error {
	if val == "" {
		return nil
	}
	return os.Setenv(key, val)
}

func updateOpenStackEnvVars(v *openstackFlags) {
	must(setenvIfVal("OS_AUTH_URL", v.OSAuthURL))
	must(setenvIfVal("OS_USERNAME", v.OSUsername))
	must(setenvIfVal("OS_PASSWORD", v.OSPassword))
	must(setenvIfVal("OS_USER_DOMAIN_ID", v.OSUserDomainID))
	must(setenvIfVal("OS_USER_DOMAIN_NAME", v.OSUserDomainName))
	must(setenvIfVal("OS_PROJECT_ID", v.OSProjectID))
	must(setenvIfVal("OS_PROJECT_NAME", v.OSProjectName))
	must(setenvIfVal("OS_PROJECT_DOMAIN_ID", v.OSProjectDomainID))
	must(setenvIfVal("OS_PROJECT_DOMAIN_NAME", v.OSProjectDomainName))
}
