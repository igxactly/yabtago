package lib

import "fmt"

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
