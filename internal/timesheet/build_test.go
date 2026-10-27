package timesheet

import (
	"strings"
	"testing"
	"time"

	"github.com/HaposanSimas/timesheet-agent/internal/vcs"
)

func TestBuild_PrefixAndRemarkTransitions(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	at := func(day int, subj string) vcs.Commit {
		return vcs.Commit{
			When:    time.Date(2026, 8, day, 10, 0, 0, 0, loc),
			Subject: subj,
			Ticket:  vcs.Ticket(subj),
		}
	}
	commits := []vcs.Commit{
		at(18, "feat(sp): SP automation (NC-570)"),
		at(19, "feat(NC-616): sync grouping totals"),
		at(20, "feat(NC-616): approval hierarchy"),
		at(22, "weekend should be skipped"), // 22 Agu 2026 = Sabtu
	}

	start := time.Date(2026, 8, 18, 0, 0, 0, 0, loc)
	end := time.Date(2026, 8, 23, 0, 0, 0, 0, loc)
	out := RenderMarkdown(Build(commits, start, end, loc, DefaultOptions()))

	checks := []string{
		"18 Agustus 2026",
		"Working on feat(sp): SP automation (NC-570)",
		"Working on feat(NC-616): sync grouping totals |  |  | In Progress",
		"Continue Working on feat(NC-616): approval hierarchy |  |  | Done",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("output tidak memuat %q\n---\n%s", want, out)
		}
	}
	if strings.Contains(out, "weekend should be skipped") {
		t.Errorf("commit akhir pekan seharusnya dilewati\n%s", out)
	}
}
