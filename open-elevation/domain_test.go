package openelevation

import "testing"

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "openelevation" {
		t.Errorf("Scheme = %q, want openelevation", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "elevation" {
		t.Errorf("Identity.Binary = %q, want elevation", info.Identity.Binary)
	}
}

func TestClassify(t *testing.T) {
	typ, id, err := Domain{}.Classify("41.161758,-8.583933")
	if err != nil {
		t.Fatalf("Classify error: %v", err)
	}
	if typ != "point" {
		t.Errorf("typ = %q, want point", typ)
	}
	if id != "41.161758,-8.583933" {
		t.Errorf("id = %q", id)
	}
}

func TestClassifyEmpty(t *testing.T) {
	_, _, err := Domain{}.Classify("")
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestClassifyInvalid(t *testing.T) {
	_, _, err := Domain{}.Classify("notacoord")
	if err == nil {
		t.Error("expected error for non-coordinate input")
	}
}

func TestLocate(t *testing.T) {
	got, err := Domain{}.Locate("point", "41.161758,-8.583933")
	if err != nil {
		t.Fatalf("Locate error: %v", err)
	}
	if got == "" {
		t.Error("Locate returned empty URL")
	}
}

func TestLocateUnknownType(t *testing.T) {
	_, err := Domain{}.Locate("unknown", "foo")
	if err == nil {
		t.Error("expected error for unknown resource type")
	}
}

func TestParseCoords(t *testing.T) {
	cases := []struct {
		in   string
		want [][2]float64
	}{
		{"41.161758,-8.583933", [][2]float64{{41.161758, -8.583933}}},
		{"10,10 -20,30", [][2]float64{{10, 10}, {-20, 30}}},
	}
	for _, c := range cases {
		got, err := parseCoords(c.in)
		if err != nil {
			t.Errorf("parseCoords(%q) error: %v", c.in, err)
			continue
		}
		if len(got) != len(c.want) {
			t.Errorf("parseCoords(%q) len = %d, want %d", c.in, len(got), len(c.want))
		}
	}
}
