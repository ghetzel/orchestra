package orchestra

import (
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/ghetzel/go-stockutil/httputil"
	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

type QueryContext map[string]any

type QueryOptions struct {
	Context
	PathParams      map[string]any `yaml:"path_params,omitempty"      json:"path_params,omitempty"`
	ForEach         string         `yaml:"foreach,omitempty"          json:"foreach,omitempty"`
	PathParamsQuery string         `yaml:"path_params_json,omitempty" json:"path_params_json,omitempty"`
	ParamsQuery     any            `yaml:"params_json,omitempty"      json:"params_json,omitempty"`
	HeadersQuery    any            `yaml:"headers_json,omitempty"     json:"headers_json,omitempty"`
	VariablesQuery  any            `yaml:"variables_json,omitempty"   json:"variables_json,omitempty"`
	Transforms      []any          `yaml:"transforms,omitempty"       json:"transforms,omitempty"`
	UseEndpoint     string         `yaml:"endpoint,omitempty"         json:"endpoint,omitempty"`
}

func NewQueryOptions() *QueryOptions {
	return &QueryOptions{
		Context: Context{
			Params:    make(map[string]any),
			Headers:   make(map[string]any),
			Variables: make(map[string]any),
		},
		PathParams: make(map[string]any),
	}
}

func (opts *QueryOptions) Render(data any) (*QueryOptions, error) {
	var rq = *opts

	if values, err := opts.RenderVariables(data); err == nil {
		rq.Variables = values
	} else {
		return nil, err
	}

	if values, err := opts.RenderPathParams(data); err == nil {
		rq.PathParams = values
	} else {
		return nil, err
	}

	if values, err := opts.RenderParams(data); err == nil {
		rq.Params = values
	} else {
		return nil, err
	}

	if values, err := opts.RenderHeaders(data); err == nil {
		rq.Headers = values
	} else {
		return nil, err
	}

	return &rq, nil
}

func (opts *QueryOptions) RenderPathParams(data any) (map[string]any, error) {
	return opts.renderValuesFor(`path_params`, data)
}

func (opts *QueryOptions) RenderParams(data any) (map[string]any, error) {
	return opts.renderValuesFor(`params`, data)
}

func (opts *QueryOptions) RenderHeaders(data any) (map[string]any, error) {
	return opts.renderValuesFor(`headers`, data)
}

func (opts *QueryOptions) RenderVariables(data any) (map[string]any, error) {
	return opts.renderValuesFor(`variables`, data)
}

func (opts *QueryOptions) renderValuesFor(field string, data any) (map[string]any, error) {
	var results = make(map[string]any)
	var explicit map[string]any
	var jq any

	switch field {
	case `path_params`:
		explicit = opts.PathParams
		jq = opts.PathParamsQuery
	case `params`:
		explicit = opts.Params
		jq = opts.ParamsQuery
	case `headers`:
		explicit = opts.Headers
		jq = opts.HeadersQuery
	case `variables`:
		explicit = opts.Variables
		jq = opts.VariablesQuery
	default:
		return nil, fmt.Errorf("no field type")
	}

	if !typeutil.IsZero(jq) {
		if queried, err := applyJsonata(data, opts.Variables, jq); err == nil {
			for k, v := range maputil.M(queried).MapNative() {
				if k == RootVarName {
					continue
				}

				results[k] = v
			}
		} else {
			return nil, err
		}
	}

	for k, v := range explicit {
		if k == RootVarName {
			continue
		}

		results[k] = v
	}

	return results, nil
}

func (query *QueryOptions) Merge(other *QueryOptions) (*QueryOptions, error) {
	var result = NewQueryOptions()

	if other == nil {
		return result, nil
	}

	for _, qo := range []*QueryOptions{
		query,
		other,
	} {
		if v := qo.ForEach; v != `` {
			result.ForEach = v
		}
		if v := qo.UseEndpoint; v != `` {
			result.UseEndpoint = v
		}
		if v := qo.PathParamsQuery; v != `` {
			result.PathParamsQuery = v
		}
		if v := qo.ParamsQuery; v != `` {
			result.ParamsQuery = v
		}
		if v := qo.HeadersQuery; v != `` {
			result.HeadersQuery = v
		}
		if v := qo.VariablesQuery; v != `` {
			result.VariablesQuery = v
		}
		if v := qo.Transforms; len(v) > 0 {
			result.Transforms = v
		}
		for k, v := range qo.Headers {
			if !typeutil.IsZero(v) {
				result.Headers[k] = v
			}
		}
		for k, v := range qo.PathParams {
			if !typeutil.IsZero(v) {
				result.PathParams[k] = v
			}
		}
		for k, v := range qo.Params {
			if !typeutil.IsZero(v) {
				result.Params[k] = v
			}
		}
		for k, v := range qo.Variables {
			if !typeutil.IsZero(v) {
				result.Variables[k] = v
			}
		}
	}

	return result, nil
}

func (query *QueryOptions) QueryConcurrent(wg *sync.WaitGroup, endpoint *Endpoint) (*QueryResponse, error) {
	if wg != nil {
		defer wg.Done()
		wg.Add(1)
	}

	var results = make(chan any)

	go func() {
		if r, err := query.Query(endpoint); err == nil {
			results <- r
		} else {
			results <- err
		}
	}()

	for res := range results {
		if r, ok := res.(*QueryResponse); ok {
			return r, nil
		} else if err := res.(error); ok {
			return nil, err
		} else {
			return nil, fmt.Errorf("invalid response from goroutine (%T)", res)
		}
	}

	return nil, fmt.Errorf("no response from goroutine")
}

func (query *QueryOptions) Query(endpoint *Endpoint) (*QueryResponse, error) {
	var queryResponse = NewQueryResponse(endpoint)
	var headers = make(map[string]any)
	var params = make(map[string]any)
	var vars = make(map[string]any)

	if query == nil {
		query = new(QueryOptions)
	}

	// endpoint-specific headers
	for k, v := range endpoint.Headers {
		headers[k] = v
	}

	// endpoint-specific params
	for k, v := range endpoint.Params {
		params[k] = v
	}

	// endpoint-specific variables
	for k, v := range endpoint.Variables {
		vars[k] = v
	}

	// query-specific headers (overrides endpoint)
	for k, v := range query.Headers {
		headers[k] = v
	}

	// query-specific params (overrides endpoint)
	for k, v := range query.Params {
		params[k] = v
	}

	// query-specific variables (overrides endpoint)
	for k, v := range query.Variables {
		vars[k] = v
	}

	queryResponse.Context = map[string]any{
		`vars`:    vars,
		`params`:  params,
		`headers`: headers,
	}

	if _, err := query.retrieveViaURL(endpoint, queryResponse, headers, params, vars); err != nil {
		return queryResponse, err
	}

	return queryResponse.Completed(nil)
}

func (query *QueryOptions) retrieveViaURL(
	endpoint *Endpoint,
	queryResponse *QueryResponse,
	headers map[string]any,
	params map[string]any,
	vars map[string]any,
) (*QueryResponse, error) {
	// interpolate any fields in the URL
	var fmturl = FormatString(endpoint.URL, queryResponse.Context)

	// parse interpolated URL into url.URL to validate it
	if endpointURL, err := url.Parse(fmturl); err == nil {
		if client, err := httputil.NewClient(endpointURL.String()); err == nil {
			if endpoint.Method == `` {
				if endpoint.GraphQL != nil {
					endpoint.Method = `POST`
				} else {
					endpoint.Method = `GET`
				}
			}

			var method = httputil.Method(strings.ToUpper(endpoint.Method))

			// perform the HTTP request
			// log.Debugf("orchestra/endpoint[%s] %s %v params=%+v headers=%+v vars=%+v", endpoint.Name, method, endpointURL, params, headers, vars)
			var body any

			if gql := endpoint.GraphQL; gql != nil {
				if gquery, err := gql.Render(); err == nil {
					var gq = map[string]any{
						`name`:      gql.Name,
						`query`:     gquery,
						`variables`: vars,
					}

					queryResponse.Context[`graphql`] = gq
					body = gq
				} else {
					return queryResponse.Failed(err)
				}
			} else {
				body = endpoint.RequestBody
			}

			if response, err := client.Request(method, "", body, params, headers); err == nil {
				var out any

				if err := client.Decode(response.Body, &out); err == nil {
					// apply endpoint-level filters first
					if filtered, err := applyJsonata(
						out,
						vars,
						endpoint.ResultFilters...,
					); err == nil {
						out = filtered
					} else {
						return queryResponse.Failed(err)
					}

					// apply query-level filters next
					if filtered, err := applyJsonata(
						out,
						vars,
						query.Transforms...,
					); err == nil {
						out = filtered
					} else {
						return queryResponse.Failed(err)
					}

					queryResponse.Result = out
				} else {
					return queryResponse.Failed(err)
				}
			} else {
				return queryResponse.Failed(err)
			}
		} else {
			return queryResponse.Failed(err)
		}
	} else {
		return queryResponse.Failed(err)
	}

	return queryResponse, nil
}
