package orchestra

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/ghetzel/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

func TestGQLRender(t *testing.T) {
	var assert = require.New(t)

	// expected GraphQL query from GraphQLQuery.Render.
	// make sure it's syntax is correct by trying to parse it
	var refql = `query RumSparklineBydatetimeMinute(
		$accountTag: string,
		$visitsFilter: ZoneHttpRequestsAdaptiveGroupsFilter_InputObject
	) {
		viewer {
			accounts(
				filter: {accountTag: $accountTag}
			) {
				visits: rumPageloadEventsAdaptiveGroups(
					limit: 5000
				) {
					avg {
						sampleInterval
					}
					dimensions {
						ts: datetimeMinute
					}
					sum {
						visits
					}
				}
			}
		}
	}`

	var _, err0 = parser.ParseQuery(&ast.Source{
		Name:  `test-1`,
		Input: refql,
	})

	assert.NoError(err0)

	// the JSON GraphQL representation we want to turn into a GraphQL query
	var gqlstr = `{
		"query": {
			"RumSparklineBydatetimeMinute": {
				"@vars": [{
					"$accountTag": "string"
				}, {
					"$visitsFilter": "ZoneHttpRequestsAdaptiveGroupsFilter_InputObject"
				}],
				"viewer": {
					"accounts": {
						"@args": {
							"filter": {
								"accountTag": "$accountTag"
							}
						},
						"rumPageloadEventsAdaptiveGroups": {
							"@alias": "visits",
							"@args": {
								"limit": 5000
							},
							"sum": {
								"visits": null
							},
							"avg": {
								"sampleInterval": null
							},
							"dimensions": {
								"ts": "datetimeMinute"
							}
						}
					}
				}
			}
		}
	}`

	var gquery GraphQLQuery

	var err1 = json.Unmarshal([]byte(gqlstr), &gquery)
	assert.NoError(err1)

	var rendered, err2 = gquery.Render()
	assert.NoError(err2)

	// make sure input and output queries are indentical except for whitespace
	assert.EqualValues(
		regexp.MustCompile(`\s*`).ReplaceAllString(refql, ``),
		regexp.MustCompile(`\s*`).ReplaceAllString(rendered, ``),
	)
}
