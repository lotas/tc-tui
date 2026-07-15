package resource

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/taskcluster/tc-tui/taskcluster"
)

type ClientsResource struct {
	tc taskcluster.Taskcluster
}

func NewClientsResource(tc taskcluster.Taskcluster) *ClientsResource {
	return &ClientsResource{tc: tc}
}

func (r *ClientsResource) Name() string        { return "clients" }
func (r *ClientsResource) Aliases() []string   { return []string{"client"} }
func (r *ClientsResource) Description() string { return "Auth clients (credentials) and their scopes" }

func (r *ClientsResource) Columns() []Column {
	return []Column{
		{Title: "CLIENT ID", Expand: true},
		{Title: "DISABLED", Width: 10},
		{Title: "EXPIRES", Width: 24},
	}
}

func (r *ClientsResource) List() ([]Row, error) {
	clients, err := r.tc.GetClients()
	if err != nil {
		return nil, err
	}

	rows := make([]Row, 0, len(clients))
	for _, c := range clients {
		rows = append(rows, Row{
			ID:    c.ClientID,
			Cells: []string{c.ClientID, strconv.FormatBool(c.Disabled), fmt.Sprint(c.Expires)},
		})
	}

	return rows, nil
}

func (r *ClientsResource) Describe(id string) (Detail, error) {
	c, err := r.tc.GetClient(id)
	if err != nil {
		return Detail{}, err
	}

	body := fmt.Sprintf(
		"[green]Description:[white]\n%s\n\n"+
			"%s%s"+
			"[green]Scopes (%d):[white]\n\n%s"+
			"\n\n[green]Expanded Scopes (%d):[white]\n\n%s",
		renderMarkdown(c.Description),
		fieldRow(24, "Created", fmt.Sprint(c.Created), "Last Modified", fmt.Sprint(c.LastModified)),
		fieldRow(24, "Last Rotated", fmt.Sprint(c.LastRotated), "Last Used", fmt.Sprint(c.LastDateUsed)),
		len(c.Scopes), strings.Join(c.Scopes, "\n"),
		len(c.ExpandedScopes), strings.Join(c.ExpandedScopes, "\n"),
	)

	return Detail{Title: fmt.Sprintf("Client :: %s", c.ClientID), Body: body}, nil
}

func (r *ClientsResource) RefreshInterval() time.Duration { return 15 * time.Second }

func (r *ClientsResource) ListWebURL(rootURL, scope string) string {
	return webUIPath(rootURL, "auth/clients")
}

func (r *ClientsResource) DetailWebURL(rootURL, id string) string {
	return webUIPath(rootURL, "auth/clients/"+pathSegment(id))
}
