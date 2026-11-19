---
layout: 'newrelic'
page_title: 'New Relic: newrelic_authorization_role'
sidebar_current: 'docs-newrelic-datasource-authorization-role'
description: |-
  Look up a role (standard or custom) in New Relic by name.
---

# Data Source: newrelic\_authorization\_role

Use this data source to look up the ID of a role, standard or custom, by its name. The ID can be used with the `newrelic_group_account_role_grant` resource.

## Example Usage

```hcl
data "newrelic_authorization_role" "foo" {
  name  = "My Custom Role"
  scope = "account"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the role. It is matched against both the name and the display name of roles.
* `scope` - (Optional) The scope of the role, such as `account`. Specify it when roles with the same name exist in different scopes. An error is returned if more than one role matches.

## Attributes Reference

In addition to the arguments above, the following attributes are exported:

* `id` - The ID of the role.
* `scope` - The scope of the role.
* `type` - The type of the role.
