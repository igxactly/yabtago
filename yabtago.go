package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/alexflint/go-arg"
)

// BlktraceRecord represents a blktrace record.
type BlktraceRecord struct {
	// Magic                        uint32

	Seq                          uint32
	Time, Sector                 uint64
	Bytes, Action, Pid, Dev, CPU uint32
	Err, PduLen                  uint16

	PduData string
}

// String -
func (r *BlktraceRecord) String() string {
	return fmt.Sprintf("BlktraceRecord:%d %d %d %d 0x%08x %d %d %d %d %d %s",
		r.Seq, r.Time, r.CPU, r.Pid, r.Action,
		r.Dev, r.Sector, r.Bytes, r.Err, r.PduLen, r.PduData)

	/* TODO: replace non-readable chars from r.Pdu_data /[\x00-\x08\x0A-\x1F\x7F]/ */
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
		case 0x0001:
			return "Q"
		case 0x0011:
			return "DRV"
		case 0x0008:
			return "C"
		default:
			return "?"
		}
	}()

	if _, ok := s.traceBatches[r.Sector]; !ok || a == "Q" {
		s.traceBatches[r.Sector] = make(map[string]*BlktraceRecord)
	}

	// enum blktrace_act {
	//     __BLK_TA_QUEUE = 1,     /* queued */
	//     __BLK_TA_BACKMERGE,     /* back merged to existing rq */
	//     __BLK_TA_FRONTMERGE,        /* front merge to existing rq */
	//     __BLK_TA_GETRQ,         /* allocated new request */
	//     __BLK_TA_SLEEPRQ,       /* sleeping on rq allocation */
	//     __BLK_TA_REQUEUE,       /* request requeued */
	//     __BLK_TA_ISSUE,         /* sent to driver */
	//     __BLK_TA_COMPLETE,      /* completed by driver */
	//     __BLK_TA_PLUG,          /* queue was plugged */
	//     __BLK_TA_UNPLUG_IO,     /* queue was unplugged by io */
	//     __BLK_TA_UNPLUG_TIMER,      /* queue was unplugged by timer */
	//     __BLK_TA_INSERT,        /* insert request */
	//     __BLK_TA_SPLIT,         /* bio was split */
	//     __BLK_TA_BOUNCE,        /* bio was bounced */
	//     __BLK_TA_REMAP,         /* bio was remapped */
	//     __BLK_TA_ABORT,         /* request aborted */
	//     __BLK_TA_DRV_DATA,      /* driver-specific binary data */
	// };

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

// ReadBlktraceRecord reads one blktrace record from reader and create/return a BlktraceRecord.
func ReadBlktraceRecord(reader io.Reader) (*BlktraceRecord, error) {
	// Reference: blktrace record structure
	//     - definition from blktrace_api.h
	//
	// blktrace log file is written in this structure.
	//
	// # __u32 magic;        /* MAGIC << 8 | version */
	// # __u32 sequence;     /* event number */
	// # __u64 time;     /* in microseconds */ ??? it seems it is not US but NS
	// # __u64 sector;       /* disk offset */
	// # __u32 bytes;        /* transfer length */
	// # __u32 action;       /* what happened */
	// # __u32 pid;      /* who did it */
	// # __u32 device;       /* device number */
	// # __u32 cpu;      /* on what cpu did it happen */
	// # __u16 error;        /* completion error */
	// # __u16 pdu_len;      /* length of data after this trace */
	//
	// and PDU data follows.

	r := new(BlktraceRecord)

	var err error
	var buf = make([]byte, 1024)

	u16le := binary.LittleEndian.Uint16
	u32le := binary.LittleEndian.Uint32
	u64le := binary.LittleEndian.Uint64

	readN := func(n int) []byte {
		_, err = io.ReadFull(reader, buf[0:n])
		return buf[0:n]
	}

	// Read a blktrace record, excluding pdu_data
	l := 8*2 + 4*7 + 2*2 // 64bit*2, 32bit*7, 16bit*2
	readN(l)

	// assign each fields
	st := 0
	_ = u32le(buf[st : st+4]) // omit d_magic
	st += 4
	r.Seq = u32le(buf[st : st+4])
	st += 4
	r.Time = u64le(buf[st : st+8])
	st += 8
	r.Sector = u64le(buf[st : st+8])
	st += 8
	r.Bytes = u32le(buf[st : st+4])
	st += 4
	r.Action = u32le(buf[st : st+4])
	st += 4
	r.Pid = u32le(buf[st : st+4])
	st += 4
	r.Dev = u32le(buf[st : st+4])
	st += 4
	r.CPU = u32le(buf[st : st+4])
	st += 4
	r.Err = u16le(buf[st : st+2])
	st += 2
	r.PduLen = u16le(buf[st : st+2])
	st += 2

	if r.PduLen > 0 {
		// TODO: handle non-char bytes
		r.PduData = string(readN(int(r.PduLen)))
	} else {
		r.PduData = ""
	}

	return r, err
}

func parse(input *bufio.Reader, output *bufio.Writer) {
	var err error
	var r *BlktraceRecord

	for true {
		err = nil
		r, err = ReadBlktraceRecord(input)

		if !(err == nil) {
			// TODO: check of any error occured while reading record.
			// if err is not eof error {
			// 	fmt.Println("parse(): something went wrong while reading blktrace record")
			// }
			break
		}

		fmt.Println(r.String())
	}
}

func report(input *bufio.Reader, output *bufio.Writer) {
	var err error
	var r *BlktraceRecord

	readStats := NewBlktraceStatistics()
	writeStats := NewBlktraceStatistics()

	for true {
		err = nil
		r, err = ReadBlktraceRecord(input)

		if !(err == nil) {
			// TODO: check of any error occured while reading record.
			break
		}

		rw := func() string {
			switch r.Action & 0x00030000 {
			case 0x00010000:
				return "R"
			case 0x00020000:
				return "W"
			default:
				return "?"
			}
		}()

		if rw == "R" {
			readStats.AddRecord(r)
		} else if rw == "W" {
			writeStats.AddRecord(r)
		} else {
			fmt.Println(r.String())
		}
	}

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

	// fmt.Println(readStats.totals)
	// fmt.Println(writeStats.totals)

	fmt.Print(readStats.totals["DRV-Q"])
	fmt.Print(", ")
	fmt.Print(readStats.totals["C-DRV"])

	fmt.Print(", ")

	fmt.Print(writeStats.totals["DRV-Q"])
	fmt.Print(", ")
	fmt.Print(writeStats.totals["C-DRV"])
}

//#################
//# main program flow starts here
func main() {
	// program arguments declaration
	var args struct {
		Action       string `arg:"positional,required,help:parse or report"`
		Input        string `arg:"positional,required,help:specify the input blktrace file"`
		Output       string `arg:"-o"`
		OutFormat    string `arg:"--oformat,-f"`
		ReportConfig string `arg:"-r"`
	}

	// default argument values
	args.OutFormat = "default"

	// parse arguments
	arg.MustParse(&args)
	fmt.Printf("%+v\n", args)

	// TODO: try to open in/output files
	fileInput, err := os.Open(args.Input)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer fileInput.Close()
	reader := bufio.NewReader(fileInput)

	var writer *bufio.Writer
	if args.Output != "" {
		fileOutput, err := os.OpenFile(args.Output, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer fileOutput.Close()
		writer = bufio.NewWriter(fileOutput)
	}

	switch args.Action {
	case "parse":
		parse(reader, writer)
		break
	case "report":
		// TODO: Check report config and than go on
		report(reader, writer)
		break
	default:
		println("main(): seems like something went wrong!!!")
	}
}
