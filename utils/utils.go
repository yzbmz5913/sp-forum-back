package utils

import (
	regexp "github.com/dlclark/regexp2"
	"time"
)

var reg = regexp.MustCompile("^(?![0-9]+$)(?![a-zA-Z]+$)[0-9A-Za-z]{8,18}$", 0)

func CheckPassword(pwd string) bool {
	match, _ := reg.MatchString(pwd)
	return match
}

func Retry(try int, wait time.Duration, f func() error) error {
	if err := f(); err != nil {
		if try--; try >= 0 {
			time.Sleep(wait)
			return Retry(try, wait, f)
		}
		return err
	}
	return nil
}
