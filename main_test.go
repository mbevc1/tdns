package main

import "testing"

func TestBuildVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		commit  string
		date    string
		builtBy string
		want    string
	}{
		{
			name:    "version only",
			version: "1.2.3",
			want:    "1.2.3",
		},
		{
			name:    "version and commit",
			version: "1.2.3",
			commit:  "abc123",
			want:    "1.2.3\ncommit: abc123",
		},
		{
			name:    "version and date",
			version: "1.2.3",
			date:    "2026-01-01",
			want:    "1.2.3\nbuilt at: 2026-01-01",
		},
		{
			name:    "version and builtBy",
			version: "1.2.3",
			builtBy: "goreleaser",
			want:    "1.2.3\nbuilt by: goreleaser",
		},
		{
			name:    "all fields",
			version: "1.2.3",
			commit:  "abc123",
			date:    "2026-01-01",
			builtBy: "goreleaser",
			want:    "1.2.3\ncommit: abc123\nbuilt at: 2026-01-01\nbuilt by: goreleaser",
		},
		{
			name:    "dev default",
			version: "dev",
			want:    "dev",
		},
		{
			name:    "empty version with commit",
			version: "",
			commit:  "abc123",
			want:    "\ncommit: abc123",
		},
		{
			name:    "commit and date no builtBy",
			version: "1.0.0",
			commit:  "deadbeef",
			date:    "2026-05-02",
			want:    "1.0.0\ncommit: deadbeef\nbuilt at: 2026-05-02",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildVersion(tt.version, tt.commit, tt.date, tt.builtBy)
			if got != tt.want {
				t.Errorf("buildVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildVersionOrdering(t *testing.T) {
	got := buildVersion("v1", "c1", "d1", "b1")
	want := "v1\ncommit: c1\nbuilt at: d1\nbuilt by: b1"
	if got != want {
		t.Errorf("buildVersion() ordering wrong:\ngot  %q\nwant %q", got, want)
	}
}
