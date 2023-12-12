package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"reflect"

	"github.com/msgpack-rpc/msgpack-rpc-go/rpc"
)

type Resolver map[string]reflect.Value

func (self Resolver) Resolve(name string, arguments []reflect.Value) (reflect.Value, error) {
	return self[name], nil
}

func (self Resolver) Functions() []string {
	var functions []string
	for el := range self {
		functions = append(functions, el)
	}
	return functions
}

func echo(test string) (string, fmt.Stringer) {
	return "Hello, " + test, nil
}

func whoami() (string, fmt.Stringer) {
	out, _ := exec.Command("whoami").Output()
	return string(out), nil
}

func add(a, b uint) (uint, fmt.Stringer) {
	return a + b, nil
}

func serialportListener(serport *os.File) {
	for {
		data := make([]byte, 1024)
		n, err := serport.Read(data)

		if err != nil {
			continue
		}

		data = data[:n]

		conn, err := net.Dial("tcp", "m4-proxy:5001")
		client := rpc.NewSession(conn, true)
		xerr := client.Send("tty", data)
		if xerr != nil {
			continue
		}
	}
}

var serport *os.File

func tty(test []reflect.Value) fmt.Stringer {
	var temp []byte
	for _, elem := range test {
		temp = append(temp, byte(elem.Int()))
	}
	serport.Write(temp)
	return nil
}

func main() {

	serport, _ = os.OpenFile("/dev/ttyGS0", os.O_RDWR, 0)

	go serialportListener(serport)

	res := Resolver{"echo": reflect.ValueOf(echo), "add": reflect.ValueOf(add), "tty": reflect.ValueOf(tty), "whoami": reflect.ValueOf(whoami)}

	serv := rpc.NewServer(res, true, nil, 5002)
	l, _ := net.Listen("tcp", ":5002")
	serv.Listen(l)
	serv.Register()
	serv.Run()
}
