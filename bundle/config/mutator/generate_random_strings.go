package mutator

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
)

const randomStringsStateFile = "random_strings.json"

type generateRandomStrings struct{}

// GenerateRandomStrings generates random string values and persists them
// in the bundle's local state directory for stability across deployments.
func GenerateRandomStrings() bundle.Mutator {
	return &generateRandomStrings{}
}

func (m *generateRandomStrings) Name() string {
	return "GenerateRandomStrings"
}

// randomStringsState is the persisted state of random string values.
type randomStringsState struct {
	// Values maps random string names to their generated values.
	Values map[string]string `json:"values"`
	// Configs maps random string names to a hash of their configuration,
	// so we can detect when config changes and regenerate.
	Configs map[string]string `json:"configs"`
}

func (m *generateRandomStrings) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if len(b.Config.RandomStrings) == 0 {
		return nil
	}

	// Validate all random string configurations.
	for name, rs := range b.Config.RandomStrings {
		if diags := validateRandomString(name, rs); diags.HasError() {
			return diags
		}
	}

	// Load existing state.
	state, err := loadRandomStringsState(ctx, b)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to load random strings state: %w", err))
	}

	// Generate or reuse values for each random string.
	changed := false
	for name, rs := range b.Config.RandomStrings {
		configHash := configFingerprint(rs)

		if existing, ok := state.Values[name]; ok && state.Configs[name] == configHash {
			log.Debugf(ctx, "Reusing existing random string %q", name)
			rs.Value = existing
			b.Config.RandomStrings[name] = rs
			continue
		}

		log.Infof(ctx, "Generating random string %q (length=%d)", name, rs.Length)
		value, err := generateRandomString(rs)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to generate random string %q: %w", name, err))
		}

		rs.Value = value
		b.Config.RandomStrings[name] = rs
		state.Values[name] = value
		state.Configs[name] = configHash
		changed = true
	}

	// Remove stale entries from state.
	for name := range state.Values {
		if _, ok := b.Config.RandomStrings[name]; !ok {
			delete(state.Values, name)
			delete(state.Configs, name)
			changed = true
		}
	}

	// Persist state if anything changed.
	if changed {
		if err := saveRandomStringsState(ctx, b, state); err != nil {
			return diag.FromErr(fmt.Errorf("failed to save random strings state: %w", err))
		}
	}

	// Write generated values into the dynamic config so they're available for interpolation.
	return diag.FromErr(b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		for name, rs := range b.Config.RandomStrings {
			p := dyn.NewPath(dyn.Key("random_strings"), dyn.Key(name), dyn.Key("value"))
			root, err = dyn.SetByPath(root, p, dyn.V(rs.Value))
			if err != nil {
				return dyn.InvalidValue, fmt.Errorf("failed to set random string %q value: %w", name, err)
			}
		}
		return root, nil
	}))
}

func validateRandomString(name string, rs config.RandomString) diag.Diagnostics {
	if rs.Length <= 0 {
		return diag.Errorf("random string %q: length must be a positive integer", name)
	}

	minTotal := rs.MinUpper + rs.MinLower + rs.MinNumeric + rs.MinSpecial
	if minTotal > rs.Length {
		return diag.Errorf(
			"random string %q: sum of min_upper (%d) + min_lower (%d) + min_numeric (%d) + min_special (%d) = %d exceeds length (%d)",
			name, rs.MinUpper, rs.MinLower, rs.MinNumeric, rs.MinSpecial, minTotal, rs.Length,
		)
	}

	if !rs.IncludeUpper() && !rs.IncludeLower() && !rs.IncludeNumeric() && !rs.IncludeSpecial() {
		return diag.Errorf("random string %q: at least one character set must be enabled", name)
	}

	return nil
}

// configFingerprint returns a string that uniquely identifies the configuration
// of a random string. If the configuration changes, the value is regenerated.
func configFingerprint(rs config.RandomString) string {
	return fmt.Sprintf("len=%d,special=%v,upper=%v,lower=%v,numeric=%v,min_upper=%d,min_lower=%d,min_numeric=%d,min_special=%d,override_special=%s",
		rs.Length, rs.IncludeSpecial(), rs.IncludeUpper(), rs.IncludeLower(), rs.IncludeNumeric(),
		rs.MinUpper, rs.MinLower, rs.MinNumeric, rs.MinSpecial, rs.OverrideSpecial)
}

func generateRandomString(rs config.RandomString) (string, error) {
	const (
		upperChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		lowerChars = "abcdefghijklmnopqrstuvwxyz"
		digitChars = "0123456789"
	)

	specialChars := rs.SpecialChars()

	// Build the full charset.
	var charset string
	if rs.IncludeUpper() {
		charset += upperChars
	}
	if rs.IncludeLower() {
		charset += lowerChars
	}
	if rs.IncludeNumeric() {
		charset += digitChars
	}
	if rs.IncludeSpecial() {
		charset += specialChars
	}

	result := make([]byte, rs.Length)
	pos := 0

	// Fill minimum required characters first.
	pos, err := fillMin(result, pos, upperChars, rs.MinUpper)
	if err != nil {
		return "", err
	}
	pos, err = fillMin(result, pos, lowerChars, rs.MinLower)
	if err != nil {
		return "", err
	}
	pos, err = fillMin(result, pos, digitChars, rs.MinNumeric)
	if err != nil {
		return "", err
	}
	pos, err = fillMin(result, pos, specialChars, rs.MinSpecial)
	if err != nil {
		return "", err
	}

	// Fill the rest from the full charset.
	for i := pos; i < rs.Length; i++ {
		idx, err := cryptoRandIntn(len(charset))
		if err != nil {
			return "", err
		}
		result[i] = charset[idx]
	}

	// Shuffle to avoid predictable ordering (minimums at the start).
	if err := cryptoShuffle(result); err != nil {
		return "", err
	}

	return string(result), nil
}

// fillMin fills the result buffer with n random characters from the given charset.
func fillMin(result []byte, pos int, charset string, n int) (int, error) {
	for range n {
		idx, err := cryptoRandIntn(len(charset))
		if err != nil {
			return pos, err
		}
		result[pos] = charset[idx]
		pos++
	}
	return pos, nil
}

// cryptoRandIntn returns a cryptographically random int in [0, n).
func cryptoRandIntn(n int) (int, error) {
	max := big.NewInt(int64(n))
	v, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0, err
	}
	return int(v.Int64()), nil
}

// cryptoShuffle performs a Fisher-Yates shuffle using crypto/rand.
func cryptoShuffle(b []byte) error {
	for i := len(b) - 1; i > 0; i-- {
		j, err := cryptoRandIntn(i + 1)
		if err != nil {
			return err
		}
		b[i], b[j] = b[j], b[i]
	}
	return nil
}

func randomStringsStatePath(ctx context.Context, b *bundle.Bundle) string {
	return filepath.Join(b.GetLocalStateDir(ctx), randomStringsStateFile)
}

func loadRandomStringsState(ctx context.Context, b *bundle.Bundle) (*randomStringsState, error) {
	state := &randomStringsState{
		Values:  make(map[string]string),
		Configs: make(map[string]string),
	}

	path := randomStringsStatePath(ctx, b)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return state, nil
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, state); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	if state.Values == nil {
		state.Values = make(map[string]string)
	}
	if state.Configs == nil {
		state.Configs = make(map[string]string)
	}

	return state, nil
}

func saveRandomStringsState(ctx context.Context, b *bundle.Bundle, state *randomStringsState) error {
	dir, err := b.LocalStateDir(ctx)
	if err != nil {
		return err
	}

	// Go's json.Marshal sorts map keys, so output is deterministic.
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(dir, randomStringsStateFile)
	return os.WriteFile(path, data, 0o600)
}
