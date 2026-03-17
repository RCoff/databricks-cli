package mutator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupBundleForRandomStrings(t *testing.T, randomStrings map[string]config.RandomString) *bundle.Bundle {
	t.Helper()
	tmpDir := t.TempDir()

	b := &bundle.Bundle{
		BundleRootPath: tmpDir,
		Config: config.Root{
			Bundle: config.Bundle{
				Target: "default",
			},
			RandomStrings: randomStrings,
		},
	}

	return b
}

func TestGenerateRandomStringsBasic(t *testing.T) {
	f := false
	b := setupBundleForRandomStrings(t, map[string]config.RandomString{
		"suffix": {
			Length:  8,
			Special: &f,
		},
	})

	ctx := t.Context()
	diags := bundle.Apply(ctx, b, GenerateRandomStrings())
	require.False(t, diags.HasError(), diags)

	rs := b.Config.RandomStrings["suffix"]
	assert.Len(t, rs.Value, 8)

	// Verify no special characters.
	for _, c := range rs.Value {
		assert.False(t, isSpecialChar(c), "unexpected special char: %c", c)
	}
}

func TestGenerateRandomStringsAllOptions(t *testing.T) {
	tr := true
	b := setupBundleForRandomStrings(t, map[string]config.RandomString{
		"test": {
			Length:   20,
			Special:  &tr,
			Upper:    &tr,
			Lower:    &tr,
			Numeric:  &tr,
			MinUpper: 3,
			MinLower: 3,
		},
	})

	ctx := t.Context()
	diags := bundle.Apply(ctx, b, GenerateRandomStrings())
	require.False(t, diags.HasError(), diags)
	assert.Len(t, b.Config.RandomStrings["test"].Value, 20)
}

func TestGenerateRandomStringsPersistence(t *testing.T) {
	f := false
	b := setupBundleForRandomStrings(t, map[string]config.RandomString{
		"suffix": {
			Length:  8,
			Special: &f,
		},
	})

	ctx := t.Context()

	// First run: generate.
	diags := bundle.Apply(ctx, b, GenerateRandomStrings())
	require.False(t, diags.HasError(), diags)
	firstValue := b.Config.RandomStrings["suffix"].Value
	require.Len(t, firstValue, 8)

	// Verify state file was written.
	statePath := filepath.Join(b.BundleRootPath, ".databricks", "bundle", "default", randomStringsStateFile)
	_, err := os.Stat(statePath)
	require.NoError(t, err)

	// Second run: should reuse the same value.
	b2 := setupBundleForRandomStrings(t, map[string]config.RandomString{
		"suffix": {
			Length:  8,
			Special: &f,
		},
	})
	b2.BundleRootPath = b.BundleRootPath

	diags = bundle.Apply(ctx, b2, GenerateRandomStrings())
	require.False(t, diags.HasError(), diags)
	assert.Equal(t, firstValue, b2.Config.RandomStrings["suffix"].Value)
}

func TestGenerateRandomStringsRegeneratesOnConfigChange(t *testing.T) {
	f := false
	b := setupBundleForRandomStrings(t, map[string]config.RandomString{
		"suffix": {
			Length:  8,
			Special: &f,
		},
	})

	ctx := t.Context()

	diags := bundle.Apply(ctx, b, GenerateRandomStrings())
	require.False(t, diags.HasError(), diags)
	firstValue := b.Config.RandomStrings["suffix"].Value

	// Change the length and run again.
	b2 := setupBundleForRandomStrings(t, map[string]config.RandomString{
		"suffix": {
			Length:  16,
			Special: &f,
		},
	})
	b2.BundleRootPath = b.BundleRootPath

	diags = bundle.Apply(ctx, b2, GenerateRandomStrings())
	require.False(t, diags.HasError(), diags)
	assert.Len(t, b2.Config.RandomStrings["suffix"].Value, 16)
	assert.NotEqual(t, firstValue, b2.Config.RandomStrings["suffix"].Value)
}

func TestGenerateRandomStringsValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		rs   config.RandomString
		msg  string
	}{
		{
			name: "zero length",
			rs:   config.RandomString{Length: 0},
			msg:  "length must be a positive integer",
		},
		{
			name: "negative length",
			rs:   config.RandomString{Length: -1},
			msg:  "length must be a positive integer",
		},
		{
			name: "min exceeds length",
			rs: config.RandomString{
				Length:   5,
				MinUpper: 3,
				MinLower: 3,
			},
			msg: "exceeds length",
		},
		{
			name: "no charset enabled",
			rs: func() config.RandomString {
				f := false
				return config.RandomString{
					Length:  8,
					Special: &f,
					Upper:   &f,
					Lower:   &f,
					Numeric: &f,
				}
			}(),
			msg: "at least one character set must be enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := setupBundleForRandomStrings(t, map[string]config.RandomString{
				"test": tt.rs,
			})

			ctx := t.Context()
			diags := bundle.Apply(ctx, b, GenerateRandomStrings())
			require.True(t, diags.HasError())
			assert.Contains(t, diags.Error().Error(), tt.msg)
		})
	}
}

func TestGenerateRandomStringsNoRandomStrings(t *testing.T) {
	b := setupBundleForRandomStrings(t, nil)

	ctx := t.Context()
	diags := bundle.Apply(ctx, b, GenerateRandomStrings())
	require.False(t, diags.HasError(), diags)
}

func TestGenerateRandomStringsOnlyNumeric(t *testing.T) {
	f := false
	b := setupBundleForRandomStrings(t, map[string]config.RandomString{
		"pin": {
			Length:  6,
			Special: &f,
			Upper:   &f,
			Lower:   &f,
		},
	})

	ctx := t.Context()
	diags := bundle.Apply(ctx, b, GenerateRandomStrings())
	require.False(t, diags.HasError(), diags)

	value := b.Config.RandomStrings["pin"].Value
	assert.Len(t, value, 6)
	for _, c := range value {
		assert.True(t, c >= '0' && c <= '9', "expected digit, got: %c", c)
	}
}

func TestGenerateRandomStringsMinConstraints(t *testing.T) {
	f := false
	tr := true
	b := setupBundleForRandomStrings(t, map[string]config.RandomString{
		"test": {
			Length:     10,
			Special:    &f,
			Upper:      &tr,
			Lower:      &tr,
			Numeric:    &tr,
			MinUpper:   3,
			MinLower:   3,
			MinNumeric: 3,
		},
	})

	ctx := t.Context()
	diags := bundle.Apply(ctx, b, GenerateRandomStrings())
	require.False(t, diags.HasError(), diags)

	value := b.Config.RandomStrings["test"].Value
	assert.Len(t, value, 10)

	var uppers, lowers, digits int
	for _, c := range value {
		switch {
		case c >= 'A' && c <= 'Z':
			uppers++
		case c >= 'a' && c <= 'z':
			lowers++
		case c >= '0' && c <= '9':
			digits++
		}
	}

	assert.GreaterOrEqual(t, uppers, 3)
	assert.GreaterOrEqual(t, lowers, 3)
	assert.GreaterOrEqual(t, digits, 3)
}

func TestGenerateRandomStringsStaleEntriesRemoved(t *testing.T) {
	f := false
	b := setupBundleForRandomStrings(t, map[string]config.RandomString{
		"keep":   {Length: 8, Special: &f},
		"remove": {Length: 8, Special: &f},
	})

	ctx := t.Context()
	diags := bundle.Apply(ctx, b, GenerateRandomStrings())
	require.False(t, diags.HasError(), diags)

	// Now run with only "keep".
	b2 := setupBundleForRandomStrings(t, map[string]config.RandomString{
		"keep": {Length: 8, Special: &f},
	})
	b2.BundleRootPath = b.BundleRootPath

	diags = bundle.Apply(ctx, b2, GenerateRandomStrings())
	require.False(t, diags.HasError(), diags)

	// Verify state file only has "keep".
	statePath := filepath.Join(b.BundleRootPath, ".databricks", "bundle", "default", randomStringsStateFile)
	data, err := os.ReadFile(statePath)
	require.NoError(t, err)

	var state randomStringsState
	require.NoError(t, json.Unmarshal(data, &state))
	assert.Contains(t, state.Values, "keep")
	assert.NotContains(t, state.Values, "remove")
}

func isSpecialChar(c rune) bool {
	rs := &config.RandomString{}
	for _, s := range rs.SpecialChars() {
		if c == s {
			return true
		}
	}
	return false
}
