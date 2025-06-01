package orchestra

import (
	"fmt"

	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/rxutil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

const DefaultResultKey string = `result`
const DefaultContextPrefix string = `context_`
const RootVarName string = `root`

type Context struct {
	Params    map[string]any `yaml:"params,omitempty"    json:"params,omitempty"`
	Headers   map[string]any `yaml:"headers,omitempty"   json:"headers,omitempty"`
	Variables map[string]any `yaml:"variables,omitempty" json:"variables,omitempty"`
}

type Pipeline struct {
	Context
	Name     string          `yaml:"name,omitempty"     json:"name,omitempty"`
	Summary  string          `yaml:"summary,omitempty"  json:"summary,omitempty"`
	Required Context         `yaml:"required,omitempty" json:"required,omitempty"`
	Steps    []*PipelineStep `yaml:"steps"              json:"steps"`
}

func (pipeline *Pipeline) validateRules(queryResponse *QueryResponse, opts *QueryOptions) error {
	for _, facet := range []string{
		`param`,
		`variable`,
	} {
		var workmap map[string]any
		var tgtmap map[string]any

		switch facet {
		case `param`:
			workmap = pipeline.Required.Params
			tgtmap = opts.Params
		case `variable`:
			workmap = pipeline.Required.Variables
			tgtmap = opts.Variables

			if _, ok := tgtmap[RootVarName]; !ok {
				return fmt.Errorf("invalid validateRules facet '%v': no $%v variable", RootVarName, facet)
			}
		default:
			log.Panicf("invalid validateRules facet '%v'", facet)
		}

		for name, req := range workmap {
			var err error

			if typeutil.Bool(req) {
				var xerr = fmt.Errorf("value for '%v' is missing", name)

				if len(tgtmap) == 0 {
					err = xerr
				} else if v, ok := tgtmap[name]; !ok || typeutil.IsZero(v) {
					err = xerr
				}
			} else {
				var testrx = tgtmap[name]
				if rxutil.Match(testrx, typeutil.String(req)) == nil {
					err = fmt.Errorf("value for '%v' must match expression '%v'", name, testrx)
				}
			}

			if err != nil {
				return queryResponse.AddErrorf("pipeline %s: %v", facet, err)
			}
		}
	}

	return nil
}

func (pipeline *Pipeline) Query(opts *QueryOptions) (*QueryResponse, error) {
	var results = make(map[string]any)
	var queryResponse = NewQueryResponse(nil)

	if opts == nil {
		opts = new(QueryOptions)
	}

	for i, step := range pipeline.Steps {
		i = i + 1

		if step.Query == nil {
			step.Query = new(QueryOptions)
		}

		var key string

		// figure out which key in the response the data goes to
		if step.ResultTarget != `` {
			key = step.ResultTarget
		} else if key == `` && step.Query != nil && step.Query.UseEndpoint != `` {
			key = step.Query.UseEndpoint
		} else {
			key = DefaultResultKey
		}

		var merged = NewQueryOptions()

		if m, err := opts.Merge(&QueryOptions{
			Context: pipeline.Context,
		}); err == nil {
			merged = m
		} else {
			return queryResponse.Failedf("step %d [%s]: %v", i, key, err)
		}

		if m, err := merged.Merge(opts); err == nil {
			merged = m
		} else {
			return queryResponse.Failedf("step %d [%s]: %v", i, key, err)
		}

		if m, err := merged.Merge(step.Query); err == nil {
			merged = m
		} else {
			return queryResponse.Failedf("step %d [%s]: %v", i, key, err)
		}

		merged.Variables[RootVarName] = results

		if err := pipeline.validateRules(queryResponse, merged); err != nil {
			return queryResponse.Failed(err)
		}

		if step.SkipStep {
			continue
		} else if result, ctx, err := step.Retrieve(merged, results); err == nil {
			step.ResultTarget = key

			results[key] = result

			if step.WithContext {
				results[DefaultContextPrefix+key] = ctx
			}
		} else if step.Optional {
			results[key] = nil
			if step.WithContext {
				results[DefaultContextPrefix+key] = ctx
			}

			log.Debugf("step %d [%s]: %v", i, key, err)
			continue
		} else {
			return queryResponse.Failedf("step %d [%s]: %v", i, key, err)
		}
	}

	for _, step := range pipeline.Steps {
		if step.OmitOutput {
			delete(results, step.ResultTarget)
		}
	}

	return queryResponse.Completed(results)
}
