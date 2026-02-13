package tui

import "testing"

func TestOperationString(t *testing.T) {
	tests := []struct {
		op   Operation
		want string
	}{
		{OpNone, ""},
		{OpInstall, "Install"},
		{OpUpdate, "Update"},
		{OpClean, "Clean"},
		{OpUninstall, "Uninstall"},
		{Operation(99), ""},
	}
	for _, tt := range tests {
		if got := tt.op.String(); got != tt.want {
			t.Errorf("Operation(%d).String() = %q, want %q", tt.op, got, tt.want)
		}
	}
}

func TestPluginStatusIsInstalled(t *testing.T) {
	tests := []struct {
		status PluginStatus
		want   bool
	}{
		{StatusInstalled, true},
		{StatusChecking, true},
		{StatusOutdated, true},
		{StatusCheckFailed, true},
		{StatusNotInstalled, false},
	}
	for _, tt := range tests {
		if got := tt.status.IsInstalled(); got != tt.want {
			t.Errorf("%s.IsInstalled() = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestPluginStatusString(t *testing.T) {
	tests := []struct {
		status PluginStatus
		want   string
	}{
		{StatusInstalled, "Installed"},
		{StatusNotInstalled, "Not Installed"},
		{StatusChecking, "Checking"},
		{StatusOutdated, "Outdated"},
		{StatusCheckFailed, "Check Failed"},
		{PluginStatus(99), "Unknown"},
	}
	for _, tt := range tests {
		if got := tt.status.String(); got != tt.want {
			t.Errorf("PluginStatus(%d).String() = %q, want %q", tt.status, got, tt.want)
		}
	}
}
