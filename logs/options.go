package logs

import (
	"errors"
	"strings"
	"time"

	"github.com/olebedev/when"
)

var (
	now = time.Now()

	errMinTimeFlag = errors.New("failed to parse --min-time flag")
	errMaxTimeFlag = errors.New("failed to parse --max-time flag")

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
	configFile string
	group      string
	system     string
	maxTime    string
	minTime    string
	json       bool
	follow     bool
	Token      string
	APIURL     string
}

func (opts *Options) Init(args []string) error {
	opts.args = args

	if opts.minTime != "" {
		result, err := parseTime(opts.minTime)
		if err != nil {
			return errors.Join(errMinTimeFlag, err)
		}

		opts.minTime = result
	}

	if opts.follow { // set maxTime to <now - 10s> when 'follow' flag is set, it is used only for the first request
		result, err := parseTime(time.Now().Add(-10 * time.Second).Format(time.RFC3339))
		if err != nil {
			return errors.Join(errMaxTimeFlag, err)
		}

		opts.maxTime = result
	}

	if opts.maxTime != "" {
		result, err := parseTime(opts.maxTime)
		if err != nil {
			return errors.Join(errMaxTimeFlag, err)
		}

		opts.maxTime = result
	}

	return nil
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
		return "", ErrInvalidDateTime
	}

	return result.Time.In(location).Format(time.RFC3339), nil
}
