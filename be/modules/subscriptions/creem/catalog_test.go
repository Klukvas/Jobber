package creem

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModeMatches(t *testing.T) {
	for _, tc := range []struct {
		name, mode string
		live, want bool
	}{
		{name: "test", mode: "test", want: true}, {name: "sandbox", mode: "sandbox", want: true}, {name: "live", mode: "prod", live: true, want: true},
		{name: "test cannot grant live", mode: "test", live: true}, {name: "sandbox cannot grant live", mode: "sandbox", live: true}, {name: "live cannot grant test", mode: "prod"},
		{name: "missing"}, {name: "unknown", mode: "local"},
	} {
		t.Run(tc.name, func(t *testing.T) { require.Equal(t, tc.want, ModeMatches(tc.mode, tc.live)) })
	}
}

// The payloads are shaped like GET /v1/products responses observed in test mode
// (2026-10-28): the trial length arrives as trial_period_days, null without one.
func TestIsProMonthlyReadsTheTrialFromTheAPIField(t *testing.T) {
	const pro = `{"id":"prod_pro","mode":"test","price":2400,"currency":"USD","billing_type":"recurring","billing_period":"every-month","status":"active","trial_period_days":%s}`
	for _, tc := range []struct {
		name, trial string
		want        bool
	}{
		{name: "no trial", trial: "null", want: true},
		{name: "a trial is not the monthly Pro product", trial: "7"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var product Product
			require.NoError(t, json.Unmarshal([]byte(fmt.Sprintf(pro, tc.trial)), &product))
			require.Equal(t, tc.want, product.IsProMonthly("prod_pro", false))
		})
	}
}
