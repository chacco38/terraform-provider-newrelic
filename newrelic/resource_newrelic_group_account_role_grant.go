package newrelic

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/newrelic/newrelic-client-go/v2/newrelic"
	"github.com/newrelic/newrelic-client-go/v2/pkg/authorizationmanagement"
)

func resourceNewRelicGroupAccountRoleGrant() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceNewRelicGroupAccountRoleGrantCreate,
		ReadContext:   resourceNewRelicGroupAccountRoleGrantRead,
		DeleteContext: resourceNewRelicGroupAccountRoleGrantDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceNewRelicGroupAccountRoleGrantImport,
		},
		Schema: map[string]*schema.Schema{
			"group_id": {
				Type:         schema.TypeString,
				Description:  "The ID of the group to which the role is granted.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"account_id": {
				Type:         schema.TypeInt,
				Description:  "The ID of the account on which the role is granted.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IntAtLeast(1),
			},
			"role_id": {
				Type:         schema.TypeString,
				Description:  "The ID of the role (standard or custom) to be granted. Must be an account-scoped role.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
		},
	}
}

func groupAccountRoleGrantID(groupID string, accountID int, roleID string) string {
	return fmt.Sprintf("%s:%d:%s", groupID, accountID, roleID)
}

func resourceNewRelicGroupAccountRoleGrantCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).NewClient

	groupID := d.Get("group_id").(string)
	accountID := d.Get("account_id").(int)
	roleID := d.Get("role_id").(string)

	log.Printf("[INFO] granting role %s on account %d to group %s\n", roleID, accountID, groupID)
	_, err := client.AuthorizationManagement.AuthorizationManagementGrantAccessWithContext(
		ctx,
		authorizationmanagement.AuthorizationManagementGrantAccess{
			GroupId: groupID,
			AccountAccessGrants: []authorizationmanagement.AuthorizationManagementAccountAccessGrant{
				{AccountID: accountID, RoleId: roleID},
			},
		},
	)
	if err != nil {
		return diag.FromErr(err)
	}

	// Read is deliberately not called here. A successful GrantAccess call is the confirmation,
	// and reading right away could miss the grant because of propagation delay.
	d.SetId(groupAccountRoleGrantID(groupID, accountID, roleID))

	return nil
}

func resourceNewRelicGroupAccountRoleGrantRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).NewClient

	groupID := d.Get("group_id").(string)
	accountID := d.Get("account_id").(int)
	roleID := d.Get("role_id").(string)

	found, err := accountRoleGrantExists(ctx, client, groupID, accountID, roleID)
	if err != nil {
		return diag.FromErr(err)
	}

	if !found {
		log.Printf("[WARN] role grant %s not found, removing from state\n", d.Id())
		d.SetId("")
	}

	return nil
}

func resourceNewRelicGroupAccountRoleGrantDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).NewClient

	groupID := d.Get("group_id").(string)
	accountID := d.Get("account_id").(int)
	roleID := d.Get("role_id").(string)

	log.Printf("[INFO] revoking role %s on account %d from group %s\n", roleID, accountID, groupID)
	_, err := client.AuthorizationManagement.AuthorizationManagementRevokeAccessWithContext(
		ctx,
		authorizationmanagement.AuthorizationManagementRevokeAccess{
			GroupId: groupID,
			AccountAccessGrants: []authorizationmanagement.AuthorizationManagementAccountAccessGrant{
				{AccountID: accountID, RoleId: roleID},
			},
		},
	)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// resourceNewRelicGroupAccountRoleGrantImport imports a grant using the ID format <group_id>:<account_id>:<role_id>.
func resourceNewRelicGroupAccountRoleGrantImport(_ context.Context, d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
	parts := strings.Split(d.Id(), ":")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return nil, fmt.Errorf("invalid import ID %q: expected format <group_id>:<account_id>:<role_id> (all parts must be non-empty)", d.Id())
	}

	accountID, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid account_id %q in import ID: %w", parts[1], err)
	}

	_ = d.Set("group_id", parts[0])
	_ = d.Set("account_id", accountID)
	_ = d.Set("role_id", parts[2])

	return []*schema.ResourceData{d}, nil
}

const groupGrantedRolesQuery = `query($groupIDs: [ID!], $cursor: String) {
  actor {
    organization {
      authorizationManagement {
        authenticationDomains {
          authenticationDomains {
            groups(id: $groupIDs) {
              groups {
                id
                roles(cursor: $cursor) {
                  roles {
                    accountId
                    roleId
                  }
                  nextCursor
                }
              }
            }
          }
        }
      }
    }
  }
}`

type groupGrantedRolesResponse struct {
	Actor struct {
		Organization struct {
			AuthorizationManagement struct {
				AuthenticationDomains struct {
					AuthenticationDomains []struct {
						Groups struct {
							Groups []struct {
								ID    string `json:"id"`
								Roles struct {
									Roles []struct {
										AccountID int `json:"accountId"`
										RoleID    int `json:"roleId"`
									} `json:"roles"`
									NextCursor string `json:"nextCursor"`
								} `json:"roles"`
							} `json:"groups"`
						} `json:"groups"`
					} `json:"authenticationDomains"`
				} `json:"authenticationDomains"`
			} `json:"authorizationManagement"`
		} `json:"organization"`
	} `json:"actor"`
}

// accountRoleGrantExists reports whether the group has been granted the role on the account.
func accountRoleGrantExists(ctx context.Context, client *newrelic.NewRelic, groupID string, accountID int, roleID string) (bool, error) {
	cursor := ""
	seenCursors := map[string]bool{}

	for {
		resp := groupGrantedRolesResponse{}
		vars := map[string]interface{}{"groupIDs": []string{groupID}}
		if cursor != "" {
			vars["cursor"] = cursor
		}

		if err := client.NerdGraph.QueryWithResponseAndContext(ctx, groupGrantedRolesQuery, vars, &resp); err != nil {
			return false, err
		}

		nextCursor := ""
		for _, ad := range resp.Actor.Organization.AuthorizationManagement.AuthenticationDomains.AuthenticationDomains {
			for _, g := range ad.Groups.Groups {
				if g.ID != groupID {
					continue
				}
				for _, r := range g.Roles.Roles {
					if r.AccountID == accountID && strconv.Itoa(r.RoleID) == roleID {
						return true, nil
					}
				}
				nextCursor = g.Roles.NextCursor
			}
		}

		if nextCursor == "" {
			return false, nil
		}
		if seenCursors[nextCursor] {
			return false, fmt.Errorf("pagination did not advance while fetching roles of group %s (cursor %q repeated)", groupID, nextCursor)
		}
		seenCursors[nextCursor] = true
		cursor = nextCursor
	}
}
