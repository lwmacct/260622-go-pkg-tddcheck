package markdowntable

import (
	"strings"
	"unicode/utf8"
)

type Alignment int

const (
	AlignLeft Alignment = iota
	AlignCenter
	AlignRight
)

type Table struct {
	Header []string
	Rows   [][]string
	Align  []Alignment
}

func Render(table Table) string {
	columns := len(table.Header)
	if columns == 0 {
		return ""
	}

	header := normalizeRow(table.Header, columns)
	rows := normalizeRows(table.Rows, columns)
	alignments := normalizeAlignments(table.Align, columns)
	widths := columnWidths(header, rows)

	var builder strings.Builder
	writeRow(&builder, header, widths, alignments)
	writeSeparator(&builder, widths, alignments)
	for _, row := range rows {
		writeRow(&builder, row, widths, alignments)
	}
	return builder.String()
}

func normalizeRows(rows [][]string, columns int) [][]string {
	values := make([][]string, 0, len(rows))
	for _, row := range rows {
		values = append(values, normalizeRow(row, columns))
	}
	return values
}

func normalizeRow(row []string, columns int) []string {
	values := make([]string, columns)
	for index := range columns {
		if index >= len(row) {
			continue
		}
		values[index] = normalizeCell(row[index])
	}
	return values
}

func normalizeAlignments(alignments []Alignment, columns int) []Alignment {
	values := make([]Alignment, columns)
	for index := range columns {
		if index >= len(alignments) {
			values[index] = AlignLeft
			continue
		}
		switch alignments[index] {
		case AlignCenter, AlignRight:
			values[index] = alignments[index]
		default:
			values[index] = AlignLeft
		}
	}
	return values
}

func normalizeCell(value string) string {
	value = strings.ReplaceAll(value, "\r\n", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "|", "\\|")
	return value
}

func columnWidths(header []string, rows [][]string) []int {
	widths := make([]int, len(header))
	for index, cell := range header {
		widths[index] = max(3, cellWidth(cell))
	}
	for _, row := range rows {
		for index, cell := range row {
			widths[index] = max(widths[index], cellWidth(cell))
		}
	}
	return widths
}

func writeRow(builder *strings.Builder, row []string, widths []int, alignments []Alignment) {
	builder.WriteString("|")
	for index, cell := range row {
		builder.WriteByte(' ')
		builder.WriteString(padCell(cell, widths[index], alignments[index]))
		builder.WriteString(" |")
	}
	builder.WriteByte('\n')
}

func writeSeparator(builder *strings.Builder, widths []int, alignments []Alignment) {
	builder.WriteString("|")
	for index, width := range widths {
		builder.WriteByte(' ')
		builder.WriteString(separator(width, alignments[index]))
		builder.WriteString(" |")
	}
	builder.WriteByte('\n')
}

func separator(width int, alignment Alignment) string {
	width = max(3, width)
	switch alignment {
	case AlignCenter:
		if width == 3 {
			return ":-:"
		}
		return ":" + strings.Repeat("-", width-2) + ":"
	case AlignRight:
		return strings.Repeat("-", width-1) + ":"
	default:
		return strings.Repeat("-", width)
	}
}

func padCell(value string, width int, alignment Alignment) string {
	padding := width - cellWidth(value)
	if padding <= 0 {
		return value
	}
	switch alignment {
	case AlignRight:
		return strings.Repeat(" ", padding) + value
	case AlignCenter:
		left := padding / 2
		right := padding - left
		return strings.Repeat(" ", left) + value + strings.Repeat(" ", right)
	default:
		return value + strings.Repeat(" ", padding)
	}
}

func cellWidth(value string) int {
	return utf8.RuneCountInString(value)
}
