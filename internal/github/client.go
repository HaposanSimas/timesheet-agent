// Package github mengambil commit dari GitHub REST API (mendukung repo private).
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/HaposanSimas/timesheet-agent/internal/vcs"
)

const (
	apiBase = "https://api.github.com"
	perPage = 100
)

// Client adalah klien GitHub minimal (stdlib saja).
type Client struct {
	token string
	http  *http.Client
}

// New membuat Client dengan personal access token (scope: repo read).
func New(token string) *Client {
	return &Client{token: token, http: &http.Client{Timeout: 30 * time.Second}}
}

type apiCommit struct {
	SHA    string `json:"sha"`
	Commit struct {
		Message string `json:"message"`
		Author  struct {
			Date time.Time `json:"date"`
		} `json:"author"`
	} `json:"commit"`
}

// FetchAll menarik commit ` + "`author`" + ` dari banyak repo (semua di bawah owner
// yang sama) pada rentang [since, until), lalu menggabungkannya.
func (c *Client) FetchAll(ctx context.Context, owner string, repos []string, author string, since, until time.Time, loc *time.Location) ([]vcs.Commit, error) {
	var all []vcs.Commit
	for _, repo := range repos {
		commits, err := c.fetchRepo(ctx, owner, repo, author, since, until, loc)
		if err != nil {
			return nil, fmt.Errorf("github %s/%s: %w", owner, repo, err)
		}
		all = append(all, commits...)
	}
	return all, nil
}

func (c *Client) fetchRepo(ctx context.Context, owner, repo, author string, since, until time.Time, loc *time.Location) ([]vcs.Commit, error) {
	var out []vcs.Commit
	full := owner + "/" + repo

	for page := 1; ; page++ {
		url := fmt.Sprintf(
			"%s/repos/%s/%s/commits?author=%s&since=%s&until=%s&per_page=%d&page=%d",
			apiBase, owner, repo, author,
			since.UTC().Format(time.RFC3339), until.UTC().Format(time.RFC3339),
			perPage, page,
		)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

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
			if cm, ok := vcs.New(ac.SHA, ac.Commit.Message, ac.Commit.Author.Date.In(loc), full); ok {
				out = append(out, cm)
			}
		}
		if len(batch) < perPage {
			break
		}
	}
	return out, nil
}
