package utils

import (
	"github.com/go-ini/ini"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func Test_Config(t *testing.T) {
	Convey("Test Set", t, func() {

		Convey("initest", func() {
			k := &Config{" ", "/tmp/ts.ini"} //note:you can set ts.ini 's location
			k.SaveDefaultUnpackerArgs(40*1024, 21)
			k.SaveFileSlice(true)
			cfg, _ := ini.LooseLoad(k.cfgfile)

			val := cfg.Section("pkg").Key("file_slice").MustBool()
			val2 := cfg.Section("default_unpacker").Key("min_seg_size").MustUint64(0)
			val3 := cfg.Section("default_unpacker").Key("max_seg_cnt").MustUint(0)
			So(val, ShouldBeTrue)
			So(val2, ShouldEqual, 40*1024)
			So(val3, ShouldEqual, 21)
		})

	})
}
