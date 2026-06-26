package markdowntable

import "testing"

func TestRenderReturnsEmptyStringWithoutHeader(t *testing.T) {
	if got := Render(Table{}); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestRenderLeftAlignedTable(t *testing.T) {
	got := Render(Table{
		Header: []string{"Name", "Type"},
		Rows: [][]string{
			{"`id`", "`int64`"},
			{"`CreatedAt`", "`time.Time`"},
		},
	})
	want := "| Name        | Type        |\n" +
		"| ----------- | ----------- |\n" +
		"| `id`        | `int64`     |\n" +
		"| `CreatedAt` | `time.Time` |\n"
	if got != want {
		t.Fatalf("unexpected table:\n%s", got)
	}
}

func TestRenderAlignments(t *testing.T) {
	got := Render(Table{
		Header: []string{"Name", "Count", "Status"},
		Rows: [][]string{
			{"alpha", "12", "ok"},
			{"b", "3", "pending"},
		},
		Align: []Alignment{AlignLeft, AlignRight, AlignCenter},
	})
	want := "| Name  | Count | Status  |\n" +
		"| ----- | ----: | :-----: |\n" +
		"| alpha |    12 |   ok    |\n" +
		"| b     |     3 | pending |\n"
	if got != want {
		t.Fatalf("unexpected table:\n%s", got)
	}
}

func TestRenderNormalizesRowWidths(t *testing.T) {
	got := Render(Table{
		Header: []string{"A", "B"},
		Rows: [][]string{
			{"one"},
			{"two", "three", "ignored"},
		},
	})
	want := "| A   | B     |\n" +
		"| --- | ----- |\n" +
		"| one |       |\n" +
		"| two | three |\n"
	if got != want {
		t.Fatalf("unexpected table:\n%s", got)
	}
}

func TestRenderEscapesPipesAndFoldsNewlines(t *testing.T) {
	got := Render(Table{
		Header: []string{"Name", "Text"},
		Rows: [][]string{
			{"a|b", "line 1\nline 2"},
		},
	})
	want := "| Name | Text          |\n" +
		"| ---- | ------------- |\n" +
		"| a\\|b | line 1 line 2 |\n"
	if got != want {
		t.Fatalf("unexpected table:\n%s", got)
	}
}

func TestRenderCountsRunes(t *testing.T) {
	got := Render(Table{
		Header: []string{"名称", "值"},
		Rows: [][]string{
			{"服务", "alpha"},
			{"表", "beta"},
		},
	})
	want := "| 名称  | 值     |\n" +
		"| --- | ----- |\n" +
		"| 服务  | alpha |\n" +
		"| 表   | beta  |\n"
	if got != want {
		t.Fatalf("unexpected table:\n%s", got)
	}
}
