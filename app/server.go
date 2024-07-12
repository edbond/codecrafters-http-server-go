package main

import (
	"fmt"
	"os/signal"
	"regexp"
	"strings"

	// Uncomment this block to pass the first stage
	"net"
	"os"
)

type Request struct {
	URL     string
	Method  string
	Headers map[string]string
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

	intChan := make(chan os.Signal, 1)
	signal.Notify(intChan, os.Interrupt)

	go func() {
		<-intChan
		os.Exit(1)
	}()

	for {
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

			writeResponse(conn, 200, "text/plain", abc)
			return
		}

		switch httpReq.URL {
		case "/":
			conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
		case "/user-agent":
			userAgent := httpReq.Headers["User-Agent"]

			writeResponse(conn, 200, "text/plain", userAgent)
		default:
			conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		}

		conn.Close()
	}

}

func writeResponse(conn net.Conn, status int, contentType, body string) {
	var statusDescription string
	switch status {
	case 200:
		statusDescription = "OK"
	case 404:
		statusDescription = "Not Found"
	}

	contentLength := len(body)

	response := fmt.Sprintf("HTTP/1.1 %d %s\r\nContent-Type: %s\r\nContent-Length: %d\r\n\r\n%s", status, statusDescription, contentType, contentLength, body)

	conn.Write([]byte(response))
}

var (
	methodUrlRe = regexp.MustCompile(`^(GET|POST|PUT) (.+?) HTTP/(\d\.?\d)$`)

	headerRe = regexp.MustCompile(`^([A-Za-z-]+): (.+)$`)
)

func parseRequest(req []byte) (Request, error) {
	r := Request{}
	r.Headers = make(map[string]string)

	lines := strings.Split(string(req), "\r\n")

	fmt.Printf("lines: %#v\n", lines)

	for _, line := range lines {
		matches := methodUrlRe.FindStringSubmatch(line)

		fmt.Printf("matches: %#v\n", matches)
		if len(matches) >= 2 {
			// 0 = full match
			r.Method = matches[1]
			r.URL = matches[2]
			continue
		}

		headerMatches := headerRe.FindStringSubmatch(line)
		fmt.Printf("headers matches: %#v\n", headerMatches)
		if len(headerMatches) > 1 {
			r.Headers[headerMatches[1]] = headerMatches[2]
		}
	}

	return r, nil
}
