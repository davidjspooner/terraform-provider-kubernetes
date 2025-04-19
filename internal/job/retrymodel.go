package job

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type RetryModel struct {
	MaxAttempts  *int64    `tfsdk:"attempts"`
	FastFail     *[]string `tfsdk:"fast_fail"`
	InitialPause *string   `tfsdk:"initial_pause"`
	Interval     *string   `tfsdk:"interval"`
	Timeout      *string   `tfsdk:"timeout"`
}

func MergeRetryModels(models ...*RetryModel) (*RetryModel, error) {
	merged := &RetryModel{}
	for _, model := range models {
		if model == nil {
			continue
		}
		if model.MaxAttempts != nil {
			merged.MaxAttempts = model.MaxAttempts
		}
		if model.FastFail != nil {
			merged.FastFail = model.FastFail
		}
		if model.InitialPause != nil {
			merged.InitialPause = model.InitialPause
		}
		if model.Interval != nil {
			merged.Interval = model.Interval
		}
		if model.Timeout != nil {
			merged.Timeout = model.Timeout
		}
	}

	return merged, nil
}

func (rs *RetryModel) NewHelper() (*RetryHelper, error) {

	if rs == nil {
		rs = &RetryModel{}
	}

	var rh RetryHelper

	if rs.MaxAttempts != nil && *rs.MaxAttempts > 0 {
		rh.MaxAttempts = int(*rs.MaxAttempts)
	}

	if rs.FastFail != nil && len(*rs.FastFail) > 0 {
		rh.FastFail = nil
		for _, hint := range *rs.FastFail {
			re, err := regexp.Compile(hint)
			if err != nil {
				return nil, fmt.Errorf("FastFail hint %q is not a valid regular expression, %v", hint, err)
			}
			rh.FastFail = append(rh.FastFail, re)
		}
	}

	var err error

	//parse pause ( default is no pause )
	if rs.InitialPause != nil {
		pauseStr := strings.TrimSpace(*rs.InitialPause)
		if pauseStr != "" {
			rh.Pause, err = time.ParseDuration(pauseStr)
			if err != nil {
				return nil, err
			}
		}
	}

	//parse interval ( default is 10s 20s 30s )
	if rs.Interval != nil && *rs.Interval != "" {
		rh.Interval, err = ParseDurationList(*rs.Interval)
		if err != nil {
			return nil, err
		}
	}

	if rs.Timeout != nil {
		timeoutStr := strings.TrimSpace(*rs.Timeout)
		if timeoutStr != "" {
			rh.Timeout, err = time.ParseDuration(timeoutStr)
			if err != nil {
				return nil, err
			}
		}
	}

	return &rh, nil
}

func DefineRetryModelSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Description: "Retry options when modifying resource.",
		Attributes: map[string]schema.Attribute{
			"attempts": schema.NumberAttribute{
				MarkdownDescription: "maximum number of attempts",
				Optional:            true,
			},
			"fast_fail": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "regex patterns to search errors and if found, fast fail withoug further attempts",
				Optional:            true,
			},
			"initial_pause": schema.StringAttribute{
				MarkdownDescription: "pause before first attempt",
				Optional:            true,
			},
			"interval": schema.StringAttribute{
				MarkdownDescription: "list of intervals between attempts ( eg 5s 10s 20s 30s )",
				Optional:            true,
			},
			"timeout": schema.StringAttribute{
				MarkdownDescription: "timeout for the whole operation",
				Optional:            true,
			},
		},
		Optional: true,
	}
}
