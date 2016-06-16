package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type MinMaxMean struct {
	Min  map[string]uint64  `json:"min"`
	Max  map[string]uint64  `json:"max"`
	Mean map[string]float64 `json:"mean"`
}

type BlktraceResult struct {
	R MinMaxMean
	W MinMaxMean
}

type BlktraceRecord struct {
	// Magic                        uint32
	Seq                          uint32
	Time, Sector                 uint64
	Bytes, Action, Pid, Dev, Cpu uint32
	Err, Pdu_len                 uint16

	Pdu_data string
}

func (r *BlktraceRecord) ToString() string {
	return fmt.Sprintf("BlktraceRecord:%d %d %d %d 0x%08x %d %d %d %d %d %s",
		r.Seq, r.Time, r.Cpu, r.Pid, r.Action,
		r.Dev, r.Sector, r.Bytes, r.Err, r.Pdu_len, r.Pdu_data)
	/* TODO: replace non-readable chars from r.Pdu_data /[\x00-\x08\x0A-\x1F\x7F]/ */
}

type BlktraceStatistics struct {
	trace_batches map[uint64]map[string]*BlktraceRecord
	num_batches   uint64

	totals   map[string]uint64
	minimums map[string]uint64
	maximums map[string]uint64
}

func NewBlktraceStatistics() *BlktraceStatistics {
	newObj := BlktraceStatistics{num_batches: 0}

	newObj.trace_batches = make(map[uint64]map[string]*BlktraceRecord)

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

func (s *BlktraceStatistics) Add_record(r *BlktraceRecord) {
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

	if _, ok := s.trace_batches[r.Sector]; !ok || a == "Q" {
		s.trace_batches[r.Sector] = make(map[string]*BlktraceRecord)
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
		s.trace_batches[r.Sector][a] = r
	}

	var rGroup map[string]*BlktraceRecord
	rGroup = s.trace_batches[r.Sector]

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
		drv_q := rGroup["DRV"].Time - rGroup["Q"].Time
		c_drv := rGroup["C"].Time - rGroup["DRV"].Time

		if drv_q < 0 {
			fmt.Printf("Warning: minus!! %d", drv_q)
			fmt.Printf(r.ToString())
		}

		if c_drv < 0 {
			fmt.Printf("Warning: minus!! %d", c_drv)
			fmt.Printf(r.ToString())
		}

		s.totals["DRV-Q"] += drv_q
		s.totals["C-DRV"] += c_drv

		s.num_batches += 1

		// #FIXME: Hardcoded action lists.
		if (s.minimums["DRV-Q"] == 0) || (s.minimums["DRV-Q"] > drv_q) {
			s.minimums["DRV-Q"] = drv_q
		}

		if (s.minimums["C-DRV"] == 0) || (s.minimums["C-DRV"] > c_drv) {
			s.minimums["C-DRV"] = c_drv
		}

		if (s.maximums["DRV-Q"] == 0) || (s.maximums["DRV-Q"] < drv_q) {
			s.maximums["DRV-Q"] = drv_q
		}

		if (s.maximums["C-DRV"] == 0) || (s.maximums["C-DRV"] < c_drv) {
			s.maximums["C-DRV"] = c_drv
		}

		delete(s.trace_batches, r.Sector)
	}
}

func (s *BlktraceStatistics) GetAvg() map[string]float64 {
	var avg_drv_q, avg_c_drv float64

	avgs := make(map[string]float64)
	cnt := s.num_batches

	if cnt > 0 {
		avg_drv_q = float64(s.totals["DRV-Q"]) / float64(cnt)
		avg_c_drv = float64(s.totals["C-DRV"]) / float64(cnt)
	} else {
		avg_drv_q, avg_c_drv = 0, 0
	}

	avgs["DRV-Q"] = avg_drv_q
	avgs["C-DRV"] = avg_c_drv
	return avgs
}

func (s *BlktraceStatistics) ToString() string {
	cnt := s.num_batches
	var avg_drv_q, avg_c_drv float64

	if cnt > 0 {
		avg_drv_q = float64(s.totals["DRV-Q"]) / float64(cnt)
		avg_c_drv = float64(s.totals["C-DRV"]) / float64(cnt)
	} else {
		avg_drv_q, avg_c_drv = 0, 0
	}

	if s.num_batches > 0 {
		// TODO: rewrite this ugly statement
		return fmt.Sprintf("BlktraceStatistics: cnt=%d\n  avg DRV-Q=%fus C-DRV=%fus\n  min DRV-Q=%fus C-DRV=%fus\n  max DRV-Q=%fus C-DRV=%fus",
			cnt, float64(avg_drv_q)/1000.0, float64(avg_c_drv)/1000.0, float64(s.minimums["DRV-Q"])/1000.0, float64(s.minimums["C-DRV"])/1000.0, float64(s.maximums["DRV-Q"])/1000.0, float64(s.maximums["C-DRV"])/1000.0)
	} else {
		return fmt.Sprintf("BlktraceStatistics: cnt=%d\n  Nothing is collected", cnt)
	}
}

func Read_and_parse_one_record(reader io.Reader) (*BlktraceRecord, error) {
	// # # # # # #
	// # Reference:
	// #   struct blk
	// #    - definition is in include/uapi/linux/blktrace_api.h
	// #
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

	l := 8*2 + 4*7 + 2*2
	readN(l)

	st := 0

	_ = u32le(buf[st : st+4]) // d_magic
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

	r.Cpu = u32le(buf[st : st+4])
	st += 4

	r.Err = u16le(buf[st : st+2])
	st += 2

	r.Pdu_len = u16le(buf[st : st+2])
	st += 2

	if r.Pdu_len > 0 {
		r.Pdu_data = string(readN(int(r.Pdu_len)))
	} else {
		r.Pdu_data = ""
	}

	return r, err
}

//#################
//# main program flow starts here
func main() {
	var err error = nil

	// Signal.trap("PIPE", "EXIT")

	readStats := NewBlktraceStatistics()
	writeStats := NewBlktraceStatistics()

	f_in, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer f_in.Close()

	reader := bufio.NewReader(f_in)

	for true {
		err = nil
		r, err := Read_and_parse_one_record(reader)

		if !(err == nil) {
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
			readStats.Add_record(r)
		} else if rw == "W" {
			writeStats.Add_record(r)
		} else {
			fmt.Println(r.ToString())
		}
	}

	fmt.Println("\n\n")
	fmt.Println("yabtar_read_stat:", readStats.ToString())
	fmt.Println("yabtar_write_stat:", writeStats.ToString())
	fmt.Println("\n\n")

	f_out, err := os.OpenFile(os.Args[2], os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer f_out.Close()
	writer := bufio.NewWriter(f_out)

	jData := BlktraceResult{}

	jData.R.Min = readStats.minimums
	jData.R.Max = readStats.maximums
	jData.R.Mean = readStats.GetAvg()

	jData.W.Min = writeStats.minimums
	jData.W.Max = writeStats.maximums
	jData.W.Mean = writeStats.GetAvg()

	j, err := json.Marshal(jData)
	fmt.Println(string(j))

	_, err = writer.Write(j)
	writer.Flush()
}
