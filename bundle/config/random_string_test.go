package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandomStringDefaults(t *testing.T) {
	rs := RandomString{Length: 8}

	assert.True(t, rs.IncludeSpecial())
	assert.True(t, rs.IncludeUpper())
	assert.True(t, rs.IncludeLower())
	assert.True(t, rs.IncludeNumeric())
	assert.Equal(t, defaultSpecialChars, rs.SpecialChars())
}

func TestRandomStringExplicitFalse(t *testing.T) {
	f := false
	rs := RandomString{
		Length:  8,
		Special: &f,
		Upper:   &f,
		Lower:   &f,
		Numeric: new(bool), // zero value is false
	}

	assert.False(t, rs.IncludeSpecial())
	assert.False(t, rs.IncludeUpper())
	assert.False(t, rs.IncludeLower())
	assert.False(t, rs.IncludeNumeric())
}

func TestRandomStringOverrideSpecial(t *testing.T) {
	rs := RandomString{
		Length:          8,
		OverrideSpecial: "-_",
	}

	assert.Equal(t, "-_", rs.SpecialChars())
}
