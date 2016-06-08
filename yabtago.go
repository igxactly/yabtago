package main

// require 'json'
import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type BlktraceRecord struct {
	// Magic                        uint32
	Seq                          uint32
	Time, Sector                 uint64
	Bytes, Action, Pid, Dev, Cpu uint32
	Err, Pdu_len                 uint16

	Pdu_data string
}

func (r *BlktraceRecord) to_s() string {
	return fmt.Sprintf("BlktraceRecord:%d %d %d %d 0x%08x %d %d %d %d %d %s",
		r.Seq, r.Time, r.Cpu, r.Pid, r.Action,
		r.Dev, r.Sector, r.Bytes, r.Err, r.Pdu_len, r.Pdu_data)
	//.gsub(/[\x00-\x08\x0A-\x1F\x7F]/, ' ')
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

func test() int32 {

	return 0
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

	// puts @trace_batches, "\n\n"
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

			// puts "Warning: minus!! %d" % drv_q
			// puts r
		}

		if c_drv < 0 {
			fmt.Printf("Warning: minus!! %d", c_drv)

			// puts r
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

func (s *BlktraceStatistics) Get_averages() map[string]float64 {
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

//TODO: rewrite to_s()

// func (s *BlktraceStatistics) to_s() {
//    cnt = @num_batches
//    avg_drv_q = @totals['DRV-Q'].to_f / cnt
//    avg_c_drv = @totals['C-DRV'].to_f / cnt
//
//    if @num_batches > 0
//        return "BlktraceStatistics: cnt=%u\n  avg DRV-Q=%fus C-DRV=%fus\n  min DRV-Q=%fus C-DRV=%fus\n  max DRV-Q=%fus C-DRV=%fus" %
//            ([cnt] + [avg_drv_q, avg_c_drv, @minimums['DRV-Q'], @minimums['C-DRV'], @maximums['DRV-Q'], @maximums['C-DRV']].map{|x| (x / 1000)})
//    else
//        return "BlktraceStatistics: cnt=%u\n  Nothing is collected" % [cnt]
//    end
//}

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

	// u16be := binary.BigEndian.Uint16
	// u32be := binary.BigEndian.Uint32
	// u64be := binary.BigEndian.Uint64

	readN := func(n int) []byte {
		//var l int
		//_, err = reader.Read(buf[0:n])
		_, err = io.ReadFull(reader, buf[0:n])
		// fmt.Println(l, err)
		return buf[0:n]
	}

	l := 8*2 + 4*7 + 2*2
	readN(l)
	/*
		_, err = io.ReadFull(reader, buf[0:l])
		//_, err = reader.Read(buf[0:l])
		for i := 0; i < l; i += 1 {
			switch i % 2 {
			case 0:
				fmt.Printf("%02x", buf[i])
				break
			case 1:
				fmt.Printf("%02x ", buf[i])
				break
			}
		}
		fmt.Println()
	*/

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

	// var count int = 0

	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer f.Close()

	reader := bufio.NewReader(f)

	for true {
		err = nil
		r, err := Read_and_parse_one_record(reader)

		if !(err == nil) {
			break
		}
		// fmt.Println(r)
		fmt.Println(r.to_s())

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
		}
		// else {
		// }
	}

	// puts "\n\n"
	// puts "yabtar_read_stat:", read_statistics
	// puts "yabtar_write_stat:", write_statistics
	//
	// puts "\n\n"
	// File.open(ARGV[1], "w") do |f|
	// # "total"=>statistics.instance_variable_get(:@totals),
	//     t = JSON.generate(
	//         {
	//             "R"=>
	//             {
	//                 "min"=>read_statistics.instance_variable_get(:@minimums),
	//                 "max"=>read_statistics.instance_variable_get(:@maximums),
	//                 "mean"=>read_statistics.get_averages
	//             },
	//             "W"=>
	//             {
	//                 "min"=>write_statistics.instance_variable_get(:@minimums),
	//                 "max"=>write_statistics.instance_variable_get(:@maximums),
	//                 "mean"=>write_statistics.get_averages
	//             }
	//
	//         })
	//     puts 'json: ', t
	//     f.write(t)
	// end
	//
	// puts "\n\n"
	// puts "statistics written to outfile"
}
