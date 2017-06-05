package lib

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
)

func collectRecord(wg *sync.WaitGroup, records chan *BlktraceRecord, stats *BlktraceStatistics) {
	defer wg.Done()

	for true {
		r := <-records
		if r == nil {
			break
		} else {
			stats.AddRecord(r)
		}
	}
}

var tpConfig *TracePointsConfig
var rConfig *ReportConfig

// Report reads/parses blktrace records and collects statistics data.
func Report(input *bufio.Reader, output *bufio.Writer, cfg *Config, outFmt string) {
	var err error
	var r *BlktraceRecord

	tpConfig = NewTracePointsConfig(cfg)
	rConfig = NewReportConfig(cfg, tpConfig)
	// log.Printf("%+v\n", cfg)
	// log.Printf("%+v\n", tpConfig)
	// log.Printf("%+v\n", rConfig)

	readStats := NewBlktraceStatistics(rConfig)
	writeStats := NewBlktraceStatistics(rConfig)

	/*
		TODO: Implement config based report
		[v] check config
		[v] create struct for stat collecting
		[v] process blktrace records
		[v] calulate additional numbers
		[ ] export to csv/json
	*/

	recordsToPrint := make(chan *BlktraceRecord, 8192)
	printFinished := make(chan int)
	go printRecords(recordsToPrint, printFinished)

	var wg sync.WaitGroup
	readRecords := make(chan *BlktraceRecord, 1024)
	writeRecords := make(chan *BlktraceRecord, 1024)
	wg.Add(1)
	go collectRecord(&wg, readRecords, readStats)
	wg.Add(1)
	go collectRecord(&wg, writeRecords, writeStats)

	for true {
		err = nil
		r, err = ReadBlktraceRecord(input)

		if !(err == nil) {
			// TODO: check of any error occured while reading record.
			break
		}

		if (r.Action & TCRead) != 0 {
			readRecords <- r
		} else if (r.Action & TCWrite) != 0 {
			writeRecords <- r
		} else { // others
			// recordsToPrint <- r
		}
	}

	// Finish all goroutines
	readRecords <- nil
	writeRecords <- nil
	recordsToPrint <- nil
	<-printFinished
	wg.Wait()

	// TODO: Rewrite report printout code
	// Print report
	if outFmt == "default" {
		fmt.Println("yabtar_read_stat:", readStats.String())
		fmt.Println("yabtar_write_stat:", writeStats.String())
	} else if outFmt == "json" {
		// JSON
		jData := BlktraceResult{}

		jData.R.Min = readStats.minimums
		jData.R.Max = readStats.maximums
		jData.R.Mean = readStats.GetAvg()

		jData.W.Min = writeStats.minimums
		jData.W.Max = writeStats.maximums
		jData.W.Mean = writeStats.GetAvg()

		j, err := json.Marshal(jData)

		if err != nil {
			log.Println("JSON marshaling problem")
		}
		if output != nil {
			_, err = output.Write(j)
			output.Flush()
		} else {
			fmt.Println(string(j))
		}
	} else if outFmt == "csv" {
		// CSV
		// TODO: Not just CSV but full data output!!
		timeSect := rConfig.TimeSections
		stats := []uint64{}

		addTotalsToList := func(bStat *BlktraceStatistics) {
			for k := range timeSect {
				stats = append(stats, bStat.totals[k])
			}
		}
		addTotalsToList(readStats)
		addTotalsToList(writeStats)

		statsStr := func() []string {
			var sList []string
			for _, u := range stats {
				sList = append(sList, strconv.FormatUint(u, 10))
			}
			return sList
		}()

		w := csv.NewWriter(os.Stdout)
		if err = w.Write(statsStr); err != nil {
			log.Fatalln("error writing record to csv:", err)
		}
		w.Flush()
	}
}

// MinMaxMean -
type MinMaxMean struct {
	Min  map[string]uint64  `json:"min"`
	Max  map[string]uint64  `json:"max"`
	Mean map[string]float64 `json:"mean"`
}

// BlktraceResult -
type BlktraceResult struct {
	R MinMaxMean
	W MinMaxMean
}

// BlktraceStatistics -
type BlktraceStatistics struct {
	traceBatches map[uint64]map[uint32]*BlktraceRecord
	numBatches   uint64

	totals   map[string]uint64
	minimums map[string]uint64
	maximums map[string]uint64
}

// NewBlktraceStatistics -
func NewBlktraceStatistics(rCfg *ReportConfig) *BlktraceStatistics {
	newObj := BlktraceStatistics{numBatches: 0}

	newObj.traceBatches = make(map[uint64]map[uint32]*BlktraceRecord)

	newObj.totals = make(map[string]uint64)
	newObj.minimums = make(map[string]uint64)
	newObj.maximums = make(map[string]uint64)

	for k := range rCfg.TimeSections {
		newObj.totals[k] = 0
		newObj.minimums[k] = 0
		newObj.maximums[k] = 0
	}
	return &newObj
}

// AddRecord is
func (s *BlktraceStatistics) AddRecord(r *BlktraceRecord) {
	var a uint32
	var drvDataExists int

	enabledTP := tpConfig.Enabled
	customTP := tpConfig.CustomPoints
	timeSect := rConfig.TimeSections

	if _, ok := enabledTP[TADrvData]; ok {
		drvDataExists = 1
	}

	a = r.Action & 0x0000FFFF

	// First record for sector. Replace if TAQueue? --> FIXME: Hardcoded?
	if _, ok := s.traceBatches[r.Sector]; !ok || a == TAQueue {
		s.traceBatches[r.Sector] = make(map[uint32]*BlktraceRecord)
	}
	if _, ok := enabledTP[a]; ok {
		if a == TADrvData { // DrvData based custom tracing points
			p := strings.Trim(r.PduData, "\x00")

			c, ok := customTP[p] // replace drvdata code with custom TP's code
			if ok {
				a = c
			} else {
				log.Printf("Undefined custom tracing point! Data: %s", p)
			}
		}
		s.traceBatches[r.Sector][a] = r
	}

	var rGroup map[uint32]*BlktraceRecord
	rGroup = s.traceBatches[r.Sector]

	// fmt.Println(len(enabledTracepoints), len(rGroup))
	// fmt.Printf("%+v\n", enabledTracepoints)
	var ready bool
	if len(rGroup) == (len(enabledTP) - drvDataExists + len(customTP)) {
		ready = true
	}

	if ready {
		s.numBatches++

		for k := range timeSect {
			timeDiff := rGroup[timeSect[k][1]].Time - rGroup[timeSect[k][0]].Time

			if timeDiff < 0 {
				log.Printf("Warning: minus!! %d", timeDiff)
			}

			s.totals[k] += timeDiff

			// #FIXME: Hardcoded stat number field
			if (s.minimums[k] == 0) || (s.minimums[k] > timeDiff) {
				s.minimums[k] = timeDiff
			}
			if (s.maximums[k] == 0) || (s.maximums[k] > timeDiff) {
				s.maximums[k] = timeDiff
			}
		}

		delete(s.traceBatches, r.Sector)
	}
}

// GetAvg is
func (s *BlktraceStatistics) GetAvg() map[string]float64 {
	timeSect := rConfig.TimeSections

	avgs := make(map[string]float64)
	cnt := s.numBatches

	for k := range timeSect {
		if cnt > 0 {
			avgs[k] = float64(s.totals[k]) / float64(cnt)
		} else {
			avgs[k] = 0
		}
	}

	return avgs
}

// String is
func (s *BlktraceStatistics) String() string {
	timeSect := rConfig.TimeSections

	// TODO: Hardcoded stat numbers
	var retString string
	if s.numBatches > 0 {

		avgs := s.GetAvg()
		mins := s.minimums
		maxs := s.maximums

		retString = fmt.Sprintf("BlktraceStatistics: cnt=%d", s.numBatches)
		for k := range timeSect {
			avgUs := float64(avgs[k]) / 1000.0
			minUs := float64(mins[k]) / 1000.0
			maxUs := float64(maxs[k]) / 1000.0
			retString += fmt.Sprintf("\n  %s avg=%9.3fus min=%9.3fus max=%9.3fus", k, avgUs, minUs, maxUs)
		}
	} else {
		retString = fmt.Sprintf("BlktraceStatistics: cnt=%d\n  Nothing is collected", s.numBatches)
	}

	return retString
}
