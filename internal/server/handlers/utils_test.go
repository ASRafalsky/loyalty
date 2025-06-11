package handlers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLuhn(t *testing.T) {
	tt := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "valid",
			input: "49927398716",
			want:  true,
		},
		{
			name:  "valid_with_spaces",
			input: "  4992 73  98 716",
			want:  true,
		},
		{
			name:  "invalid",
			input: "49927398717",
			want:  false,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, isValidOrderByLuhn(cleanSpaces(tc.input)))
		})
	}
}
