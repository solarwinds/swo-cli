package logs

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/olebedev/when"
	"gopkg.in/yaml.v3"
)

const (
	defaultCount      = 100
	defaultConfigFile = "~/.swo-cli.yaml"
	defaultApiUrl     = "https://api.na-01.cloud.solarwinds.com"
)

var (
	now = time.Now()

	errMinTimeFlag  = errors.New("failed to parse --min-time flag")
	errMaxTimeFlag  = errors.New("failed to parse --max-time flag")
	errMissingToken = errors.New("failed to find token")

	timeLayouts = []string{
		time.Layout,
		time.ANSIC,
		time.UnixDate,
		time.RubyDate,
		time.RFC822,
		time.RFC822Z,
		time.RFC850,
		time.RFC1123,
		time.RFC1123Z,
		time.RFC3339,
		time.RFC3339Nano,
		time.Kitchen,
		time.Stamp,
		time.StampMilli,
		time.StampNano,
		time.DateTime,
		time.DateOnly,
		time.TimeOnly,
		"2006-01-02 15:04:05",
	}
)

type Options struct {
	args       []string
	count      int
	configFile string
	group      string
	system     string
	maxTime    string
	minTime    string
	json       bool
	version    bool

	ApiUrl string `yaml:"api-url"`
	Token  string `yaml:"token"`
}

func (opts *Options) Init(args []string) (*Options, error) {
	opts.args = args

	if opts.minTime != "" {
		result, err := parseTime(opts.minTime)
		if err != nil {
			return nil, errors.Join(errMinTimeFlag, err)
		}

		opts.minTime = result
	}

	if opts.maxTime != "" {
		result, err := parseTime(opts.maxTime)
		if err != nil {
			return nil, errors.Join(errMaxTimeFlag, err)
		}

		opts.maxTime = result
	}

	configPath := opts.configFile
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	localConfig := filepath.Join(cwd, ".swo-cli.yaml")
	if _, err := os.Stat(localConfig); err == nil {
		configPath = localConfig
	} else if strings.HasPrefix(opts.configFile, "~/") {
		usr, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("error while resolving current user to read configuration file: %w", err)
		}

		configPath = filepath.Join(usr.HomeDir, opts.configFile[2:])
	}

	if content, err := os.ReadFile(configPath); err == nil {
		err = yaml.Unmarshal(content, opts)
		if err != nil {
			return nil, fmt.Errorf("error while unmarshaling %s config file: %w", configPath, err)
		}
	}

	if token := os.Getenv("SWO_API_TOKEN"); token != "" {
		opts.Token = token
	}

	if opts.Token == "" && !opts.version {
		return nil, errMissingToken
	}

	return opts, nil
}

func parseTime(input string) (string, error) {
	location := time.Local
	if strings.HasSuffix(input, " UTC") {
		l, err := time.LoadLocation("UTC")
		if err != nil {
			return "", err
		}

		location = l

		input = strings.ReplaceAll(input, " UTC", "")
	}

	for _, layout := range timeLayouts {
		result, err := time.Parse(layout, input)
		if err == nil {
			result = result.In(location)
			return result.Format(time.RFC3339), nil
		}
	}

	result, err := when.EN.Parse(input, now)
	if err != nil {
		return "", err
	}
	if result == nil {
		return "", errors.New("failed to parse time")
	}

	return result.Time.In(location).Format(time.RFC3339), nil
}
