package orchestra

import (
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io/fs"
	"net/http"
	"strings"

	"github.com/ghetzel/go-stockutil/httputil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/typeutil"

	"github.com/ghetzel/go-stockutil/log"
)

const DefaultAddress = `127.0.0.1:42305`

type Server struct {
	*http.Server
	*http.ServeMux
	config *Config
}

func NewServer(config *Config) *Server {
	var server = new(Server)
	var addr string

	if config != nil && config.ServerAddress != `` {
		addr = config.ServerAddress
	} else {
		addr = DefaultAddress
	}

	server.config = config
	server.Server = &http.Server{
		Addr:    addr,
		Handler: server,
	}

	server.ServeMux = http.NewServeMux()
	server.init()

	return server
}

func (server *Server) init() {
	if subfs, err := fs.Sub(embedded, `static`); err == nil {
		server.HandleFunc(`/orchestra/v1/config/`, server.httpGetConfig)
		server.HandleFunc(`/orchestra/v1/queries/`, server.httpDatasetQuery)

		server.Handle(`/`, http.FileServer(
			http.FS(subfs),
		))
	} else {
		log.Panicf("embedded fs: %v", err)
	}

	log.Noticef("starting server at http://%v", server.Addr)
}

func (server *Server) httpGetConfig(w http.ResponseWriter, r *http.Request) {
	httputil.RespondJSON(w, server.config)
}

func (server *Server) httpDatasetQuery(w http.ResponseWriter, r *http.Request) {
	if qname := pathParam(r, 4).String(); qname != `` {
		var opts = NewQueryOptions()
		opts.Variables = make(map[string]any)

		for k, vv := range r.URL.Query() {
			switch len(vv) {
			case 0:
				opts.Variables[k] = nil
			case 1:
				opts.Variables[k] = vv[0]
			default:
				opts.Variables[k] = vv
			}
		}

		var response, err = DefaultConfig.Datasets.QuerySchema(qname, opts)
		w.Header().Set(`Content-Type`, `application/json`)

		if err == nil {
			w.WriteHeader(http.StatusOK)
		} else {
			httputil.RespondJSON(w, err, http.StatusInternalServerError)
			return
		}

		if httputil.QBool(r, `_debug`) {
			httputil.RespondJSON(w, response)
		} else if response != nil {
			httputil.RespondJSON(w, response.Result)
		}
	} else {
		httputil.RespondJSON(
			w,
			fmt.Errorf("no query name provided"),
			http.StatusNotFound,
		)
	}
}

func pathParam(r *http.Request, i int) typeutil.Variant {
	return typeutil.V(
		sliceutil.Get(
			strings.Split(r.URL.Path, `/`),
			i,
		),
	)
}
