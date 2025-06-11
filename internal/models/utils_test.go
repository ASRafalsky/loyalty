package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIDHash(t *testing.T) {
	tt := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"ordinary",
			"123456789012345678901234567890",
			"",
		},
		{
			"short",
			"1",
			"",
		},
		{
			"long",
			"11111111111111112222222222233333333334444444444555555555555566666666666777777777778888888",
			"",
		},
		{
			"empty",
			"",
			"",
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			res, err := IDHash(tc.input)
			require.NoError(t, err)
			t.Log(tc.input, res, len(res))
		})
	}
}
