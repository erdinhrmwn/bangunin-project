package pagination

import "testing"

func TestParse(t *testing.T) {
	cases := []struct {
		name              string
		page, perPage     string
		wantPage, wantPer int
	}{
		{"defaults on empty", "", "", 1, DefaultPerPage},
		{"defaults on invalid", "abc", "xyz", 1, DefaultPerPage},
		{"defaults on non-positive", "0", "-5", 1, DefaultPerPage},
		{"clamps per_page to max", "2", "500", 2, MaxPerPage},
		{"passes through valid values", "3", "50", 3, 50},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Parse(c.page, c.perPage)
			if got.Page != c.wantPage || got.PerPage != c.wantPer {
				t.Errorf("Parse(%q, %q) = %+v, want Page=%d PerPage=%d", c.page, c.perPage, got, c.wantPage, c.wantPer)
			}
		})
	}
}

func TestOffset(t *testing.T) {
	if got := (Params{Page: 1, PerPage: 20}).Offset(); got != 0 {
		t.Errorf("page 1 offset = %d, want 0", got)
	}
	if got := (Params{Page: 3, PerPage: 20}).Offset(); got != 40 {
		t.Errorf("page 3 offset = %d, want 40", got)
	}
}
