package resource

import (
	"fmt"
	"time"

	"github.com/taskcluster/tc-tui/taskcluster"
)

type SecretsResource struct {
	tc taskcluster.Taskcluster
}

func NewSecretsResource(tc taskcluster.Taskcluster) *SecretsResource {
	return &SecretsResource{tc: tc}
}

func (r *SecretsResource) Name() string      { return "secrets" }
func (r *SecretsResource) Aliases() []string { return []string{"secret"} }
func (r *SecretsResource) Description() string {
	return "Secret names and their values (fetched on open)"
}

func (r *SecretsResource) Columns() []Column {
	return []Column{{Title: "NAME", Expand: true}}
}

func (r *SecretsResource) List() ([]Row, error) {
	names, err := r.tc.GetSecrets()
	if err != nil {
		return nil, err
	}

	rows := make([]Row, 0, len(names))
	for _, name := range names {
		rows = append(rows, Row{ID: name, Cells: []string{name}})
	}

	return rows, nil
}

func (r *SecretsResource) Describe(id string) (Detail, error) {
	secret, err := r.tc.GetSecret(id)
	if err != nil {
		return Detail{}, err
	}

	body := fmt.Sprintf(
		"[green]Expires:[white] %s\n\n[green]Secret:[white]\n%s\n",
		secret.Expires, renderYAML(secret.Secret),
	)

	return Detail{Title: fmt.Sprintf("Secret :: %s", id), Body: body}, nil
}

func (r *SecretsResource) RefreshInterval() time.Duration { return 15 * time.Second }

func (r *SecretsResource) ListWebURL(rootURL, scope string) string {
	return webUIPath(rootURL, "secrets")
}

func (r *SecretsResource) DetailWebURL(rootURL, id string) string {
	return webUIPath(rootURL, "secrets/"+pathSegment(id))
}
