package lib

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"github.com/BurntSushi/toml"
)

// Config stores parse/report config including tracepoint, time section and so on.
type Config struct {
	Title string
	Parse struct {
		PrintFormat string
	}
	TracePoints struct {
		Enabled      []string
		CustomPoints [][2]string
	}
	Report struct {
		Read, Write, Others bool
		TimeSections        [][3]string
		Numbers             struct {
			Minimum, Maximum, Average, Median, Percentiles, Distribution bool
		}
	}
}

// NewConfig creates a config instance, parsing the config file f.
func NewConfig(f *os.File) *Config {
	var cfg = new(Config)
	var err error

	buf := new(bytes.Buffer)
	buf.ReadFrom(f)
	contents := buf.String()

	if _, err = toml.Decode(contents, cfg); err != nil {
		log.Fatal(err)
	}

	return cfg
}

// DefaultConfig creates a default config instance.
func DefaultConfig(f *os.File) *Config {
	var cfg = new(Config)
	return cfg
}

// TracePointsConfig -
type TracePointsConfig struct {
	Enabled      map[uint32]bool
	CustomPoints map[string]uint32
}

// ReportConfig -
type ReportConfig struct {
	Read, Write, Others bool
	TimeSections        map[string][2]uint32
	Numbers             struct {
		Minimum, Maximum, Average, Median, Percentiles, Distribution bool
	}
}

const startCustomPointNumber uint32 = 32

var nextCustomPointNumber = startCustomPointNumber

func getNewCustomPointNumber() uint32 {
	retNum := nextCustomPointNumber
	nextCustomPointNumber++
	return retNum
}

// NewTracePointsConfig - config that is used in processing
func NewTracePointsConfig(cfg *Config) *TracePointsConfig {
	var tpCfg = new(TracePointsConfig)

	tpCfg.Enabled = make(map[uint32]bool)
	tpCfg.CustomPoints = make(map[string]uint32)
	for i, v := range cfg.TracePoints.Enabled {
		tpCfg.Enabled[ActionsMap[v]] = true
		fmt.Println(i, v)
	}

	for i, v := range cfg.TracePoints.CustomPoints {
		tpCfg.CustomPoints["ytg:"+v[0]] = getNewCustomPointNumber()
		fmt.Println(i, v)
	}

	return tpCfg
}

// NewReportConfig - config that is used in processing
func NewReportConfig(cfg *Config, tpCfg *TracePointsConfig) *ReportConfig {
	var rCfg = new(ReportConfig)

	rCfg.Read = cfg.Report.Read
	rCfg.Write = cfg.Report.Write
	rCfg.Others = cfg.Report.Others
	rCfg.TimeSections = make(map[string][2]uint32)

	for i, v := range cfg.Report.TimeSections {
		var a, b uint32

		if u, ok := ActionsMap[v[1]]; ok {
			a = u
		} else if u, ok := tpCfg.CustomPoints["ytg:"+v[1]]; ok {
			a = u
		} else {

		}

		if u, ok := ActionsMap[v[2]]; ok {
			b = u
		} else if u, ok := tpCfg.CustomPoints["ytg:"+v[2]]; ok {
			b = u
		} else {

		}

		rCfg.TimeSections[v[0]] = [2]uint32{a, b}
		fmt.Println(i, v)
		fmt.Println(a, b)
	}

	return rCfg
}

// ActionsMap -
var ActionsMap = map[string]uint32{
	"queue":       TAQueue,
	"backmerge":   TABackmerge,
	"frontmerge":  TAFrontmerge,
	"getrq":       TAGetrq,
	"sleeprq":     TASleeprq,
	"requeue":     TARequeue,
	"issue":       TAIssue,
	"complete":    TAComplete,
	"plug":        TAPlug,
	"unplugio":    TAUnplugIO,
	"unplugtimer": TAUnplugTimer,
	"insert":      TAInsert,
	"split":       TASplit,
	"bounce":      TABounce,
	"remap":       TARemap,
	"abort":       TAAbort,
	"drvdata":     TADrvData,
}
