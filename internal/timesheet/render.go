package timesheet

import (
	"fmt"
	"strings"
)

const header = "Task / Activity/ Project Name / Ticket no."

// RenderTSV menghasilkan TSV; paste langsung ke Google Sheet -> kolom otomatis.
func RenderTSV(rows []Row) string {
	var b strings.Builder
	fmt.Fprintf(&b, "DATE\t%s\tSTART TIME\tEND TIME\tREMARK\n", header)
	for _, r := range rows {
		fmt.Fprintf(&b, "%s\t%s\t%s\t%s\t%s\n", r.Date, r.Task, r.Start, r.End, r.Remark)
	}
	return b.String()
}

// RenderMarkdown menghasilkan tabel markdown untuk pengecekan visual.
func RenderMarkdown(rows []Row) string {
	var b strings.Builder
	fmt.Fprintf(&b, "| DATE | %s | START TIME | END TIME | REMARK |\n", header)
	b.WriteString("| :-: | :-: | :-: | :-: | :-: |\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n", r.Date, r.Task, r.Start, r.End, r.Remark)
	}
	return b.String()
}
