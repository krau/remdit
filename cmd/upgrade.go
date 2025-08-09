package cmd

import (
	"fmt"

	"github.com/blang/semver"
	"github.com/krau/remdit/config"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
)

func Upgrade() error {
	v := semver.MustParse(config.Version)
	latest, err := selfupdate.UpdateSelf(v, "krau/remdit")
	if err != nil {
		return err
	}
	if latest.Version.Equals(v) {
		fmt.Println("You are already using the latest version:", v)
	} else {
		fmt.Printf("Successfully updated to version %s\n", latest.Version)
		fmt.Println("Release note:\n", latest.ReleaseNotes)
	}
	return nil
}
