// internal/nodeid/parser_test.go
package nodeid

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	testCases := []struct {
		name         string
		rawID        string
		expectErr    bool
		expectedAddr *Address
	}{
		{
			name:      "simple path",
			rawID:     "a.b.c",
			expectErr: false,
			expectedAddr: &Address{
				Path: []PathSegment{NewPathSegment("a"), NewPathSegment("b"), NewPathSegment("c")},
			},
		},
		{
			name:      "multi-level path with index",
			rawID:     "db.users[0].posts[15]",
			expectErr: false,
			expectedAddr: &Address{
				Path: []PathSegment{NewPathSegment("db"), NewPathSegmentWithIndex("users", 0), NewPathSegmentWithIndex("posts", 15)},
			},
		},
		{
			name:      "zero index",
			rawID:     "http.request[0]",
			expectErr: false,
			expectedAddr: &Address{
				Path: []PathSegment{NewPathSegment("http"), NewPathSegmentWithIndex("request", 0)},
			},
		},
		{
			name:      "error - empty path segment",
			rawID:     "a..b",
			expectErr: true,
		},
		{
			name:      "error - invalid segment format",
			rawID:     "a.b[x]",
			expectErr: true,
		},
		{
			name:      "error - empty string",
			rawID:     "",
			expectErr: true,
		},
		{
			name:      "error - invalid segment name hyphen",
			rawID:     "a.b.-.c",
			expectErr: true,
		},
		{
			name:      "error - invalid segment name just hyphen",
			rawID:     "-",
			expectErr: true,
		},
		{
			name:      "error - invalid segment name just dot",
			rawID:     ".",
			expectErr: true,
		},
		{
			name:      "error - invalid segment name just double dot",
			rawID:     "..",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			addr, err := Parse(tc.rawID)

			if tc.expectErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, addr)
			assert.True(t, tc.expectedAddr.Equal(addr), "Parsed address does not match expected address")
		})
	}
}
