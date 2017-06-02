package lib

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
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

// Report reads/parses blktrace records and collects statistics data.
func Report(input *bufio.Reader, output *bufio.Writer, cfg *os.File) {
	var err error
	var r *BlktraceRecord

	readStats := NewBlktraceStatistics()
	writeStats := NewBlktraceStatistics()

	/*
		TODO: Implement config based report
		[ ] check config
		[ ] create struct for stat collecting
		[ ] process blktrace records
		[ ] calulate additional numbers
		[ ] export to csv/json
	*/

	recordsToPrint := make(chan *BlktraceRecord, 8192)
	printFinished := make(chan int)

	go printRecords(recordsToPrint, printFinished)

	readRecords := make(chan *BlktraceRecord, 1024)
	writeRecords := make(chan *BlktraceRecord, 1024)
	var wg sync.WaitGroup

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
			// readStats.AddRecord(r)
			readRecords <- r
		} else if (r.Action & TCWrite) != 0 {
			// writeStats.AddRecord(r)
			writeRecords <- r
		} else { // others
			recordsToPrint <- r
		}
	}

	// Finish all goroutines
	readRecords <- nil
	writeRecords <- nil
	recordsToPrint <- nil
	<-printFinished
	wg.Wait()

	fmt.Print("\n\n\n")
	fmt.Println("yabtar_read_stat:", readStats.String())
	fmt.Println("yabtar_write_stat:", writeStats.String())
	fmt.Print("\n\n\n")

	jData := BlktraceResult{}

	jData.R.Min = readStats.minimums
	jData.R.Max = readStats.maximums
	jData.R.Mean = readStats.GetAvg()

	jData.W.Min = writeStats.minimums
	jData.W.Max = writeStats.maximums
	jData.W.Mean = writeStats.GetAvg()

	j, err := json.Marshal(jData)

	if output != nil {
		_, err = output.Write(j)
		output.Flush()
	} else {
		fmt.Println(string(j))
	}

	stats := []uint64{
		readStats.totals["DRV-Q"],
		readStats.totals["C-DRV"],
		writeStats.totals["DRV-Q"],
		writeStats.totals["C-DRV"],
	}

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
	traceBatches map[uint64]map[string]*BlktraceRecord
	numBatches   uint64

	totals   map[string]uint64
	minimums map[string]uint64
	maximums map[string]uint64
}

// NewBlktraceStatistics -
func NewBlktraceStatistics() *BlktraceStatistics {
	newObj := BlktraceStatistics{numBatches: 0}

	newObj.traceBatches = make(map[uint64]map[string]*BlktraceRecord)

	newObj.totals = make(map[string]uint64)
	newObj.minimums = make(map[string]uint64)
	newObj.maximums = make(map[string]uint64)

	newObj.totals["DRV-Q"] = 0
	newObj.totals["C-DRV"] = 0
	newObj.minimums["DRV-Q"] = 0
	newObj.minimums["C-DRV"] = 0
	newObj.maximums["DRV-Q"] = 0
	newObj.maximums["C-DRV"] = 0
	return &newObj
}

// AddRecord is
func (s *BlktraceStatistics) AddRecord(r *BlktraceRecord) {
	var a string

	a = func() string {
		switch r.Action & 0x0000FFFF {
		case TAQueue:
			return "Q"
		case TADrvData:
			return "DRV"
		case TAComplete:
			return "C"
		default:
			return "?"
		}
	}()

	if _, ok := s.traceBatches[r.Sector]; !ok || a == "Q" {
		s.traceBatches[r.Sector] = make(map[string]*BlktraceRecord)
	}

	// FIXME: Hardcoded action lists.
	// FIXME: Move bin/str action representation into a method
	if !(a == "?") {
		s.traceBatches[r.Sector][a] = r
	}

	var rGroup map[string]*BlktraceRecord
	rGroup = s.traceBatches[r.Sector]

	// FIXME: Hardcoded action lists.
	var ready bool
	ready = func() bool { // check if all of three are collected
		for _, k := range []string{"Q", "DRV", "C"} {
			if _, ok := rGroup[k]; !ok {
				return false
			}
		}
		return true
	}()

	if ready {
		drvToQ := rGroup["DRV"].Time - rGroup["Q"].Time
		cToDrv := rGroup["C"].Time - rGroup["DRV"].Time

		if drvToQ < 0 {
			fmt.Printf("Warning: minus!! %d", drvToQ)
			fmt.Printf(r.String())
		}

		if cToDrv < 0 {
			fmt.Printf("Warning: minus!! %d", cToDrv)
			fmt.Printf(r.String())
		}

		s.totals["DRV-Q"] += drvToQ
		s.totals["C-DRV"] += cToDrv

		s.numBatches++

		// #FIXME: Hardcoded action lists.
		if (s.minimums["DRV-Q"] == 0) || (s.minimums["DRV-Q"] > drvToQ) {
			s.minimums["DRV-Q"] = drvToQ
		}

		if (s.minimums["C-DRV"] == 0) || (s.minimums["C-DRV"] > cToDrv) {
			s.minimums["C-DRV"] = cToDrv
		}

		if (s.maximums["DRV-Q"] == 0) || (s.maximums["DRV-Q"] < drvToQ) {
			s.maximums["DRV-Q"] = drvToQ
		}

		if (s.maximums["C-DRV"] == 0) || (s.maximums["C-DRV"] < cToDrv) {
			s.maximums["C-DRV"] = cToDrv
		}

		delete(s.traceBatches, r.Sector)
	}
}

// GetAvg is
func (s *BlktraceStatistics) GetAvg() map[string]float64 {
	var avgDrvToQ, avgCToDrv float64

	avgs := make(map[string]float64)
	cnt := s.numBatches

	if cnt > 0 {
		avgDrvToQ = float64(s.totals["DRV-Q"]) / float64(cnt)
		avgCToDrv = float64(s.totals["C-DRV"]) / float64(cnt)
	} else {
		avgDrvToQ, avgCToDrv = 0, 0
	}

	avgs["DRV-Q"] = avgDrvToQ
	avgs["C-DRV"] = avgCToDrv
	return avgs
}

// String is
func (s *BlktraceStatistics) String() string {
	var avgDrvToQ, avgCToDrv float64

	if s.numBatches > 0 {
		avgDrvToQ = float64(s.totals["DRV-Q"]) / float64(s.numBatches)
		avgCToDrv = float64(s.totals["C-DRV"]) / float64(s.numBatches)

		// TODO: rewrite this ugly statement
		return fmt.Sprintf("BlktraceStatistics: cnt=%d\n  avg DRV-Q=%fus C-DRV=%fus\n  min DRV-Q=%fus C-DRV=%fus\n  max DRV-Q=%fus C-DRV=%fus",
			s.numBatches, float64(avgDrvToQ)/1000.0, float64(avgCToDrv)/1000.0, float64(s.minimums["DRV-Q"])/1000.0, float64(s.minimums["C-DRV"])/1000.0, float64(s.maximums["DRV-Q"])/1000.0, float64(s.maximums["C-DRV"])/1000.0)
	}

	avgDrvToQ, avgCToDrv = 0, 0
	return fmt.Sprintf("BlktraceStatistics: cnt=%d\n  Nothing is collected", s.numBatches)
}
