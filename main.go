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
	mode       string
	proxyAddr  string
	serverAddr string
	fileName   string
)

func init() {
	flag.StringVar(&mode, "mode", "client", "mode: client, server")
	flag.StringVar(&proxyAddr, "proxy_addr", "", "proxy addr")
	flag.StringVar(&serverAddr, "server_addr", "0.0.0.0:39002", "server addr")
	flag.StringVar(&fileName, "file", "", "file to transfer")
}

func main() {
	flag.Parse()
	switch mode {
	case "client":
		serveClient()
	case "server":
		serveServer()
	default:
		fmt.Printf("invalid mode: %s\n", mode)
	}
}

func serveClient() {
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Printf("open file '%s' error, %s\n", fileName, err)
		os.Exit(1)
	}
	defer file.Close()

	dest := serverAddr
	if proxyAddr != "" {
		dest = proxyAddr
	}

	conn, err := net.Dial("tcp", dest)
	if err != nil {
		fmt.Printf("dial to '%s' error, %s\n", dest, err)
		os.Exit(1)
	}
	defer conn.Close()

	baseName := filepath.Base(fileName)
	bw := bufio.NewWriter(conn)
	bw.WriteString(fmt.Sprintf("name:%s\n", baseName))
	proxyTo := serverAddr
	if proxyAddr == "" {
		proxyTo = "no"
	}
	bw.WriteString(fmt.Sprintf("proxyTo:%s\n", proxyTo))
	bw.Flush()
	log.Printf("transfer name:%s", baseName)
	io.Copy(bw, file)
}

func serveServer() {
	log.Printf("serve ftransfer-server at %s", serverAddr)
	defer log.Printf("exit ftransfer-server")
	ln, err := net.Listen("tcp", serverAddr)
	if err != nil {
		log.Printf("listen '%s' error, %s", serverAddr, err)
		os.Exit(1)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept error, %s", err)
			continue
		}
		handleServerConn(conn)
	}
}

func handleServerConn(conn net.Conn) {
	defer conn.Close()
	br := bufio.NewReader(conn)

	nameLine, _, err := br.ReadLine()
	if err != nil {
		log.Printf("read name line error, %s", err)
		return
	}
	proxyToLine, _, err := br.ReadLine()
	if err != nil {
		log.Printf("read proxyTo line error, %s", err)
		return
	}

	var name, proxyTo string
	fmt.Sscanf(string(nameLine), "name:%s", &name)
	fmt.Sscanf(string(proxyToLine), "proxyTo:%s", &proxyTo)
	log.Printf("accept file: %s, proxy to: %s", name, proxyTo)

	if proxyTo == "no" {
		path := "upload/" + name
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("open file '%s' to write error, %s", path, err)
			return
		}
		defer f.Close()
		io.Copy(f, br)
	} else {
		pconn, err := net.Dial("tcp", proxyTo)
		if err != nil {
			log.Printf("dial to '%s' err, %s", proxyTo, err)
			return
		}
		defer pconn.Close()

		bw := bufio.NewWriter(pconn)
		bw.WriteString(fmt.Sprintf("name:%s\n", name))
		bw.WriteString(fmt.Sprintf("proxyTo:no\n"))
		bw.Flush()
		log.Printf("transfer name:%s to %s", name, proxyTo)
		io.Copy(bw, conn)
	}
}
