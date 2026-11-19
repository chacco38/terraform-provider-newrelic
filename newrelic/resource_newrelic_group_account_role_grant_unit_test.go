//go:build unit

package newrelic

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"
)

func TestGroupAccountRoleGrantID(t *testing.T) {
	require.Equal(t, "grp-1:1234567:1253", groupAccountRoleGrantID("grp-1", 1234567, "1253"))
}

func TestGroupAccountRoleGrantImport(t *testing.T) {
	r := resourceNewRelicGroupAccountRoleGrant()

	t.Run("valid ID", func(t *testing.T) {
		d := r.Data(nil)
		d.SetId("grp-1:1234567:1253")

		out, err := resourceNewRelicGroupAccountRoleGrantImport(context.Background(), d, nil)
		require.NoError(t, err)
		require.Len(t, out, 1)
		require.Equal(t, "grp-1", out[0].Get("group_id"))
		require.Equal(t, 1234567, out[0].Get("account_id"))
		require.Equal(t, "1253", out[0].Get("role_id"))
	})

	for name, id := range map[string]string{
		"too few parts":      "grp-1:1234567",
		"too many parts":     "grp-1:1234567:1253:extra",
		"non-numeric a/c ID": "grp-1:abc:1253",
		"empty":              "",
		"empty group_id":     ":1234567:1253",
		"empty account_id":   "grp-1::1253",
		"empty role_id":      "grp-1:1234567:",
	} {
		id := id
		t.Run(name, func(t *testing.T) {
			var d *schema.ResourceData = r.Data(nil)
			d.SetId(id)

			_, err := resourceNewRelicGroupAccountRoleGrantImport(context.Background(), d, nil)
			require.Error(t, err)
		})
	}
}
