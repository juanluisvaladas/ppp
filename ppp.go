package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
)

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "%s <listen address> <destination address>\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Example:\n")
	fmt.Fprintf(os.Stderr, "%s 0.0.0.0:8000 myproxy.local:443\n", os.Args[0])
}

func generateProxyHeader(src net.Conn) string {
	srcLocAdrr := src.LocalAddr().(*net.TCPAddr)
	srcRemAdrr := src.RemoteAddr().(*net.TCPAddr)

	// PROXY TCP 4 <src remote address> <src local address> <src remote port> <src local port>\r\n
	return "PROXY TCP4 " +
		srcRemAdrr.IP.String() + " " +
		srcLocAdrr.IP.String() + " " +
		strconv.Itoa(srcRemAdrr.Port) + " " +
		strconv.Itoa(srcLocAdrr.Port) + "\r\n"
}

func proxy(src net.Conn, dest net.Conn, waitChan chan int) {
	for {
		data := make([]byte, 1024)
		rx, err := src.Read(data)
		if err != nil {
			fmt.Printf(err.Error())
			waitChan <- 1
			return
		}
		fmt.Println(rx)
		fmt.Printf("Received: %d\n", rx)

		if rx > 0 {
			tx, err := dest.Write(data[:rx])
			if err != nil {
				fmt.Printf(err.Error())
				waitChan <- 1
				return
			}
			if rx != tx {
				fmt.Printf("Received: %d Sent: %d\n", rx, tx)
				waitChan <- 1
				return
			}
		}

	}
}

func proxyConn(src net.Conn) {
	defer src.Close()

	fmt.Printf("Handling connection %s\n", src)
	fmt.Printf("Local connection from %s to %s\n", src.LocalAddr().String(), src.RemoteAddr().String())

	// Start destination connection
	dest, err := net.Dial("tcp", os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}
	defer dest.Close()

	fmt.Println(generateProxyHeader(src))
	_, err = dest.Write([]byte(generateProxyHeader(src)))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to write PROXY PROTOCOL HEADER")
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}

	waitChan := make(chan int)
	go proxy(src, dest, waitChan)
	go proxy(dest, src, waitChan)
	_ = <-waitChan

	fmt.Println("Closed conn")
}
func main() {
	if len(os.Args) != 3 {
		printUsage()
		os.Exit(1)
	}

	ln, err := net.Listen("tcp", os.Args[1])

	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	fmt.Printf("Listening on %s\n", os.Args[1])
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			ln.Close()
			return
		}
		go proxyConn(conn)

	}
}
