package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
)

var (
	mode      string
	proxyAddr string
	destAddr  string
	fileName  string
	useProxy  bool
)

func init() {
	flag.StringVar(&mode, "mode", "client", "mode: client, proxy or dest")
	flag.StringVar(&proxyAddr, "proxy_addr", "0.0.0.0:39001", "proxy addr")
	flag.StringVar(&destAddr, "dest_addr", "0.0.0.0:39002", "dest addr")
	flag.StringVar(&fileName, "file", "", "file to transfer")
	flag.BoolVar(&useProxy, "use_proxy", true, "transfer to proxy first")
}

func main() {
	flag.Parse()
	switch mode {
	case "client":
		serveClient()
	case "proxy":
		serveProxy()
	case "dest":
		serveDest()
	default:
		fmt.Printf("invalid mode: %s\n", mode)
	}
}

func serveClient() {
	file, err := os.Open(fileName)
	if err != nil {
		panic(file)
	}
	defer file.Close()

	conn, err := net.Dial("tcp", destAddr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	baseName := filepath.Base(fileName)
	bw := bufio.NewWriter(conn)
	bw.WriteString(fmt.Sprintf("name:%s\n", baseName))
	bw.Flush()
	log.Printf("transfer name:%s", baseName)
	io.Copy(bw, file)
}

func serveProxy() {

}

func serveDest() {
	log.Printf("serve ftransfer-dest at %s", destAddr)
	defer log.Printf("exit ftransfer-dest")
	ln, err := net.Listen("tcp", destAddr)
	if err != nil {
		panic(err)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}
		handleDestConn(conn)
	}
}

func handleDestConn(conn net.Conn) {
	br := bufio.NewReader(conn)
	nameLine, _, err := br.ReadLine()
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	var name string
	fmt.Sscanf(string(nameLine), "name:%s", &name)
	log.Printf("accept file: %s", name)

	path := "upload/" + name
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	io.Copy(f, br)
}
