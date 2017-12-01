package tool

import (
	"bytes"

	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/ciao-project/ciao/client"
	"github.com/pkg/errors"
)

func listComputeNodes(c *client.Client, flags CommandOpts) (types.CiaoNodes, error) {
	nodes, err := c.ListComputeNodes()
	if err != nil {
		return nodes, errors.Wrap(err, "Error listing compute nodes")
	}

	return nodes, err
}

func listNetworkNodes(c *client.Client, flags CommandOpts) (types.CiaoNodes, error) {
	nodes, err := c.ListNetworkNodes()
	if err != nil {
		return nodes, errors.Wrap(err, "Error listing network nodes")
	}

	return nodes, err
}

func listNodes(c *client.Client, flags CommandOpts) (types.CiaoNodes, error) {
	nodes, err := c.ListNodes()
	if err != nil {
		return nodes, errors.Wrap(err, "Error listing nodes")
	}

	return nodes, err
}

func listCNCINodes(c *client.Client, flags CommandOpts) (types.CiaoCNCIs, error) {
	cncis, err := c.ListCNCIs()
	if err != nil {
		return cncis, errors.Wrap(err, "Error listing CNCIs")
	}
	/*
		if t != nil {
			if err := t.Execute(os.Stdout, &cncis.CNCIs); err != nil {
				fatalf(err.Error())
			}
			return cncis, err
		}*/

	return cncis, err
}

func GetNodes(c *client.Client, flags CommandOpts) (bytes.Buffer, error) {
	var result bytes.Buffer

	if flags.ComputeNode {
		nodes, err := listComputeNodes(c, flags)
		if err == nil {
			c.PrettyPrint(&result, "list-computenode", nodes)
			return result, err
		}
	} else if flags.NetworkNode {
		nodes, err := listNetworkNodes(c, flags)
		if err == nil {
			c.PrettyPrint(&result, "list-networknode", nodes)
			return result, err
		}
	} else if flags.CNCINode {
		nodes, err := listCNCINodes(c, flags)
		if err == nil {
			c.PrettyPrint(&result, "list-cncinode", nodes)
			return result, err
		}
	} else if flags.All {
		nodes, err := listNodes(c, flags)
		if err == nil {
			c.PrettyPrint(&result, "list-nodes", nodes)
			return result, err
		}
	}

	return bytes.Buffer{}, errors.New("Please enter a valid node type")
}
