package sdk

func Show(object string, args []string) {
	var ret error

	switch object {
	case "instance":
		if len(args) == 0 {
			ret = listInstances()
		} else {
			ret = showInstance(args)
		}
	case "workload":
		if len(args) == 0 {
			ret = listWorkloads()
		} else {
			ret = showWorkload(args)
		}
	case "event":
		listEvent(args)
	case "image":
		listImages()
	}
	if ret != nil {
		errorf("ERROR:%s\n", ret)
	}
}
