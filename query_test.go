package orchestra

import (
	"testing"

	"github.com/ghetzel/testify/require"
)

func TestQueryMerge(t *testing.T) {
	var assert = require.New(t)

	var parent = &QueryOptions{
		Context: Context{
			Variables: map[string]any{
				`hello`: `there`,
			},
			Headers: map[string]any{
				`X-Cool-Test`: `1`,
			},
		},
	}

	var child = &QueryOptions{
		Context: Context{
			Variables: map[string]any{
				`hello`: `you`,
				`whats`: `up`,
			},
			Headers: map[string]any{
				`X-Cool-Other`: `1`,
			},
		},
	}

	var result, err = parent.Merge(child)

	assert.NoError(err)
	assert.EqualValues(map[string]any{
		`hello`: `you`,
		`whats`: `up`,
	}, result.Variables)

	assert.EqualValues(map[string]any{
		`X-Cool-Test`:  `1`,
		`X-Cool-Other`: `1`,
	}, result.Headers)
}
