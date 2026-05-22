package driver

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInfoResponseUnmarshalsDecimalSpaceSizes(t *testing.T) {
	const payload = `{
		"state": true,
		"data": {
			"space_info": {
				"all_total": {"size": 155555555555555.25, "size_format": "141.49TB"},
				"all_remain": {"size": "123.75", "size_format": "123B"},
				"all_use": {"size": 42, "size_format": "42B"}
			},
			"login_devices_info": {},
			"imei_info": false
		}
	}`

	var resp InfoResponse
	require.NoError(t, json.Unmarshal([]byte(payload), &resp))
	assert.Equal(t, int64(155555555555555), resp.Data.SpaceInfo.AllTotal.Size)
	assert.Equal(t, int64(123), resp.Data.SpaceInfo.AllRemain.Size)
	assert.Equal(t, int64(42), resp.Data.SpaceInfo.AllUse.Size)
}
