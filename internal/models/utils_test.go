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

func TestCheckCreds(t *testing.T) {
	tt := []struct {
		name   string
		creds  Credential
		errStr string
	}{
		{
			name: "valid",
			creds: Credential{
				Login:    "Ivan",
				Password: "qwerty",
			},
		},
		{
			name: "empty_login",
			creds: Credential{
				Password: "qwerty",
			},
			errStr: "required",
		},
		{
			name: "empty_password",
			creds: Credential{
				Login: "Ivan",
			},
			errStr: "required",
		},
		{
			name:   "empty_everything",
			errStr: "required",
		},
		{
			name: "too_short_login",
			creds: Credential{
				Login:    "I",
				Password: "qwerty",
			},
			errStr: "min",
		},
		{
			name: "too_short_password",
			creds: Credential{
				Login:    "Ivan",
				Password: "q",
			},
			errStr: "min",
		},
		{
			name: "too_long_password",
			creds: Credential{
				Login:    "Ivan",
				Password: "qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq",
			},
			errStr: "max",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckCreds(&tc.creds)
			if tc.errStr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errStr)
			}
		})
	}
}
