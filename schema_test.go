package orchestra

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ghetzel/go-stockutil/httputil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/testify/require"
)

var TestServer = func() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("%v\n", r.URL)

		switch p := strings.TrimSuffix(r.URL.Path, `/`); p {
		case `/test/v1/services`:
			httputil.RespondJSON(w, map[string]any{
				`services`: []string{
					`/test/v1/services/first`,
					`/test/v1/services/second`,
					`/test/v1/services/k8s-test-1`,
					`/test/v1/services/k8s-test-2`,
					`/test/v1/services/k8s-test-3`,
					`/test/v1/services/k8s-test-4`,
				},
			})
		case `/test/v1/repos`:
			httputil.RespondJSON(w, map[string]any{
				`entries`: []map[string]any{
					{`name`: `test-1-thing`, `path`: `/repos/test-1-thing`},
					{`name`: `test-2-thing`, `path`: `/repos/test-2-thing`},
					{`name`: `test-3-api`, `path`: `/repos/test-3-api`},
					{`name`: `test-4-api`, `path`: `/repos/test-4-api`},
				},
			})
		case `/test/v1/manifest.json`:
			httputil.RespondJSON(w, map[string]any{
				`env-dev`: []map[string]any{
					{`env`: `env-dev`, `project`: `project-a`},
					{`env`: `env-dev`, `project`: `project-b`},
					{`env`: `env-dev`, `project`: `project-c`},
				},
				`env-prod`: []map[string]any{
					{`env`: `env-prod`, `project`: `project-a`},
					{`env`: `env-prod`, `project`: `project-b`},
					{`env`: `env-prod`, `project`: `project-c`},
				},
			})
		default:
			httputil.RespondJSON(w, fmt.Errorf("nope"), http.StatusNotImplemented)
		}
	}))
}()

func init() {
	RegisterEndpoint(`k8s`, &Endpoint{
		URL: TestServer.URL + `/test/v1/services/`,
		ResultFilters: []any{
			`$filter(services, /\/k8s-/)`,
		},
	})

	RegisterEndpoint(`api-repos`, &Endpoint{
		URL: TestServer.URL + `/test/v1/repos/`,
		ResultFilters: []any{
			`entries[path ~> /-api$/].(name)`,
		},
	})

	RegisterEndpoint(`deploy-manifest`, &Endpoint{
		URL: TestServer.URL + `/test/v1/manifest.json`,
	})

	RegisterEndpoint(`deploy-project`, &Endpoint{
		URL: TestServer.URL + `/test/v1/deployments/`,
	})
}

func TestSchemaBasicQuery(t *testing.T) {
	var assert = require.New(t)

	var schema = &Schema{
		Name: `test-1`,
		Pipeline: &Pipeline{
			Steps: []*PipelineStep{
				{
					Query: &QueryOptions{
						UseEndpoint: `api-repos`,
					},
				}, {
					ResultTarget: `kubes`,
					Query: &QueryOptions{
						UseEndpoint: `k8s`,
					},
				},
			},
		},
	}

	var response, err = schema.Query(nil)
	assert.NoError(err)

	var results, ok = response.Result.(map[string]any)
	assert.True(ok)

	assert.EqualValues(
		[]string{
			"test-3-api",
			"test-4-api",
		},
		sliceutil.Stringify(results[`api-repos`]),
	)

	assert.EqualValues(
		[]string{
			"/test/v1/services/k8s-test-1",
			"/test/v1/services/k8s-test-2",
			"/test/v1/services/k8s-test-3",
			"/test/v1/services/k8s-test-4",
		},
		sliceutil.Stringify(results[`kubes`]),
	)
}

// func TestSchemaRepeatingQuery(t *testing.T) {
// 	var assert = require.New(t)
// 	var testEnv = `mga-dev`

// 	var schema = &Schema{
// 		Name: `test-repeater`,
// 		Pipeline: &Pipeline{
// 			Steps: []*PipelineStep{
// 				{
// 					ResultTarget: `manifest`,
// 					Query: &QueryOptions{
// 						UseEndpoint: `deploy-manifest`,
// 					},
// 					Transforms: []any{
// 						`$sift($, function($v, $k){ $k = 'mga-dev' })`,
// 					},
// 				}, {
// 					ResultTarget: `envs`,
// 					Transforms:   []any{`$keys(manifest)`},
// 					// OmitOutput: true,
// 				}, {
// 					ResultTarget: `projects`,
// 					Query: &QueryOptions{
// 						UseEndpoint: `deploy-project`,
// 						ForEach: `$each(manifest, function($v, $k){
// 							$v
// 						})`,
// 						ParamsQuery: `{'project': project}`,
// 						Context: Context{
// 							Params: map[string]any{
// 								`env`: testEnv,
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}

// 	var response, err = schema.Query(&QueryOptions{
// 		Context: Context{
// 			Variables: map[string]any{
// 				`ENV_STAGE`: testEnv,
// 			},
// 		},
// 	})
// 	assert.NoError(err)

// 	var results, ok1 = response.Result.(map[string]any)
// 	assert.True(ok1)

// 	var manifest, ok2 = results[`manifest`].(map[string]any)
// 	assert.True(ok2)

// 	for env, manifestProjects := range manifest {
// 		assert.Equal(testEnv, env)

// 		var prjs, ok = manifestProjects.([]any)
// 		assert.True(ok)
// 		assert.True(len(prjs) > 0)
// 	}

// 	var envs = sliceutil.Stringify(results[`envs`])
// 	assert.True(len(envs) > 0)

// 	var projects = sliceutil.Sliceify(results[`projects`])
// 	assert.True(len(projects) == 6)

// 	for _, p := range projects {
// 		var project = maputil.M(p)

// 		assert.NotEmpty(project.String(`env`))
// 		assert.NotEmpty(project.String(`project`))
// 	}

// }
