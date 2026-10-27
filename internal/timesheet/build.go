// Package timesheet menyusun commit menjadi baris timesheet sesuai format
// Google Sheet: DATE | Task/Activity | START TIME | END TIME | REMARK.
package timesheet

import (
	"fmt"
	"strings"
	"time"

	"github.com/HaposanSimas/timesheet-agent/internal/vcs"
)

// Nilai default yang menyesuaikan pola sheet.
const (
	DefaultStandup = "Daily Stand Up - New Collection System"
	DefaultQA      = "Stand By QA Testing"
	DefaultStart   = "08.00"
	DefaultEnd     = "17.00"

	remarkDone       = "Done"
	remarkInProgress = "In Progress"
)

var bulanID = []string{
	"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember",
}

// Row adalah satu baris timesheet.
type Row struct {
	Date   string
	Task   string
	Start  string
	End    string
	Remark string
}

// Options mengatur baris rutin & jam default.
type Options struct {
	Standup     bool
	QA          bool
	StandupText string
	QAText      string
	Start       string
	End         string
}

// DefaultOptions mengembalikan setelan standar.
func DefaultOptions() Options {
	return Options{
		Standup:     true,
		QA:          true,
		StandupText: DefaultStandup,
		QAText:      DefaultQA,
		Start:       DefaultStart,
		End:         DefaultEnd,
	}
}

// Build menyusun baris timesheet untuk rentang [start, end] (inklusif),
// melewati akhir pekan. start & end harus berada di timezone `loc`.
func Build(commits []vcs.Commit, start, end time.Time, loc *time.Location, opts Options) []Row {
	byDay := groupByDay(commits)
	spans := computeSpans(commits)

	var rows []Row
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		if wd := d.Weekday(); wd == time.Saturday || wd == time.Sunday {
			continue
		}
		key := d.Format("2006-01-02")
		dayCommits := byDay[key]
		if len(dayCommits) == 0 && !opts.Standup && !opts.QA {
			continue
		}

		dateStr := FormatTanggal(d)
		firstRow := true
		emit := func(task, remark string) {
			r := Row{Task: task, Remark: remark}
			if firstRow {
				r.Date, r.Start, r.End = dateStr, opts.Start, opts.End
				firstRow = false
			}
			rows = append(rows, r)
		}

		if opts.Standup {
			emit(opts.StandupText, remarkDone)
		}

		seen := map[string]bool{}
		for _, c := range dayCommits {
			if seen[c.Subject] {
				continue
			}
			seen[c.Subject] = true
			task, remark := composeTask(c, key, spans)
			emit(task, remark)
		}

		if opts.QA {
			emit(opts.QAText, remarkDone)
		}
	}
	return rows
}

type span struct{ first, last string }

func keyOf(c vcs.Commit) string {
	if c.Ticket != "" {
		return strings.ToUpper(c.Ticket)
	}
	return "subj:" + strings.ToLower(c.Subject)
}

func groupByDay(commits []vcs.Commit) map[string][]vcs.Commit {
	m := map[string][]vcs.Commit{}
	for _, c := range commits {
		k := c.When.Format("2006-01-02")
		m[k] = append(m[k], c)
	}
	return m
}

// computeSpans mencatat hari pertama & terakhir tiap unit kerja (ticket/subject)
// muncul, untuk menentukan "Working on" vs "Continue Working on" dan
// "Done" vs "In Progress".
func computeSpans(commits []vcs.Commit) map[string]*span {
	spans := map[string]*span{}
	for _, c := range commits {
		day := c.When.Format("2006-01-02")
		k := keyOf(c)
		s := spans[k]
		if s == nil {
			spans[k] = &span{first: day, last: day}
			continue
		}
		if day < s.first {
			s.first = day
		}
		if day > s.last {
			s.last = day
		}
	}
	return spans
}

func composeTask(c vcs.Commit, day string, spans map[string]*span) (task, remark string) {
	s := spans[keyOf(c)]
	prefix := "Working on"
	remark = remarkDone
	if s != nil {
		if day > s.first {
			prefix = "Continue Working on"
		}
		if day < s.last {
			remark = remarkInProgress
		}
	}
	task = prefix + " " + c.Subject
	if c.Ticket != "" && !strings.Contains(c.Subject, c.Ticket) {
		task += " (" + strings.ToUpper(c.Ticket) + ")"
	}
	return task, remark
}

// FormatTanggal mengembalikan tanggal gaya Indonesia, mis. "18 Mei 2026".
func FormatTanggal(t time.Time) string {
	return fmt.Sprintf("%d %s %d", t.Day(), bulanID[int(t.Month())], t.Year())
}
