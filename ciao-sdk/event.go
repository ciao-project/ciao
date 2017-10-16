package sdk

import (
	"fmt"
	"os"

	"github.com/ciao-project/ciao/ciao-controller/types"

	"github.com/intel/tfortools"
)

func listEvent(args []string) error {
	tenant := *tenantID

	if len(args) != 0 {
		tenant = args[0]
	}

	if CommandFlags.All == false && tenant == "" {
		errorf("Missing required tenant-id parameter")
		return nil
	}

	var events types.CiaoEvents
	var url string

	if CommandFlags.All == true {
		url = buildComputeURL("events")
	} else {
		url = buildComputeURL("%s/events", tenant)
	}

	resp, err := sendHTTPRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalHTTPResponse(resp, &events)
	if err != nil {
		fatalf(err.Error())
	}

	if Template != "" {
		return tfortools.OutputToTemplate(os.Stdout, "event-list", Template,
			&events.Events, nil)
	}

	fmt.Printf("%d Ciao event(s):\n", len(events.Events))
	for i, event := range events.Events {
		fmt.Printf("\t[%d] %v: %s:%s (Tenant %s)\n", i+1, event.Timestamp, event.EventType, event.Message, event.TenantID)
	}
	return nil
}