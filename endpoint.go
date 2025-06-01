package orchestra

import (
	"github.com/ghetzel/go-stockutil/stringutil"
)

var registeredEndpoints = make(map[string]*Endpoint)

func RegisterEndpoint(name string, endpoint *Endpoint) {
	var regname, _ = stringutil.SplitPair(name, `:`)
	endpoint.Name = regname
	registeredEndpoints[regname] = endpoint
}

type DataKind int

const (
	AnyKind DataKind = iota
	ObjectKind
	ListKind
)

func (kind DataKind) String() string {
	switch kind {
	case ObjectKind:
		return `object`
	case ListKind:
		return `list`
	default:
		return `any`
	}
}

type Endpoint struct {
	Name          string         `yaml:"name,omitempty"        json:"name,omitempty"`
	Method        string         `yaml:"method,omitempty"      json:"method,omitempty"`
	URL           string         `yaml:"url"                   json:"url"`
	RequestBody   any            `yaml:"body,omitempty"        json:"body,omitempty"`
	GraphQL       *GraphQLQuery  `yaml:"graphql,omitempty"     json:"graphql,omitempty"`
	PathParams    map[string]any `yaml:"path_params,omitempty" json:"path_params,omitempty"`
	Params        map[string]any `yaml:"params,omitempty"      json:"params,omitempty"`
	Headers       map[string]any `yaml:"headers,omitempty"     json:"headers,omitempty"`
	ResultType    DataKind       `yaml:"type,omitempty"        json:"type,omitempty"`
	ResultFilters []any          `yaml:"filters,omitempty"     json:"filters,omitempty"`
	Variables     map[string]any `yaml:"variables,omitempty"   json:"variables,omitempty"`
}
