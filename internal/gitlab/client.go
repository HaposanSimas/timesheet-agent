// Package gitlab mengambil commit dari GitLab self-hosted (git.smmf.co.id).
//
// Catatan: endpoint commits GitLab tidak memfilter author di sisi server secara
// andal, jadi kita ambil semua commit di rentang lalu filter lokal berdasarkan
// author_name / author_email.
package gitlab

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/HaposanSimas/timesheet-agent/internal/vcs"
)

const perPage = 100

// Client adalah klien GitLab minimal.
type Client struct {
	base  string // mis. https://git.smmf.co.id
	token string
	http  *http.Client
	names []string // author_name yang cocok (lowercase)
	mails []string // author_email yang cocok (lowercase)
}

// New membuat Client. insecureTLS=true melewati verifikasi sertifikat
// (kadang perlu untuk GitLab internal dengan CA sendiri; pakai hati-hati).
func New(baseURL, token string, names, emails []string, insecureTLS bool) *Client {
	tr := &http.Transport{}
	if insecureTLS {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // opt-in untuk CA internal
	}
	return &Client{
		base:  strings.TrimRight(baseURL, "/"),
		token: token,
		http:  &http.Client{Timeout: 30 * time.Second, Transport: tr},
		names: lower(names),
		mails: lower(emails),
	}
}

type apiCommit struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Message     string    `json:"message"`
	CommittedAt time.Time `json:"committed_date"`
	AuthorName  string    `json:"author_name"`
	AuthorEmail string    `json:"author_email"`
}

// FetchAll menarik commit milik author dari banyak project pada [since, until).
func (c *Client) FetchAll(ctx context.Context, projects []string, since, until time.Time, loc *time.Location) ([]vcs.Commit, error) {
	var all []vcs.Commit
	for _, p := range projects {
		commits, err := c.fetchProject(ctx, p, since, until, loc)
		if err != nil {
			return nil, fmt.Errorf("gitlab %s: %w", p, err)
		}
		all = append(all, commits...)
	}
	return all, nil
}

func (c *Client) fetchProject(ctx context.Context, project string, since, until time.Time, loc *time.Location) ([]vcs.Commit, error) {
	var out []vcs.Commit
	pid := url.PathEscape(project) // "root/repo" -> "root%2Frepo"

	for page := 1; ; page++ {
		u := fmt.Sprintf(
			"%s/api/v4/projects/%s/repository/commits?since=%s&until=%s&per_page=%d&page=%d&all=true",
			c.base, pid,
			since.UTC().Format(time.RFC3339), until.UTC().Format(time.RFC3339),
			perPage, page,
		)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("PRIVATE-TOKEN", c.token)

		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("%d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}

		var batch []apiCommit
		if err := json.Unmarshal(body, &batch); err != nil {
			return nil, err
		}
		for _, ac := range batch {
			if !c.mine(ac) {
				continue
			}
			msg := ac.Message
			if msg == "" {
				msg = ac.Title
			}
			if cm, ok := vcs.New(ac.ID, msg, ac.CommittedAt.In(loc), project); ok {
				out = append(out, cm)
			}
		}
		if len(batch) < perPage {
			break
		}
	}
	return out, nil
}

func (c *Client) mine(ac apiCommit) bool {
	name, mail := strings.ToLower(ac.AuthorName), strings.ToLower(ac.AuthorEmail)
	for _, m := range c.mails {
		if m != "" && m == mail {
			return true
		}
	}
	for _, n := range c.names {
		if n != "" && n == name {
			return true
		}
	}
	return false
}

func lower(ss []string) []string {
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		out = append(out, strings.ToLower(strings.TrimSpace(s)))
	}
	return out
}
