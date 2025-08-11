// internal/nodeid/address_test.go
package nodeid

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddress_String(t *testing.T) {
	testCases := []struct {
		name        string
		addr        *Address
		expectedStr string
	}{
		{
			name: "simple path",
			addr: &Address{
				Path: []PathSegment{NewPathSegment("a"), NewPathSegment("b")},
			},
			expectedStr: "a.b",
		},
		{
			name: "path with indices",
			addr: &Address{
				Path: []PathSegment{NewPathSegment("db"), NewPathSegmentWithIndex("users", 0), NewPathSegmentWithIndex("posts", 15)},
			},
			expectedStr: "db.users[0].posts[15]",
		},
		{
			name:        "nil address",
			addr:        nil,
			expectedStr: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectedStr, tc.addr.String())
		})
	}
}

func TestAddress_RoundTrip(t *testing.T) {
	testIDs := []string{
		"a.b.c",
		"db.users[0].posts[15]",
		"http-client.get[0]",
	}

	for _, id := range testIDs {
		t.Run(id, func(t *testing.T) {
			addr, err := Parse(id)
			require.NoError(t, err)

			roundTripID := addr.String()
			assert.Equal(t, id, roundTripID)

			roundTripAddr, err := Parse(roundTripID)
			require.NoError(t, err)
			assert.True(t, addr.Equal(roundTripAddr))
		})
	}
}

func TestAddress_Equal(t *testing.T) {
	addr1, _ := Parse("a.b[0]")
	addr2, _ := Parse("a.b[0]")
	addr3, _ := Parse("a.b[1]")
	addr4, _ := Parse("a.c[0]")
	addr5, _ := Parse("a.b[0]")

	assert.True(t, addr1.Equal(addr2))
	assert.False(t, addr1.Equal(addr3))
	assert.False(t, addr1.Equal(addr4))
	assert.True(t, addr1.Equal(addr5))
	assert.False(t, addr1.Equal(nil))
	assert.False(t, (*Address)(nil).Equal(addr1))
	assert.True(t, (*Address)(nil).Equal(nil))
}
