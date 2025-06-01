package orchestra

type Schema struct {
	Name     string    `yaml:"name,omitempty"     json:"name,omitempty"`
	Summary  string    `yaml:"summary,omitempty"  json:"summary,omitempty"`
	Pipeline *Pipeline `yaml:"pipeline,omitempty" json:"pipeline,omitempty"`
}

func (schema *Schema) Query(query *QueryOptions) (*QueryResponse, error) {
	var queryResponse = NewQueryResponse(nil)

	if pipeline := schema.Pipeline; pipeline != nil {
		if res, err := pipeline.Query(query); err == nil {
			queryResponse = res
		} else {
			return queryResponse.Failed(err)
		}
	} else {
		queryResponse.EndpointName = schema.Name
	}

	queryResponse.Query = query

	return queryResponse.Completed(nil)
}
