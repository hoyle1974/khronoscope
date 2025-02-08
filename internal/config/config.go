package config

import (
	"fmt"

	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
)

type Keys struct {
	WindowDetails    string `default:"1"`
	WindowLogs       string `default:"2"`
	FilterLogsToggle string `default:"L"`
	LogToggle        string `default:"l"`
	FilterSearch     string `default:"/"`
	Save             string `default:"q"`
	NewLabel         string `default:"m"`
	RotateViewToggle string `default:"tab"`
	Quit             string `default:"ctrl+c"`
	VCRRewind        string `default:"left"`
	VCRFastForward   string `default:"right"`
	VCRPlay          string `default:" "`
	VCROff           string `default:"esc"`
	Toggle           string `default:"enter"`
	DetailsUp        string `default:"shift+up"`
	DetailsDown      string `default:"shift+down"`
	Up               string `default:"up"`
	Down             string `default:"down"`
	PageUp           string `default:"alt+up"`
	PageDown         string `default:"alt+down"`
	DeleteResource   string `default:"ctrl+d"`
	Exec             string `default:"s"`
	Pod              string `default:"P"`
}

type Config struct {
	Metrics     bool
	Profiling   bool
	KeyBindings Keys
}

var cfg = Config{}

func Get() Config {
	return cfg
}

func InitConfig() (Config, error) {
	config.WithOptions(config.ParseEnv, config.ParseDefault)
	config.AddDriver(yaml.Driver)

	err := config.LoadExists("config.yaml", "~/.khronoscope/config.yaml", "../../config.yaml")
	if err != nil {
		return cfg, fmt.Errorf("error loading config file: %w", err)
	}
	err = config.Decode(&cfg)
	if err != nil {
		return cfg, fmt.Errorf("error decoding config data: %w", err)
	}

	// Hack
	err = config.BindStruct("keybindings", &cfg.KeyBindings)
	if err != nil {
		return cfg, fmt.Errorf("error decoding config data (keybindings): %w", err)
	}
	return cfg, nil
}
