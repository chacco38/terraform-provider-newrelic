---
layout: 'newrelic'
page_title: 'New Relic: newrelic_group_account_role_grant'
sidebar_current: 'docs-newrelic-resource-group-account-role-grant'
description: |-
  Grant a role (standard or custom) on an account to a group in New Relic.
---

# Resource: newrelic\_group\_account\_role\_grant

The `newrelic_group_account_role_grant` resource grants a role on a specific account to a group, and revokes it on destroy. Both standard and custom roles are supported, as long as the role is account-scoped.

Each resource manages exactly one grant (one group, one account, one role).

## Example Usage

```hcl
data "newrelic_authentication_domain" "foo" {
  name = "Test Authentication Domain"
}

resource "newrelic_group" "foo" {
  name                     = "Test Group"
  authentication_domain_id = data.newrelic_authentication_domain.foo.id
}

data "newrelic_authorization_role" "custom" {
  name  = "My Custom Role"
  scope = "account"
}

resource "newrelic_group_account_role_grant" "foo" {
  group_id   = newrelic_group.foo.id
  account_id = 1234567
  role_id    = data.newrelic_authorization_role.custom.id
}
```

## Argument Reference

The following arguments are supported. Changing any of them replaces the resource.

* `group_id` - (Required) The ID of the group to which the role is granted.
* `account_id` - (Required) The ID of the account on which the role is granted.
* `role_id` - (Required) The ID of the role to be granted. This can be looked up by name with the `newrelic_authorization_role` data source.

## Attributes Reference

In addition to the attributes listed above, the following attribute is also exported by this resource:

* `id` - The ID of the grant, in the format `<group_id>:<account_id>:<role_id>`.

## Import

A grant can be imported using the format `<group_id>:<account_id>:<role_id>`. Example:

```shell
$ terraform import newrelic_group_account_role_grant.foo <group_id>:<account_id>:<role_id>
```
