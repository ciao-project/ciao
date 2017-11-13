package sdk

import (
	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/ciao-project/ciao/client"
	"github.com/pkg/errors"
)

func GetTraces(c *client.Client, flags CommandOpts) (types.CiaoTracesSummary, error) {
	traces, err := c.ListTraceLabels()
	if err != nil {
		return traces, errors.Wrap(err, "Error listing trace labels")
	}

	return traces, err
}

func GetTraceData(c *client.Client, flags CommandOpts) (types.CiaoTraceData, error) {
	var traceData types.CiaoTraceData

	if flags.Args[0] == "" {
		return traceData, errors.New("Missing required label parameter")
	}

	traceData, err := c.GetTraceData(flags.Args[0])
	if err != nil {
		return traceData, errors.Wrap(err, "Error getting trace data")
	}

	return traceData, err
}
