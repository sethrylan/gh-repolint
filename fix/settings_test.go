package fix

import (
	"context"
	"testing"

	"github.com/sethrylan/gh-repolint/checks"
	"github.com/sethrylan/gh-repolint/config"
)

func TestSettingsFixer_Fix_NilMergeConfig(t *testing.T) {
	// Test that merge-related settings return an error when Merge config is nil
	// instead of causing a nil pointer dereference panic.

	mergeSettings := []string{
		"merge_commit",
		"squash_merge",
		"rebase_merge",
		"auto_merge",
		"delete_branch_on_merge",
		"update_branch",
	}

	for _, setting := range mergeSettings {
		t.Run(setting, func(t *testing.T) {
			// Create fixer with nil Merge config
			cfg := &config.SettingsConfig{
				Merge: nil, // This would cause a panic without the nil check
			}
			fixer := NewSettingsFixer(nil, cfg, false)

			issue := checks.Issue{
				Type:    checks.CheckTypeSettings,
				Name:    "settings",
				Message: "test issue",
				Fixable: true,
				Data: map[string]string{
					checks.DataKeySetting: setting,
				},
			}

			// This should not panic
			result, err := fixer.Fix(context.Background(), issue)

			// Should return a result with error, not a Go error
			if err != nil {
				t.Fatalf("Fix() returned unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("Fix() returned nil result")
			}

			if result.Fixed {
				t.Error("Fix() should not have marked issue as fixed")
			}

			if result.Error == nil {
				t.Error("Fix() should have returned an error in result")
			}

			expectedMsg := "merge settings not configured"
			if result.Error.Error() != expectedMsg {
				t.Errorf("Fix() error = %q, want %q", result.Error.Error(), expectedMsg)
			}
		})
	}
}

func TestSettingsFixer_Fix_MissingSettingData(t *testing.T) {
	cfg := &config.SettingsConfig{}
	fixer := NewSettingsFixer(nil, cfg, false)

	issue := checks.Issue{
		Type:    checks.CheckTypeSettings,
		Name:    "settings",
		Message: "test issue",
		Fixable: true,
		Data:    map[string]string{}, // Missing "setting" key
	}

	result, err := fixer.Fix(context.Background(), issue)

	if err != nil {
		t.Fatalf("Fix() returned unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Fix() returned nil result")
	}

	if result.Fixed {
		t.Error("Fix() should not have marked issue as fixed")
	}

	expectedMsg := "issue data missing setting"
	if result.Error == nil || result.Error.Error() != expectedMsg {
		t.Errorf("Fix() error = %v, want %q", result.Error, expectedMsg)
	}
}

func TestSettingsFixer_Fix_UnknownSetting(t *testing.T) {
	cfg := &config.SettingsConfig{}
	fixer := NewSettingsFixer(nil, cfg, false)

	issue := checks.Issue{
		Type:    checks.CheckTypeSettings,
		Name:    "settings",
		Message: "test issue",
		Fixable: true,
		Data: map[string]string{
			checks.DataKeySetting: "nonexistent_setting",
		},
	}

	result, err := fixer.Fix(context.Background(), issue)

	if err != nil {
		t.Fatalf("Fix() returned unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Fix() returned nil result")
	}

	if result.Fixed {
		t.Error("Fix() should not have marked issue as fixed")
	}

	if result.Error == nil {
		t.Error("Fix() should have returned an error in result")
	}

	expectedMsg := "unknown setting: nonexistent_setting"
	if result.Error.Error() != expectedMsg {
		t.Errorf("Fix() error = %q, want %q", result.Error.Error(), expectedMsg)
	}
}
