package utils

import (
	"cydex"
	"errors"
	"fmt"
	"github.com/go-ini/ini"
	"strconv"
	"strings"
	"syscall"
)

const (
	CFG_SEP = ","
)

var (
	SizeStrPostfix = []string{"k", "m", "g"}
	SizeUnits      = []uint64{1024, 1024 * 1024, 1024 * 1024 * 1024}
	config         *Config
)

type Config struct {
	profile string
	cfgfile string
}

func NewConfig(profile, cfgfile string) *Config {
	o := &Config{
		profile, cfgfile,
	}
	return o
}

func (self *Config) SaveFileSlice(Switch bool) error {
	cfg, err := ini.LooseLoad(self.cfgfile)
	if err != nil {
		return err
	}
	sec := cfg.Section("pkg")

	_, err = sec.NewKey("file_slice", fmt.Sprint(Switch))
	if err != nil {
		return err
	}
	return cfg.SaveTo(self.cfgfile)
}

func (self *Config) SaveDefaultUnpackerArgs(min_seg_size uint64, max_seg_num uint) error {
	cfg, err := ini.LooseLoad(self.cfgfile)
	if err != nil {
		return err
	}

	sec := cfg.Section("default_unpacker")

	if _, err = sec.NewKey("min_seg_size", fmt.Sprint(min_seg_size)); err != nil {
		return err
	}
	if _, err = sec.NewKey("max_seg_cnt", fmt.Sprint(max_seg_num)); err != nil {
		return err
	}

	return cfg.SaveTo(self.cfgfile)
}

func (self *Config) Load() (*ini.File, error) {
	if err := syscall.Access(self.profile, syscall.F_OK); err != nil {
		return nil, fmt.Errorf("%s: %s", err.Error(), self.profile)
	}
	return ini.LooseLoad(self.profile, self.cfgfile)
}

func (self *Config) SaveEmailInfo(email *cydex.EmailInfo) error {
	cfg, err := ini.LooseLoad(self.cfgfile)
	if err != nil {
		return err
	}

	sec := cfg.Section("notification.email")
	if email.Enable != nil {
		if _, err = sec.NewKey("enable", fmt.Sprintf("%t", *email.Enable)); err != nil {
			return err
		}
	}
	if email.ContactName != nil {
		if _, err = sec.NewKey("contact_name", fmt.Sprintf("%s", *email.ContactName)); err != nil {
			return err
		}
	}
	if email.Language != nil {
		if _, err = sec.NewKey("language", fmt.Sprintf("%s", *email.Language)); err != nil {
			return err
		}
	}
	if email.SmtpServer != nil {
		if _, err = sec.NewKey("smtp_server", fmt.Sprintf("%s", *email.SmtpServer)); err != nil {
			return err
		}
	}

	return cfg.SaveTo(self.cfgfile)
}

func (self *Config) SaveEmailSmtpServer(v *cydex.EmailSmtpServer, label string) error {
	if v == nil {
		return errors.New("invalid param")
	}
	cfg, err := ini.LooseLoad(self.cfgfile)
	if err != nil {
		return err
	}

	sec := cfg.Section(fmt.Sprintf("smtp_server.%s", label))

	if _, err = sec.NewKey("host", v.Host); err != nil {
		return err
	}
	if _, err = sec.NewKey("port", fmt.Sprintf("%d", v.Port)); err != nil {
		return err
	}
	if _, err = sec.NewKey("account", v.Account); err != nil {
		return err
	}
	if _, err = sec.NewKey("password", v.Password); err != nil {
		return err
	}
	if _, err = sec.NewKey("use_tls", fmt.Sprintf("%t", v.UseTLS)); err != nil {
		return err
	}

	return cfg.SaveTo(self.cfgfile)
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

func MakeDefaultConfig(cfg *Config) {
	if cfg != nil {
		config = cfg
	}
}

func DefaultConfig() *Config {
	return config
}
