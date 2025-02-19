package config

import (
	"fmt"
	"reflect"

	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
)

type Keys struct {
	WindowDetails    string `default:"1" doc:"Press this to show the details of a resource"`
	WindowLogs       string `default:"2" doc:"Press this to show any collected logs of a resource"`
	FilterLogsToggle string `default:"L" doc:"Filter resources by those currently logging"`
	LogToggle        string `default:"l" doc:"Toggle log collection for this pod"`
	FilterSearch     string `default:"/" doc:"Filter by a string"`
	Save             string `default:"q" doc:"Save the recorded state to a file"`
	NewLabel         string `default:"m" doc:"Mark this timestamp with a label"`
	RotateViewToggle string `default:"tab" doc:"Rotate view"`
	Quit             string `default:"ctrl+c" doc:"Quit"`
	VCRRewind        string `default:"left" doc:"In VCR mode, rewind, press multiple times to speed up"`
	VCRFastForward   string `default:"right" doc:"In VCR mode, fast forward, press multiple times to speed up"`
	VCRPlay          string `default:" " doc:"In VCR mode, toggle play/pause"`
	VCROff           string `default:"esc" doc:"Exit VCR mode and resume at the latest timestamp"`
	Toggle           string `default:"enter" doc:"Toggle folding the resource category/kind view"`
	DetailsUp        string `default:"shift+up" doc:"Jump up in the details view by 10 lines"`
	DetailsDown      string `default:"shift+down" doc:"Jump down in the details view by 10 lines"`
	Up               string `default:"up" doc:"Go up a resource"`
	Down             string `default:"down" doc:"Go down a resource"`
	PageUp           string `default:"alt+up" doc:"Go up 10 resources"`
	PageDown         string `default:"alt+down" doc:"Go down 10 resources"`
	DeleteResource   string `default:"ctrl+d" doc:"Delete a resource"`
	Exec             string `default:"s" doc:"Exec into a shell for this pod"`
	Pod              string `default:"P" doc:"Filter all pods"`
	Debug            string `default:"ctrl+d" doc:"Debug log window"`
	NextLabel        string `default:"shift+right" doc:"In VCR mode, jump to the next marked label"`
	PrevLabel        string `default:"shift+left" doc:"In VCR mode, jump to the previous marked label"`
}

func (k Keys) Print() {
	typeOfK := reflect.TypeOf(k)
	fmt.Println("Keybindings:")
	for i := 0; i < typeOfK.NumField(); i++ {
		field := typeOfK.Field(i)
		defaultVal := field.Tag.Get("default")
		doc := field.Tag.Get("doc")
		fmt.Printf("    '%s' - %s\n", defaultVal, doc)
	}
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

	temp := map[string]any{
		"metrics":     "false",
		"profiling":   "false",
		"keybindings": map[string]string{},
	}
	err := config.LoadData(temp)
	if err != nil {
		return cfg, fmt.Errorf("error loading data: %w", err)
	}

	err = config.LoadExists("config.yaml", "~/.khronoscope/config.yaml", "../../config.yaml")
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
