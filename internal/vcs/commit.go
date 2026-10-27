// Package vcs mendefinisikan tipe commit yang netral terhadap sumber
// (GitHub / GitLab) beserta util normalisasi bersama.
package vcs

import (
	"regexp"
	"strings"
	"time"
)

// ticketRe menangkap nomor ticket: NC-123, CUS-45, US-7 (case-insensitive).
var ticketRe = regexp.MustCompile(`(?i)\b((?:NC|CUS|US)-\d+)\b`)

var noisePrefixes = []string{
	"merge branch", "merge pull request", "merge remote",
	"wip", "revert", "bump version",
}

// Commit adalah satu commit yang sudah dinormalkan dari sumber mana pun.
type Commit struct {
	SHA     string
	When    time.Time // dalam timezone lokal
	Subject string    // baris pertama commit message
	Ticket  string    // NC-xxx / CUS-xxx / US-xx (boleh kosong)
	Repo    string    // "owner/repo" atau "group/project" — untuk debug/log
}

// FirstLine mengambil baris pertama pesan commit.
func FirstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

// IsNoise menandai subject yang tidak perlu masuk timesheet.
func IsNoise(subject string) bool {
	low := strings.ToLower(subject)
	for _, p := range noisePrefixes {
		if strings.HasPrefix(low, p) {
			return true
		}
	}
	return false
}

// Ticket mengekstrak nomor ticket dari subject (uppercased), atau "".
func Ticket(subject string) string {
	return strings.ToUpper(ticketRe.FindString(subject))
}

// New membuat Commit ternormalisasi; mengembalikan (Commit{}, false) untuk noise.
func New(sha, message string, when time.Time, repo string) (Commit, bool) {
	subject := FirstLine(message)
	if subject == "" || IsNoise(subject) {
		return Commit{}, false
	}
	return Commit{
		SHA:     sha,
		When:    when,
		Subject: subject,
		Ticket:  Ticket(subject),
		Repo:    repo,
	}, true
}
