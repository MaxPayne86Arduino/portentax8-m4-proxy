package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"reflect"
	"time"

	"github.com/facchinm/msgpack-go"
)

const (
	REQUEST      = 0
	RESPONSE     = 1
	NOTIFICATION = 2
)

func handleConnection(c net.Conn, chardev *os.File, resp chan []byte) {

	fmt.Printf("Serving %s\n", c.RemoteAddr().String())

	data, _, err := msgpack.Unpack(c)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(data)
	var buf bytes.Buffer

	msgpack.Pack(&buf, data.Interface())

	/*
		msgId := _req[1]
		msgName := _req[2]
		msgArgs := _req[3]

		rawdata := make([]byte, 5)
		rawdata[0] = byte(msgType.Int())
		rawdata[1] = byte(msgId.Int())
		rawdata[2] = byte(msgId.Int())
		rawdata[3] = byte(msgId.Int())
		rawdata[4] = byte(msgId.Int())
		rawdata = append(rawdata, msgName.Bytes()...)

		something := msgArgs.Addr().Bytes()

		fmt.Println(something)
		rawdata = append(rawdata, something...)

		fmt.Println(data)
		fmt.Println(rawdata)
	*/

	fmt.Println(buf)

	chardev.Write(buf.Bytes())

	msgType := buf.Bytes()[1]

	if msgType == REQUEST {
		// wait to be unlocked by the other reading goroutine
		// TODO: add timeout handling
		fmt.Println("wait for response")
		select {
		case response := <-resp:
			//chardev.Read(response)
			fmt.Println("return response to client")
			c.Write(response)
		}
	}
	fmt.Println("done")

	if msgType == NOTIFICATION {
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

			conn.Close()
		}
	}
}

func main() {

	chardev, err := os.OpenFile("/dev/x8h7_ui", os.O_RDWR, 0)

	chardev_reader_chan := make(chan []byte, 1)

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
