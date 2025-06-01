package orchestra

import (
	"fmt"
	"sync"

	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

type PipelineStep struct {
	Name         string        `yaml:"name,omitempty"         json:"name,omitempty"`
	Summary      string        `yaml:"summary,omitempty"      json:"summary,omitempty"`
	ResultTarget string        `yaml:"target"                 json:"target"`
	Query        *QueryOptions `yaml:"query,omitempty"        json:"query,omitempty"`
	Transforms   []any         `yaml:"transforms,omitempty"   json:"transforms,omitempty"`
	OmitOutput   bool          `yaml:"omit,omitempty"         json:"omit,omitempty"`
	SkipStep     bool          `yaml:"skip,omitempty"         json:"skip,omitempty"`
	WithContext  bool          `yaml:"with_context,omitempty" json:"with_context,omitempty"`
	Optional     bool          `yaml:"optional,omitempty"     json:"optional,omitempty"`
	Parallel     bool          `yaml:"parallel"               json:"parallel"`
}

func (step *PipelineStep) Retrieve(parentOptions *QueryOptions, initdata any) (any, QueryContext, error) {
	var result any = initdata
	var vars = make(map[string]any)
	var context = make(QueryContext)

	context[`vars`] = vars
	context[RootVarName] = result

	if v, err := parentOptions.RenderVariables(result); err == nil {
		vars = v
	} else {
		return nil, nil, err
	}

	if query, err := parentOptions.Merge(step.Query); step.Query != nil && err == nil {
		if v, err := query.RenderVariables(result); err == nil {
			vars = v
		} else {
			return nil, nil, err
		}

		if endpoint, ok := registeredEndpoints[query.UseEndpoint]; ok && endpoint != nil {
			var concurrentGroup sync.WaitGroup
			var concurrentUsed bool

			if foreach := query.ForEach; foreach != `` {
				if elements, err := applyJsonata(
					result,
					vars,
					foreach,
				); err == nil {
					var accumulatedResults []any
					var subcontexts []QueryContext

					if !typeutil.IsArray(elements) {
						return nil, nil, fmt.Errorf("foreach: JSONata query must return an array")
					}

					for i, el := range sliceutil.Sliceify(elements) {
						var subvars = maputil.M(vars).MapNative()
						subvars[`item`] = el
						subvars[`index`] = i

						subcontexts = append(subcontexts, QueryContext(subvars))

						if renderedQuery, err := query.Render(subvars); err == nil {
							var res *QueryResponse
							var rerr error

							if step.Parallel {
								concurrentUsed = true
								res, rerr = renderedQuery.QueryConcurrent(&concurrentGroup, endpoint)
							} else {
								res, rerr = renderedQuery.Query(endpoint)
							}

							if rerr == nil {
								accumulatedResults = append(accumulatedResults, res.Result)
							} else if step.Optional {
								continue
							} else {
								return nil, nil, rerr
							}
						} else {
							return nil, nil, err
						}
					}

					if concurrentUsed {
						concurrentGroup.Wait()
					}

					result = accumulatedResults
					context[`elements`] = subcontexts
				} else {
					return nil, nil, fmt.Errorf("foreach: %v", err)
				}
			} else if renderedQuery, err := query.Render(result); err == nil {
				var res *QueryResponse
				var rerr error

				if step.Parallel {
					concurrentUsed = true
					res, rerr = renderedQuery.QueryConcurrent(&concurrentGroup, endpoint)
				} else {
					res, rerr = renderedQuery.Query(endpoint)
				}

				if rerr == nil {
					result = res.Result
				} else {
					return nil, nil, rerr
				}
			} else {
				return nil, nil, err
			}
		} else if query.UseEndpoint != `` {
			return nil, nil, fmt.Errorf("undefined endpoint %q", query.UseEndpoint)
		}
	} else if err != nil {
		return nil, nil, fmt.Errorf("bad query: %v", err)
	}

	if r, err := applyJsonata(result, vars, step.Transforms...); err == nil {
		result = r
	} else {
		return nil, nil, fmt.Errorf("output filters: %v", err)
	}

	return result, context, nil
}
