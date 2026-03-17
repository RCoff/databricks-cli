package config

// RandomString configures generation of a random string value.
// Values are generated once and persisted in the bundle's local state directory
// so they remain stable across deployments.
//
// Reference: https://registry.terraform.io/providers/hashicorp/random/latest/docs/resources/string
type RandomString struct {
	// Length of the random string.
	Length int `json:"length"`

	// Whether to include special characters. Default: true.
	Special *bool `json:"special,omitempty"`

	// Whether to include uppercase letters. Default: true.
	Upper *bool `json:"upper,omitempty"`

	// Whether to include lowercase letters. Default: true.
	Lower *bool `json:"lower,omitempty"`

	// Whether to include numeric digits. Default: true.
	Numeric *bool `json:"numeric,omitempty"`

	// Minimum number of uppercase letters. Default: 0.
	MinUpper int `json:"min_upper,omitempty"`

	// Minimum number of lowercase letters. Default: 0.
	MinLower int `json:"min_lower,omitempty"`

	// Minimum number of numeric digits. Default: 0.
	MinNumeric int `json:"min_numeric,omitempty"`

	// Minimum number of special characters. Default: 0.
	MinSpecial int `json:"min_special,omitempty"`

	// Custom set of special characters to use. Only used when special is true.
	OverrideSpecial string `json:"override_special,omitempty"`

	// The generated value. This field is read-only and set by the GenerateRandomStrings mutator.
	Value string `json:"value,omitempty" bundle:"readonly"`
}

const defaultSpecialChars = "!@#$%&*()-_=+[]{}<>:?"

// IncludeSpecial returns whether special characters should be included.
func (r *RandomString) IncludeSpecial() bool {
	return r.Special == nil || *r.Special
}

// IncludeUpper returns whether uppercase letters should be included.
func (r *RandomString) IncludeUpper() bool {
	return r.Upper == nil || *r.Upper
}

// IncludeLower returns whether lowercase letters should be included.
func (r *RandomString) IncludeLower() bool {
	return r.Lower == nil || *r.Lower
}

// IncludeNumeric returns whether numeric digits should be included.
func (r *RandomString) IncludeNumeric() bool {
	return r.Numeric == nil || *r.Numeric
}

// SpecialChars returns the set of special characters to use.
func (r *RandomString) SpecialChars() string {
	if r.OverrideSpecial != "" {
		return r.OverrideSpecial
	}
	return defaultSpecialChars
}
