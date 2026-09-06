package views

import (
	"reflect"
	"testing"

	"github.com/glutechnologies/vpptop/gui/xtui"
)

func TestTableViewResizeWithFixedColumnWidths(t *testing.T) {
	want := []int{22, 11, 7, 5, 6, 4, 6, 8}
	view := NewTableView(nil, xtui.TableRows{{"Name", "Worker"}}, 0, 1, append([]int(nil), want...), false)

	view.Resize(80, 24)

	if !reflect.DeepEqual(view.table.Table.ColumnWidths, want) {
		t.Fatalf("table column widths = %v, want %v", view.table.Table.ColumnWidths, want)
	}
	if !reflect.DeepEqual(view.header.Table.ColumnWidths, want) {
		t.Fatalf("header column widths = %v, want %v", view.header.Table.ColumnWidths, want)
	}
}
