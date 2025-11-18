package main

import "testing"

func TestParseFormatFlag_Valid(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		want   string
	}{
		{"auto", "auto", "auto"},
		{"empty treated as auto", "", "auto"},
		{"json", "json", "json"},
		{"yaml", "yaml", "yaml"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseFormatFlag(tc.input)
			if err != nil {
				t.Fatalf("parseFormatFlag(%q) returned error: %v", tc.input, err)
			}
			if string(got) != tc.want {
				t.Fatalf("parseFormatFlag(%q) = %q, want %q", tc.input, string(got), tc.want)
			}
		})
	}
}

func TestParseFormatFlag_Invalid(t *testing.T) {
	if _, err := parseFormatFlag("bogus"); err == nil {
		t.Fatalf("expected error for invalid format, got nil")
	}
}
