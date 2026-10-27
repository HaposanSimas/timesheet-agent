// Command timesheet menarik commit dari GitHub + GitLab untuk satu periode dan
// menyusunnya menjadi baris timesheet siap-paste ke Google Sheet.
//
//	export GITHUB_TOKEN=ghp_xxx
//	export GITLAB_TOKEN=glpat_xxx
//	go run ./cmd/timesheet -config config.json -since 2026-08-16 -until 2026-09-15
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/HaposanSimas/timesheet-agent/internal/config"
	"github.com/HaposanSimas/timesheet-agent/internal/github"
	"github.com/HaposanSimas/timesheet-agent/internal/gitlab"
	"github.com/HaposanSimas/timesheet-agent/internal/timesheet"
	"github.com/HaposanSimas/timesheet-agent/internal/vcs"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		cfgPath = flag.String("config", "config.json", "path file config JSON")
		month   = flag.String("month", "", "periode YYYY-MM (opsional)")
		since   = flag.String("since", "", "tanggal mulai YYYY-MM-DD (override -month)")
		until   = flag.String("until", "", "tanggal akhir YYYY-MM-DD (override -month)")
		format  = flag.String("format", "tsv", "format output: tsv | md")
		standup = flag.Bool("standup", true, "baris Daily Stand Up tiap hari kerja")
		qa      = flag.Bool("qa", true, "baris Stand By QA Testing tiap hari kerja")
	)
	flag.Parse()

	if *month == "" && *since == "" && *until == "" {
		return fmt.Errorf("tentukan periode: -since/-until atau -month")
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return fmt.Errorf("timezone: %w", err)
	}
	start, end, err := resolveRange(*month, *since, *until, loc)
	if err != nil {
		return err
	}
	untilExclusive := end.AddDate(0, 0, 1)

	ctx := context.Background()
	var commits []vcs.Commit

	// --- GitHub ---
	if len(cfg.GitHub.Repos) > 0 {
		token := os.Getenv("GITHUB_TOKEN")
		if token == "" {
			return fmt.Errorf("GITHUB_TOKEN belum di-set (ada %d repo GitHub di config)", len(cfg.GitHub.Repos))
		}
		gh := github.New(token)
		cs, err := gh.FetchAll(ctx, cfg.GitHub.Owner, cfg.GitHub.Repos, cfg.Author.GitHubLogin, start, untilExclusive, loc)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "GitHub : %d commit dari %d repo\n", len(cs), len(cfg.GitHub.Repos))
		commits = append(commits, cs...)
	}

	// --- GitLab ---
	if len(cfg.GitLab.Projects) > 0 {
		token := os.Getenv("GITLAB_TOKEN")
		if token == "" {
			return fmt.Errorf("GITLAB_TOKEN belum di-set (ada %d project GitLab di config)", len(cfg.GitLab.Projects))
		}
		gl := gitlab.New(cfg.GitLab.BaseURL, token, cfg.Author.Names, cfg.Author.Emails, cfg.GitLab.InsecureTLS)
		cs, err := gl.FetchAll(ctx, cfg.GitLab.Projects, start, untilExclusive, loc)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "GitLab : %d commit dari %d project\n", len(cs), len(cfg.GitLab.Projects))
		commits = append(commits, cs...)
	}

	sort.Slice(commits, func(i, j int) bool { return commits[i].When.Before(commits[j].When) })

	opts := timesheet.DefaultOptions()
	opts.Standup, opts.QA = *standup, *qa
	rows := timesheet.Build(commits, start, end, loc, opts)

	if *format == "md" {
		fmt.Print(timesheet.RenderMarkdown(rows))
	} else {
		fmt.Print(timesheet.RenderTSV(rows))
	}

	fmt.Fprintf(os.Stderr, "\nTotal %d commit -> %d baris (%s s/d %s)\n",
		len(commits), len(rows), timesheet.FormatTanggal(start), timesheet.FormatTanggal(end))
	fmt.Fprintln(os.Stderr, "Lengkapi meeting/cuti/standby manual sebelum submit.")
	return nil
}

func resolveRange(month, since, until string, loc *time.Location) (time.Time, time.Time, error) {
	if since != "" || until != "" {
		s, err := time.ParseInLocation("2006-01-02", since, loc)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("-since invalid: %w", err)
		}
		u, err := time.ParseInLocation("2006-01-02", until, loc)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("-until invalid: %w", err)
		}
		return s, u, nil
	}
	m, err := time.ParseInLocation("2006-01", month, loc)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("-month invalid: %w", err)
	}
	first := time.Date(m.Year(), m.Month(), 1, 0, 0, 0, 0, loc)
	return first, first.AddDate(0, 1, -1), nil
}
