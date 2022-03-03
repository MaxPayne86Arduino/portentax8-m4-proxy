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
	//fmt.Println("resolving ", name)
	return self[name], nil
}

func (self Resolver) Functions() []string {
	var functions []string
	for el := range self {
		functions = append(functions, el)
	}
	fmt.Println(functions)
	return functions
}

func echo(test string) (string, fmt.Stringer) {
	fmt.Println(test)
	return "Hello, " + test, nil
}

func whoami() (string, fmt.Stringer) {
	out, _ := exec.Command("whoami").Output()
	return string(out), nil
}

func add(a, b uint) (uint, fmt.Stringer) {
	fmt.Println("calling add on ", a, " and ", b)
	return a + b, nil
}

func serialportListener(serport *os.File) {
	for {
		data := make([]byte, 1024)
		n, err := serport.Read(data)

		fmt.Println("got data on serial port")

		if err != nil {
			fmt.Println(err)
			continue
		}

		data = data[:n]
		fmt.Println(data)

		conn, err := net.Dial("tcp", ":5001")
		client := rpc.NewSession(conn, true)
		_, xerr := client.Send("tty", data)
		if xerr != nil {
			fmt.Println(xerr)
			continue
		}
	}
}

var serport *os.File

func tty(test []reflect.Value) (int, fmt.Stringer) {
	//fmt.Println("tty called: ", test)
	var temp []byte
	for _, elem := range test {
		temp = append(temp, byte(elem.Int()))
	}
	//fmt.Println(temp)
	serport.Write(temp)
	return len(test), nil
}

func main() {

	serport, _ = os.OpenFile("/dev/ttyGS0", os.O_RDWR, 0)

	conn, _ := net.Dial("tcp", ":5001")
	client := rpc.NewSession(conn, true)
	retval, xerr := client.Send("subtract", 35, 36)
	if xerr != nil {
		fmt.Println(xerr)
	}
	fmt.Println(retval.Int())

	go serialportListener(serport)

	res := Resolver{"echo": reflect.ValueOf(echo), "add": reflect.ValueOf(add), "tty": reflect.ValueOf(tty), "whoami": reflect.ValueOf(whoami)}

	serv := rpc.NewServer(res, true, nil, 5002)
	l, _ := net.Listen("tcp", "127.0.0.1:5002")
	serv.Listen(l)
	serv.Register()
	serv.Run()
}
