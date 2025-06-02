package orchestra

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/stringutil"
)

var ConfigFile string = func() (config string) {
	if v := os.Getenv(`ORCHESTRA_CONFIG`); v != `` {
		config = v
	} else {
		config = "~/.config/orchestra/config.yaml"
	}

	config = fileutil.MustExpandUser(config)

	if _, err := os.Stat(config); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Panicf("config error: %v", err)
	}

	return
}()

var DatasetsPath []string = func() (pathlist []string) {
	if v := os.Getenv(`ORCHESTRA_DATASET_PATH`); v != `` {
		pathlist = stringutil.SplitTrimSpace(v, `:`)
	} else {
		pathlist = []string{
			"~/.config/orchestra/datasets",
		}
	}

	for i, _ := range pathlist {
		pathlist[i] = fileutil.MustExpandUser(pathlist[i])
	}

	return
}()

var ManagedConfigDir string = func() string {
	if v := os.Getenv(`ORCHESTRA_CONFIG_DIR`); v != `` {
		return v
	} else {
		return filepath.Dir(ConfigFile)
	}
}()

type DatasetConfig struct {
	Endpoints map[string]*Endpoint `yaml:"endpoints" json:"endpoints"`
	Queries   map[string]*Schema   `yaml:"queries"   json:"queries"`
}

func (dataset *DatasetConfig) QuerySchema(name string, query *QueryOptions) (*QueryResponse, error) {
	if schema, ok := dataset.Queries[name]; ok {
		return schema.Query(query)
	} else {
		return nil, fmt.Errorf("undefined schema %q", name)
	}
}

type Config struct {
	ServerAddress string         `yaml:"address,omitempty" json:"address,omitempty"`
	Datasets      *DatasetConfig `yaml:"datasets"          json:"datasets"`
}

var DefaultConfig *Config

func NewConfig() *Config {
	return &Config{
		Datasets: &DatasetConfig{
			Endpoints: make(map[string]*Endpoint),
			Queries:   make(map[string]*Schema),
		},
	}
}

func loadConfigFile(filename string) (*Config, error) {
	var cfg = NewConfig()

	if f, err := os.Open(filename); err == nil {
		defer f.Close()

		if err := configDecoder(f).Decode(cfg); err == nil {
			log.Infof("loaded config %v", filename)
			return cfg, nil
		} else {
			return nil, fmt.Errorf("yaml parse error in %v: %v", filename, err)
		}
	} else {
		return nil, fmt.Errorf("%v: %v", filename, err)
	}
}

func loadDatasets(base *DatasetConfig, datasetDirs ...string) error {
	if base == nil {
		return fmt.Errorf("cannot merge with nil DatasetConfig")
	}

	for _, setdir := range datasetDirs {
		if fileutil.IsNonemptyDir(setdir) {
			if err := filepath.WalkDir(setdir, func(path string, d fs.DirEntry, err error) error {
				if !d.IsDir() {
					switch fileutil.GetMimeType(d.Name()) {
					case `application/yaml`:
						if f, err := os.Open(path); err == nil {
							defer f.Close()

							var subset DatasetConfig

							if err := configDecoder(f).Decode(&subset); err == nil {
								for k, v := range subset.Endpoints {
									base.Endpoints[k] = v
								}

								for k, v := range subset.Queries {
									base.Queries[k] = v
								}
							} else {
								log.Errorf("datasets %v: %v", d.Name(), err)
							}
						} else {
							log.Errorf("datasets: %v", err)
						}
					}
				}

				return nil
			}); err != nil {
				return err
			}
		}
	}

	for name, endpoint := range base.Endpoints {
		endpoint.Name = name
		RegisterEndpoint(name, endpoint)
	}

	for name, query := range base.Queries {
		query.Name = name

		if pipeline := query.Pipeline; pipeline != nil {
			for i, step := range pipeline.Steps {
				if stepq := step.Query; stepq != nil {
					if ep := stepq.UseEndpoint; ep != `` {
						if e, ok := base.Endpoints[ep]; !ok || e == nil {
							return fmt.Errorf("query %v, step %v: undefined endpoint %q", name, i, ep)
						}
					}
				}
			}
		}
	}

	return nil
}

func LoadDefaultConfig() error {
	DefaultConfig = NewConfig()

	if ConfigFile != `` {
		if cfg, err := loadConfigFile(ConfigFile); err == nil {
			DefaultConfig = cfg
		} else {
			return err
		}

		if err := loadDatasets(DefaultConfig.Datasets, DatasetsPath...); err != nil {
			return err
		}
	}

	return nil
}

func configDecoder(r io.Reader) (decoder *yaml.Decoder) {
	decoder = yaml.NewDecoder(r)
	decoder.KnownFields(true)
	return
}
