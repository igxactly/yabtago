package lib

import (
	"fmt"
)

// Reference: blktrace record structure
//     - definition from blktrace_api.h
//
// blktrace log file is written in this structure.
// Byte order(endian) depends on system architecture.
//
// __u32 magic;     /* MAGIC << 8 | version */
// __u32 sequence;  /* event number */
// __u64 time;      /* in nanoseconds */
// __u64 sector;    /* disk offset */
// __u32 bytes;     /* transfer length */
// __u32 action;    /* what happened */
// __u32 pid;       /* who did it */
// __u32 device;    /* device number */
// __u32 cpu;       /* on what cpu did it happen */
// __u16 error;     /* completion error */
// __u16 pdu_len;   /* length of data after this trace */
//
// and PDU data follows.

// BlktraceRecord represents a blktrace record.
type BlktraceRecord struct {
	// Magic                        uint32

	Seq                          uint32
	Time, Sector                 uint64
	Bytes, Action, Pid, Dev, CPU uint32
	Err, PduLen                  uint16

	PduData string
}

const btBaseLength int = 8*2 + 4*7 + 2*2 // 64bit*2, 32bit*7, 16bit*2
var btFieldSize = [12]int{4, 4, 8, 8, 4, 4, 4, 4, 4, 2, 2, 0}
var btFieldOffset = [12]int{} // value is set in init()

func init() {
	btFieldOffset[0] = 0
	for i := 1; i < len(btFieldSize); i++ {
		btFieldOffset[i] = btFieldOffset[i-1] + btFieldSize[i-1]
	}
}

// String -
func (r *BlktraceRecord) String() string {
	return fmt.Sprintf("BlktraceRecord:%d %d %d %d 0x%08x %d %d %d %d %d %s",
		r.Seq, r.Time, r.CPU, r.Pid, r.Action,
		r.Dev, r.Sector, r.Bytes, r.Err, r.PduLen, r.PduData)

	/* TODO: replace non-readable chars from r.Pdu_data /[\x00-\x08\x0A-\x1F\x7F]/ */
}
