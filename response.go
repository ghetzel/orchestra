package orchestra

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ghetzel/go-stockutil/sliceutil"
)

type QueryResponse struct {
	Endpoint     *Endpoint     `yaml:"-"                  json:"-"`
	EndpointName string        `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
	Result       any           `yaml:"result"             json:"result"`
	StartedAt    time.Time     `yaml:"started_at"         json:"started_at"`
	CompletedAt  time.Time     `yaml:"completed_at"       json:"completed_at"`
	Took         float64       `yaml:"took,omitempty"     json:"took,omitempty"`
	Errors       []string      `yaml:"errors,omitempty"   json:"errors,omitempty"`
	Query        *QueryOptions `yaml:"query"              json:"query"`
	Context      QueryContext  `yaml:"context"            json:"context"`
}

func NewQueryResponse(endpoint *Endpoint) *QueryResponse {
	var qr = new(QueryResponse)

	if endpoint != nil {
		qr.Endpoint = endpoint
		qr.EndpointName = endpoint.Name
	}

	qr.StartedAt = time.Now()
	return qr
}

func (response *QueryResponse) Error() error {
	var errs []error

	for _, msg := range response.Errors {
		errs = append(errs, errors.New(msg))
	}

	return errors.Join(errs...)
}

func (response *QueryResponse) AddError(errs ...error) error {
	for _, err := range errs {
		var msgs = strings.Split(err.Error(), "\n")
		response.Errors = append(response.Errors, msgs...)
		response.Errors = sliceutil.UniqueStrings(response.Errors)
	}

	return response.Error()
}

func (response *QueryResponse) AddErrorf(format string, items ...any) error {
	var err = fmt.Errorf(format, items...)
	response.AddError(err)
	return err
}

func (response *QueryResponse) Completed(result any) (*QueryResponse, error) {
	if result != nil {
		response.Result = result
	}

	response.CompletedAt = time.Now()

	if response.CompletedAt.After(response.StartedAt) {
		response.Took = float64(response.CompletedAt.Sub(response.StartedAt).Milliseconds())
	}

	return response, response.Error()
}

func (response *QueryResponse) Failed(err error) (*QueryResponse, error) {
	if err != nil {
		response.AddError(err)
	}

	return response.Completed(nil)
}

func (response *QueryResponse) Failedf(format string, args ...any) (*QueryResponse, error) {
	return response.Failed(fmt.Errorf(format, args...))
}
