package main

import (
	"fmt"
	"regexp"
	"strings"

	// Uncomment this block to pass the first stage
	"net"
	"os"
)

type Request struct {
	URL    string
	Method string
}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	// Uncomment this block to pass the first stage
	//
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	conn, err := l.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}

	req := make([]byte, 256)
	n, err := conn.Read(req)
	if err != nil {
		fmt.Println("error reading request: ", err.Error())
		os.Exit(1)
	}
	fmt.Printf("bytes read: %d\n", n)

	httpReq, err := parseRequest(req[:n])
	if err != nil {
		fmt.Println("error parsing request: ", err.Error())
		os.Exit(2)
	}

	fmt.Printf("request: %+v\n", httpReq)

	if strings.HasPrefix(httpReq.URL, "/echo") {
		abc := strings.TrimPrefix(httpReq.URL, "/echo/")

		conn.Write([]byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 3\r\n\r\n%s", abc)))
		return
	}

	switch httpReq.URL {
	case "/":
		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	default:
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	}

}

var (
	methodUrlRe = regexp.MustCompile(`^(GET|POST|PUT) (.+?) HTTP/(\d\.?\d)$`)
)

func parseRequest(req []byte) (Request, error) {
	r := Request{}

	lines := strings.Split(string(req), "\r\n")

	fmt.Printf("lines: %#v\n", lines)

	for _, l := range lines {
		matches := methodUrlRe.FindStringSubmatch(l)

		fmt.Printf("matches: %#v\n", matches)
		if len(matches) >= 2 {
			// 0 = full match
			r.Method = matches[1]
			r.URL = matches[2]
		}
	}

	return r, nil
}
