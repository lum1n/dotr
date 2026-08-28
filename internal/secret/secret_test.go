package secret_test

import (
	"testing"

	"github.com/vegarringdal/dotr/internal/secret"
)

func TestPath(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/home/u/.ssh/id_ed25519", true},
		{"/home/u/.ssh/id_ed25519.pub", false},
		{"/home/u/.config/chatty/secrets.json", true},
		{"/home/u/.netrc", true},
		{"/home/u/.config/alacritty/alacritty.toml", false},
		{"/home/u/.config/foo/credentials", true},
		{"/home/u/.config/app/.env", true},
	}
	for _, tc := range cases {
		if got := secret.Path(tc.path); got != tc.want {
			t.Errorf("%s: got %v want %v", tc.path, got, tc.want)
		}
	}
}
