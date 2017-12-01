package tool

import (
	"bytes"
	"text/template"

	"github.com/ciao-project/ciao/client"
	"github.com/intel/tfortools"
	"github.com/pkg/errors"
)

func GetTenants(c *client.Client, flags CommandOpts) (bytes.Buffer, error) {
	var t *template.Template
	var result bytes.Buffer

	if c.Template != "" {
		var err error
		t, err = tfortools.CreateTemplate("tenant-list", c.Template, nil)
		if err != nil {
			return result, errors.Wrap(err, "Failed to create template")
		}
	} else {
		var err error
		t, err = tfortools.CreateTemplate("tenant-list", "{{table .}}", nil)
		if err != nil {
			return result, errors.Wrap(err, "Failed to create template")
		}
	}

	if flags.Quotas {
		return listTenantQuotas(c, t)
	}
	if flags.Resources {
		return listTenantResources(c, t)
	}
	if flags.Config {
		if c.IsPrivileged() == false {
			if c.TenantID == "" {
				return result, errors.New("Missing required tenant ID")
			}
			return listTenantConfig(c, t, c.TenantID)
		}

		if flags.TenantID == "" {
			return result, errors.New("Missing required tenant parameter")
		}

		return listTenantConfig(c, t, flags.TenantID)
	}
	if flags.All {
		if c.IsPrivileged() == false {
			return result, errors.New("The all command is for privileged users only")
		}
		return listAllTenants(c, t)
	}

	return listUserTenants(c, t)
}

func listUserTenants(c *client.Client, t *template.Template) (bytes.Buffer, error) {
	var projects []Project
	var result bytes.Buffer

	for _, t := range c.Tenants {
		projects = append(projects, Project{ID: t})
	}

	if t != nil {
		if err := t.Execute(&result, &projects); err != nil {
			return result, errors.Wrap(err, "Error listing user tenants")
		}
		return result, nil
	}

	return result, nil
}

func listTenantQuotas(c *client.Client, t *template.Template) (bytes.Buffer, error) {
	var result bytes.Buffer

	if c.TenantID == "" {
		return result, errors.New("Missing required -tenant-id parameter")
	}

	resources, err := c.ListTenantQuotas()
	if err != nil {
		return result, errors.Wrap(err, "Error listing tenant quotas")
	}

	if t != nil {
		if err := t.Execute(&result, &resources); err != nil {
			return result, errors.Wrap(err, "Error listing user tenants")
		}
		return result, nil
	}

	return result, nil
}

func listTenantResources(c *client.Client, t *template.Template) (bytes.Buffer, error) {
	var result bytes.Buffer

	if c.TenantID == "" {
		return result, errors.New("Missing required -tenant-id parameter")
	}

	usage, err := c.ListTenantResources()
	if err != nil {
		return result, errors.Wrap(err, "Error listing tenant resources")
	}

	if t != nil {
		if err := t.Execute(&result, &usage); err != nil {
			return result, errors.Wrap(err, "Error listing user tenants")
		}
		return result, nil
	}

	return result, nil
}

func listTenantConfig(c *client.Client, t *template.Template, tenantID string) (bytes.Buffer, error) {
	var result bytes.Buffer

	config, err := c.GetTenantConfig(tenantID)
	if err != nil {
		return result, errors.Wrap(err, "Failed to get tenant config")
	}

	if t != nil {
		if err := t.Execute(&result, &config); err != nil {
			return result, errors.Wrap(err, "Failed to list tenant config")
		}
		return result, nil
	}

	return result, nil
}

func listAllTenants(c *client.Client, t *template.Template) (bytes.Buffer, error) {
	var result bytes.Buffer

	tenants, err := c.ListTenants()
	if err != nil {
		return result, errors.Wrap(err, "Error listing tenants")
	}

	if t != nil {
		if err := t.Execute(&result, &tenants); err != nil {
			return result, errors.Wrap(err, "Failed to list all tenants")
		}
		return result, nil
	}

	return result, nil
}
