package resource

import (
	"fmt"
	"strings"
	"time"

	"github.com/taskcluster/tc-tui/taskcluster"
)

type RolesResource struct {
	tc taskcluster.Taskcluster
}

func NewRolesResource(tc taskcluster.Taskcluster) *RolesResource {
	return &RolesResource{tc: tc}
}

func (r *RolesResource) Name() string      { return "roles" }
func (r *RolesResource) Aliases() []string { return []string{"role"} }

func (r *RolesResource) Columns() []Column {
	return []Column{{Title: "ROLE ID"}}
}

func (r *RolesResource) List() ([]Row, error) {
	roles, err := r.tc.GetRoles()
	if err != nil {
		return nil, err
	}

	rows := make([]Row, 0, len(roles))
	for _, role := range roles {
		rows = append(rows, Row{ID: role.RoleID, Cells: []string{role.RoleID}})
	}

	return rows, nil
}

func (r *RolesResource) Describe(id string) (Detail, error) {
	role, err := r.tc.GetRole(id)
	if err != nil {
		return Detail{}, err
	}

	body := fmt.Sprintf(
		"[green]Description:[white] %s\n\n[green]Created:[white] %s\n"+
			"[green]Last Modified:[white] %s\n\n[green]Scopes (%d):[white]\n\n%s"+
			"\n\n[green]Expanded Scopes (%d):[white]\n\n%s",
		role.Description,
		role.Created,
		role.LastModified,
		len(role.Scopes),
		strings.Join(role.Scopes, "\n"),
		len(role.ExpandedScopes),
		strings.Join(role.ExpandedScopes, "\n"),
	)

	return Detail{
		Title: fmt.Sprintf("Role :: %s", role.RoleID),
		Body:  body,
	}, nil
}

func (r *RolesResource) RefreshInterval() time.Duration {
	return 15 * time.Second
}
