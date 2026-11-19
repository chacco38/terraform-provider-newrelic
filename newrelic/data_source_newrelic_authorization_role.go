package newrelic

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceNewRelicAuthorizationRole() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceNewRelicAuthorizationRoleRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The name of the role (standard or custom) to be queried. Matched against both the name and the display name of the role.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"scope": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The scope of the role (for example, `account`). Useful when roles with the same name exist in different scopes.",
			},
			"type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The type of the role.",
			},
		},
	}
}

func dataSourceNewRelicAuthorizationRoleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).NewClient

	name := d.Get("name").(string)
	scope := d.Get("scope").(string)

	log.Printf("[INFO] fetching roles to find %q\n", name)

	var matches []authorizationRole
	cursor := ""
	seenCursors := map[string]bool{}
	for {
		resp := authorizationRolesResponse{}
		vars := map[string]interface{}{}
		if cursor != "" {
			vars["cursor"] = cursor
		}
		if err := client.NerdGraph.QueryWithResponseAndContext(ctx, authorizationRolesQuery, vars, &resp); err != nil {
			return diag.FromErr(err)
		}

		roles := resp.Actor.Organization.AuthorizationManagement.Roles
		for _, r := range roles.Roles {
			if (r.Name == name || r.DisplayName == name) && (scope == "" || r.Scope == scope) {
				matches = append(matches, r)
			}
		}

		if roles.NextCursor == "" {
			break
		}
		if seenCursors[roles.NextCursor] {
			return diag.Errorf("pagination did not advance while fetching roles (cursor %q repeated)", roles.NextCursor)
		}
		seenCursors[roles.NextCursor] = true
		cursor = roles.NextCursor
	}

	switch len(matches) {
	case 0:
		if scope != "" {
			return diag.Errorf("no role found with the name %q and scope %q", name, scope)
		}
		return diag.Errorf("no role found with the name %q", name)
	case 1:
		// found exactly one role
	default:
		return diag.Errorf("%d roles found with the name %q; specify `scope` to narrow the result", len(matches), name)
	}

	d.SetId(matches[0].ID)
	_ = d.Set("scope", matches[0].Scope)
	_ = d.Set("type", matches[0].Type)

	return nil
}

// The query is defined here instead of using the client's GetRoles, because GetRoles does not
// declare the $cursor variable in its query and therefore cannot page through the results.
const authorizationRolesQuery = `query($cursor: String) {
  actor {
    organization {
      authorizationManagement {
        roles(cursor: $cursor) {
          nextCursor
          roles {
            displayName
            id
            name
            scope
            type
          }
        }
      }
    }
  }
}`

type authorizationRole struct {
	DisplayName string `json:"displayName"`
	ID          string `json:"id"`
	Name        string `json:"name"`
	Scope       string `json:"scope"`
	Type        string `json:"type"`
}

type authorizationRolesResponse struct {
	Actor struct {
		Organization struct {
			AuthorizationManagement struct {
				Roles struct {
					NextCursor string              `json:"nextCursor"`
					Roles      []authorizationRole `json:"roles"`
				} `json:"roles"`
			} `json:"authorizationManagement"`
		} `json:"organization"`
	} `json:"actor"`
}
