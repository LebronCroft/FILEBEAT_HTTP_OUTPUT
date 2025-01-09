//go:generate protoc --gogofaster_out=:. bridge.proto
package plugins

import (
	"bufio"
	"encoding/binary"
	"fmt"
	infraLog "github.com/fufuok/beats-http-output/infra"
	io "io"
	"sync"
)

type Client struct {
	rx     io.ReadCloser
	tx     io.WriteCloser
	reader *bufio.Reader
	writer *bufio.Writer
	rmu    *sync.Mutex
	wmu    *sync.Mutex
}

func (c *Client) SendRecord(rec *Record) (err error) {
	infraLog.GlobalLog.Debug("Attempting to acquire write lock...")
	c.wmu.Lock()
	infraLog.GlobalLog.Debug("Write lock acquired.")
	defer c.wmu.Unlock()

	size := rec.Size()
	infraLog.GlobalLog.Debug(fmt.Sprintf("Writing size: %d to writer", size))
	err = binary.Write(c.writer, binary.LittleEndian, uint32(size))
	if err != nil {
		infraLog.GlobalLog.Error(fmt.Sprintf("Error writing size: %v", err))
		return err
	}

	var buf []byte
	infraLog.GlobalLog.Debug("Marshalling record...")
	buf, err = rec.Marshal()
	if err != nil {
		infraLog.GlobalLog.Error(fmt.Sprintf("Error marshalling record: %v", err))
		return err
	}
	infraLog.GlobalLog.Debug("Record marshalled successfully.")

	infraLog.GlobalLog.Debug("Writing data to writer...")
	_, err = c.writer.Write(buf)
	if err != nil {
		infraLog.GlobalLog.Error(fmt.Sprintf("Error writing data: %v", err))
		return err
	}
	infraLog.GlobalLog.Debug("Data written to writer successfully.")

	return
}
func (c *Client) ReceiveTask() (t *Task, err error) {
	c.rmu.Lock()
	defer c.rmu.Unlock()
	var len uint32
	err = binary.Read(c.reader, binary.LittleEndian, &len)
	if err != nil {
		return
	}
	var buf []byte
	buf, err = c.reader.Peek(int(len))
	if err != nil {
		return
	}
	_, err = c.reader.Discard(int(len))
	if err != nil {
		return
	}
	t = &Task{}
	err = t.Unmarshal(buf)
	return
}
func (c *Client) Flush() (err error) {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	if c.writer.Buffered() != 0 {
		err = c.writer.Flush()
	}
	return
}

func (c *Client) Close() {
	c.writer.Flush()
	c.rx.Close()

}
