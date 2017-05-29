package lib

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
)

// Parse reads/parses/shows blktrace records.
func Parse(input *bufio.Reader, output *bufio.Writer) {
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
