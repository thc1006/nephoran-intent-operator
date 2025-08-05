package internal

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUrlnorm(t *testing.T) {
	testCases := []struct {
		url      string
		expected string
	}{
		{
			url:      "HTTPs://xYz.cOm:443/folder//file",
			expected: "https://xyz.com/folder/file",
		},
		{
			url:      "HTTP://xYz.cOm:80/folder//file",
			expected: "http://xyz.com/folder/file",
		},
		{
			url:      "postGRES://xYz.cOm:5432/folder//file",
			expected: "postgres://xyz.com:5432/folder/file",
		},
	}

	for _, toPin := range testCases {
		testCase := toPin

		u, err := url.Parse(testCase.url)
		require.NoError(t, err)

		NormalizeURL(u)
		assert.Equal(t, testCase.expected, u.String())
	}
}
