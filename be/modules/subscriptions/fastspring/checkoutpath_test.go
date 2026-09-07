package fastspring

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePopupCheckoutPath(t *testing.T) {
	t.Run("accepts a popup checkout", func(t *testing.T) {
		require.NoError(t, ValidatePopupCheckoutPath("fluxlab/popup-jobber"))
	})

	t.Run("refuses the old full-page web checkout", func(t *testing.T) {
		// This is the exact value the migration replaces. It is a *valid*
		// checkout path in every other respect, which is why it needs its own
		// check: SBL would open a popup onto nothing.
		err := ValidatePopupCheckoutPath("fluxlab/jobber-checkout")

		require.ErrorIs(t, err, ErrNotPopupCheckout)
	})

	t.Run("refuses anything that is not a plain two-segment path", func(t *testing.T) {
		invalid := map[string]string{
			"single segment":  "fluxlab",
			"three segments":  "fluxlab/popup/jobber",
			"leading slash":   "/fluxlab/popup-jobber",
			"trailing slash":  "fluxlab/popup-jobber/",
			"path traversal":  "fluxlab/../accounts",
			"encoded slash":   "fluxlab/popup%2Fjobber",
			"query smuggling": "fluxlab/popup-jobber?x=1",
			"empty":           "",
		}

		for name, path := range invalid {
			t.Run(name, func(t *testing.T) {
				require.ErrorIs(t, ValidatePopupCheckoutPath(path), ErrInvalidCheckoutPath)
			})
		}
	})

	t.Run("the prefix must start the segment, not merely appear in it", func(t *testing.T) {
		require.ErrorIs(t, ValidatePopupCheckoutPath("fluxlab/jobber-popup-checkout"), ErrNotPopupCheckout)
	})
}

func TestPopupStorefront(t *testing.T) {
	t.Run("derives the storefront for each store mode", func(t *testing.T) {
		tests := []struct {
			name string
			live bool
			want string
		}{
			{name: "test", live: false, want: "fluxlab.test.onfastspring.com/popup-jobber"},
			{name: "live", live: true, want: "fluxlab.onfastspring.com/popup-jobber"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				got, err := PopupStorefront("fluxlab/popup-jobber", tc.live)

				require.NoError(t, err)
				assert.Equal(t, tc.want, got)
			})
		}
	})

	t.Run("test and live storefronts are never the same host", func(t *testing.T) {
		// The whole point of deriving both from one value: a deployment cannot
		// be pointed at the other store by editing a variable in isolation.
		test, err := PopupStorefront("fluxlab/popup-jobber", false)
		require.NoError(t, err)
		live, err := PopupStorefront("fluxlab/popup-jobber", true)
		require.NoError(t, err)

		assert.NotEqual(t, test, live)
	})

	t.Run("refuses a checkout path the popup cannot open", func(t *testing.T) {
		_, err := PopupStorefront("fluxlab/jobber-checkout", false)

		require.ErrorIs(t, err, ErrNotPopupCheckout)
	})

	t.Run("refuses a path that could reach another host", func(t *testing.T) {
		// The result goes into a script tag in a buyer's browser, so a
		// configured value must not be able to bend the host.
		for _, path := range []string{
			"fluxlab.evil.com/popup-jobber",
			"fluxlab/popup-jobber/../../evil",
			"evil.com%2F/popup-jobber",
			"/popup-jobber",
			"",
		} {
			t.Run(path, func(t *testing.T) {
				_, err := PopupStorefront(path, false)

				require.ErrorIs(t, err, ErrInvalidCheckoutPath)
			})
		}
	})

	t.Run("uses the canonical lowercase host", func(t *testing.T) {
		got, err := PopupStorefront("FluxLab/popup-Jobber", true)

		require.NoError(t, err)
		// Hostnames are case-insensitive; the checkout id is not, so it is left
		// exactly as the dashboard spells it.
		assert.Equal(t, "fluxlab.onfastspring.com/popup-Jobber", got)
	})
}
