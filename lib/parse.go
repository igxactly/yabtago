package lib

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
)

var u16le = binary.LittleEndian.Uint16
var u32le = binary.LittleEndian.Uint32
var u64le = binary.LittleEndian.Uint64

func printRecords(records chan *BlktraceRecord, finished chan int) {
	for true {
		r := <-records
		if r == nil {
			break
		} else {
			fmt.Println(r.String())
		}
	}
	finished <- 1
}

// Parse reads/parses/shows blktrace records.
func Parse(input *bufio.Reader, output *bufio.Writer, cfg *Config, outFmt string) {
	var err error
	var r *BlktraceRecord

	// TODO: Use config in parse
	recordsToPrint := make(chan *BlktraceRecord, 8192)
	printFinished := make(chan int)

	go printRecords(recordsToPrint, printFinished)

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

		recordsToPrint <- r
	}

	recordsToPrint <- nil
	<-printFinished
}

// ReadBlktraceRecord reads one blktrace record from reader and create/return a BlktraceRecord.
func ReadBlktraceRecord(reader io.Reader) (r *BlktraceRecord, err error) {
	var buf = make([]byte, 1024)

	readN := func(n int) []byte {
		_, err = io.ReadFull(reader, buf[0:n])
		return buf[0:n]
	}

	getField := func(field int) []byte {
		fo := &btFieldOffset
		return buf[fo[field]:fo[field+1]]
	}

	r = new(BlktraceRecord)

	// Read a blktrace record, excluding pdu_data
	readN(btBaseLength)

	// assign each fields
	// TODO: check endianess
	_ = u32le(getField(0)) // omit d_magic
	r.Seq = u32le(getField(1))
	r.Time = u64le(getField(2))
	r.Sector = u64le(getField(3))
	r.Bytes = u32le(getField(4))
	r.Action = u32le(getField(5))
	r.Pid = u32le(getField(6))
	r.Dev = u32le(getField(7))
	r.CPU = u32le(getField(8))
	r.Err = u16le(getField(9))
	r.PduLen = u16le(getField(10))

	if r.PduLen > 0 {
		// TODO: handle non-char bytes
		r.PduData = string(readN(int(r.PduLen)))
	} else {
		r.PduData = ""
	}

	return r, err
}
