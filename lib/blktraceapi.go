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

// Trace actions - definition from blktrace_api.h
const (
	TAQueue       uint32 = iota + 1 /* queued */
	TABackmerge                     /* back merged to existing rq */
	TAFrontmerge                    /* front merge to existing rq */
	TAGetrq                         /* allocated new request */
	TASleeprq                       /* sleeping on rq allocation */
	TARequeue                       /* request requeued */
	TAIssue                         /* sent to driver */
	TAComplete                      /* completed by driver */
	TAPlug                          /* queue was plugged */
	TAUnplugIO                      /* queue was unplugged by io */
	TAUnplugTimer                   /* queue was unplugged by timer */
	TAInsert                        /* insert request */
	TASplit                         /* bio was split */
	TABounce                        /* bio was bounced */
	TARemap                         /* bio was remapped */
	TAAbort                         /* request aborted */
	TADrvData                       /* binary driver data */
)

// Trace notify events
const (
	TNProcess   uint32 = iota /* establish pid/name mapping */
	TNTimestamp               /* include system clock */
	TNMessage                 /* Character string message */
)

// Trace category bit is most significiant half of action field
const (
	TCShift = 16
)

// Trace categories - definition from blktrace_api.h
const (
	TCRead     uint32 = 1 << (iota + TCShift) /* reads */
	TCWrite                                   /* writes */
	TCFlush                                   /* flush */
	TCSync                                    /* sync */
	TCQueue                                   /* queueing/merging */
	TCRequeue                                 /* requeueing */
	TCIssue                                   /* issue */
	TCComplete                                /* completions */
	TCFs                                      /* fs requests */
	TCPc                                      /* pc requests */
	TCNotify                                  /* special message */
	TCAhead                                   /* readahead */
	TCMeta                                    /* metadata */
	TCDiscard                                 /* discard requests */
	TCDrvData                                 /* binary driver data */
	TCFua                                     /* fua requests */

	TCEnd = TCFua /* we've run out of bits! */
)

// Trace actions in FULL. Additionally, read or write is masked
const (
	FAQueue       uint32 = TAQueue | TCQueue
	FABackmerge          = TABackmerge | TCQueue
	FAFrontmerge         = TAFrontmerge | TCQueue
	FAGetrq              = TAGetrq | TCQueue
	FASleeprq            = TASleeprq | TCQueue
	FARequeue            = TARequeue | TCRequeue
	FAIssue              = TAIssue | TCIssue
	FAComplete           = TAComplete | TCComplete
	FAPlug               = TAPlug | TCQueue
	FAUnplugIO           = TAUnplugIO | TCQueue
	FAUnplugTimer        = TAUnplugTimer | TCQueue
	FAInsert             = TAInsert | TCQueue
	FASplit              = TASplit
	FABounce             = TABounce
	FARemap              = TARemap | TCQueue
	FAAbort              = TAAbort | TCQueue
	FADrvData            = TADrvData | TCDrvData

	// Notify
	FNProcess   uint32 = TNProcess | TCNotify
	FNTimestamp        = TNTimestamp | TCNotify
	FNMessage          = TNMessage | TCNotify
)

// blktrace magic / version sequence
const (
	BlktraceMagic   uint32 = 0x65617400
	BlktraceVersion        = 0x07
)
