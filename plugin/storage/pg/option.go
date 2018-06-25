package pg

import (
	// "github.com/jaegertracing/jaeger/pkg/es/config"
	"flag"

	"github.com/jaegertracing/jaeger/pkg/pg/config"
	"github.com/spf13/viper"
)

const (
	suffixUsername   = ".username"
	suffixPassword   = ".password"
	suffixServerURLs = ".server-urls"
)

type Options struct {
	primary *namespaceConfig

	others map[string]*namespaceConfig
}

type namespaceConfig struct {
	config.Configuration
	namespace string
}

func NewOption(namespace string) *Options {
	opt := &Options{
		primary: &namespaceConfig{
			Configuration: config.Configuration{
				Username:  "postgres",
				Password:  "123456",
				ServerURL: "127.0.0.1:5432",
			},
			namespace: namespace,
		},
	}
	return opt
}

// AddFlags adds flags for Options
func (opt *Options) AddFlags(flagSet *flag.FlagSet) {
	addFlags(flagSet, opt.primary)
}

func addFlags(flagSet *flag.FlagSet, nsConfig *namespaceConfig) {
	flagSet.String(
		nsConfig.namespace+suffixUsername,
		nsConfig.Username,
		"The username required by ElasticSearch")
	flagSet.String(
		nsConfig.namespace+suffixPassword,
		nsConfig.Password,
		"The password required by ElasticSearch")
	flagSet.String(
		nsConfig.namespace+suffixServerURLs,
		nsConfig.ServerURL,
		"The comma-separated list of ElasticSearch servers, must be full url i.e. http://localhost:9200")
}

func (opt *Options) InitFromViper(v *viper.Viper) {
	initFromViper(opt.primary, v)
}

func initFromViper(cfg *namespaceConfig, v *viper.Viper) {
	cfg.ServerURL = v.GetString(cfg.namespace + suffixServerURLs)
	cfg.Username = v.GetString(cfg.namespace + suffixUsername)
	cfg.Password = v.GetString(cfg.namespace + suffixPassword)
}
