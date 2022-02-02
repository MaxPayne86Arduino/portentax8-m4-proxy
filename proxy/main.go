package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"reflect"
	"time"

	"github.com/msgpack/msgpack-go"
)

const (
	REQUEST      = 0
	RESPONSE     = 1
	NOTIFICATION = 2
)

func handleConnection(c net.Conn, chardev *os.File, resp chan []byte) {

	fmt.Printf("Serving %s\n", c.RemoteAddr().String())

	data, _, err := msgpack.UnpackReflected(c)
	if err != nil {
		fmt.Println(err)
		return
	}

	_req, ok := data.Interface().([]reflect.Value)
	if !ok {
		return
	}
	msgType := _req[0]

	chardev.Write(data.Bytes())

	if msgType.Int() == REQUEST {
		// wait to be unlocked by the other reading goroutine
		select {
		case response := <-resp:
			chardev.Read(response)
			c.Write(response)
		}
	}

	if msgType.Int() == NOTIFICATION {
		// fire and forget
	}

	c.Close()
}

func chardevListener(chardev *os.File, resp chan []byte) {

	for {

		data := make([]byte, 1024)
		response := make([]byte, 1024)

		n, err := chardev.Read(data)

		data = data[:n]

		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Println("chardev.Read returned")

		if n <= 0 {
			continue
		}

		fmt.Println("got data from chardev")
		fmt.Println(data)

		start := 0
		for {

			copy_data := data[start:]

			message, n, err := msgpack.UnpackReflected(bytes.NewReader(copy_data))

			// TODO: ATTENTION: bytes_consumed returned by this function are WRONG on mainline
			// The patch is the following one

			/*

				diff --git a/unpack.go b/unpack.go
				index 9732d0a..7771380 100644
				--- a/unpack.go
				+++ b/unpack.go
				@@ -35,7 +35,7 @@ func readUint16(reader io.Reader) (v uint16, n int, err error) {
				        var data Bytes2
				        n, e := reader.Read(data[0:])
				        if e != nil {
				-               return 0, n, e
				+               return 0, 2, e
				        }
				        return (uint16(data[0]) << 8) | uint16(data[1]), n, nil
				 }
				@@ -44,7 +44,7 @@ func readUint32(reader io.Reader) (v uint32, n int, err error) {
				        var data Bytes4
				        n, e := reader.Read(data[0:])
				        if e != nil {
				-               return 0, n, e
				+               return 0, 4, e
				        }
				        return (uint32(data[0]) << 24) | (uint32(data[1]) << 16) | (uint32(data[2]) << 8) | uint32(data[3]), n, nil
				 }
				@@ -53,7 +53,7 @@ func readUint64(reader io.Reader) (v uint64, n int, err error) {
				        var data Bytes8
				        n, e := reader.Read(data[0:])
				        if e != nil {
				-               return 0, n, e
				+               return 0, 8, e
				        }
				        return (uint64(data[0]) << 56) | (uint64(data[1]) << 48) | (uint64(data[2]) << 40) | (uint64(data[3]) << 32) | (uint64(data[4]) << 24) | (uint64(data[5]) << 16) | (uint64(data[6]) << 8) | uint64(data[7]), n, nil
				 }
				@@ -62,7 +62,7 @@ func readInt16(reader io.Reader) (v int16, n int, err error) {
				        var data Bytes2
				        n, e := reader.Read(data[0:])
				        if e != nil {
				-               return 0, n, e
				+               return 0, 2, e
				        }
				        return (int16(data[0]) << 8) | int16(data[1]), n, nil
				 }
				@@ -71,7 +71,7 @@ func readInt32(reader io.Reader) (v int32, n int, err error) {
				        var data Bytes4
				        n, e := reader.Read(data[0:])
				        if e != nil {
				-               return 0, n, e
				+               return 0, 4, e
				        }
				        return (int32(data[0]) << 24) | (int32(data[1]) << 16) | (int32(data[2]) << 8) | int32(data[3]), n, nil
				 }
				@@ -80,7 +80,7 @@ func readInt64(reader io.Reader) (v int64, n int, err error) {
				        var data Bytes8
				        n, e := reader.Read(data[0:])
				        if e != nil {
				-               return 0, n, e
				+               return 0, 8, e
				        }
				        return (int64(data[0]) << 56) | (int64(data[1]) << 48) | (int64(data[2]) << 40) | (int64(data[3]) << 32) | (int64(data[4]) << 24) | (int64(data[5]) << 16) | (int64(data[6]) << 8) | int64(data[7]), n, nil
				 }
				@@ -197,7 +197,6 @@ func unpack(reader io.Reader, reflected bool) (v reflect.Value, n int, err error
				                if e != nil {
				                        return reflect.Value{}, nbytesread, e
				                }
				-               nbytesread += n
				        } else if c >= FIXARRAY && c <= FIXARRAYMAX {
				                if reflected {
				                        retval, n, e = unpackArrayReflected(reader, lownibble(c))
				@@ -208,7 +207,6 @@ func unpack(reader io.Reader, reflected bool) (v reflect.Value, n int, err error
				                if e != nil {
				                        return reflect.Value{}, nbytesread, e
				                }
				-               nbytesread += n
				        } else if c >= FIXRAW && c <= FIXRAWMAX {
				                data := make([]byte, lowfive(c))
				                n, e := reader.Read(data)

			*/

			fmt.Printf("%d bytes consumed\n", n)
			fmt.Printf("%v\n", message)
			fmt.Println(message)
			fmt.Println(err)

			if err == io.EOF {
				break
			}

			_req, ok := message.Interface().([]reflect.Value)
			if !ok {
				break
			}

			msgType := _req[0]

			if msgType.Int() == RESPONSE {
				// unlock thread waiting on handleConnection
				resp <- copy_data[:n]
				break
			}

			// REQUEST or NOTIFICATION
			conn, err := net.Dial("tcp", ":5002")
			if err != nil {
				fmt.Println(err)
				break
			}
			conn.Write(copy_data[:n])

			start += n

			if msgType.Int() == REQUEST {
				fmt.Println("ask for a response")

				var to_send []byte
				i := 0
				for {
					n, err := conn.Read(response)
					conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
					to_send = append(to_send, response[:n]...)
					i += n
					if err != nil {
						break
					}
				}
				fmt.Println("sending ", to_send[:i])
				chardev.Write(to_send[:i])
			}

			if msgType.Int() == NOTIFICATION {
				// fire and forget
			}

			//conn.Close()
		}
	}
}

func main() {

	chardev, err := os.Open("/dev/x8h7_ui")

	chardev_reader_chan := make(chan []byte, 100)

	l, err := net.Listen("tcp4", ":5001")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer l.Close()

	go chardevListener(chardev, chardev_reader_chan)

	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}
		go handleConnection(c, chardev, chardev_reader_chan)
	}
}
