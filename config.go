package main

import (
	"fmt"
	"github.com/go-ini/ini"
	"strconv"
	"strings"
	"syscall"
)

const (
	CFG_SEP = ","
	PROFILE = "/opt/cydex/etc/ts.d/profile.ini"
	CFGFILE = "/opt/cydex/config/ts.ini"
)

var (
	SizeStrPostfix = []string{"k", "m", "g"}
	SizeUnits      = []uint64{1024, 1024 * 1024, 1024 * 1024 * 1024}
)

var seps = []string{"k", "m", "g"}

func LoadConfig() (*ini.File, error) {
	if err := syscall.Access(PROFILE, syscall.F_OK); err != nil {
		return nil, fmt.Errorf("%s: %s", err.Error(), PROFILE)
	}
	return ini.LooseLoad(PROFILE, CFGFILE)
}

// sizestr, 例如100, 100K/k, 100M/m, 100G/g等
func ConfigGetSize(sizestr string) uint64 {
	if sizestr == "" {
		return 0
	}

	sep_idx := -1
	lower_str := strings.ToLower(sizestr)
	for i, sep := range SizeStrPostfix {
		r := strings.Index(lower_str, sep)
		if r != -1 {
			// 字母必须是最后
			if r != len(lower_str)-1 {
				return 0
			}
			lower_str = lower_str[:r]
			sep_idx = i
			break
		}
	}
	n, err := strconv.ParseUint(lower_str, 10, 64)
	if err != nil {
		return 0
	}
	if sep_idx >= 0 {
		n = n * SizeUnits[sep_idx]
	}
	return n
}
