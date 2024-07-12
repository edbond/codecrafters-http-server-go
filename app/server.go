package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path"
	"regexp"
	"strings"
)

type Request struct {
	URL     string
	Method  string
	Headers map[string]string
	Body    []byte
}

func main() {
	directory := flag.String("directory", "not-existing", "directory")
	flag.Parse()

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

		go handleConnection(conn, directory)

	}
}

func handleConnection(conn net.Conn, directory *string) {
	fmt.Println("accepted", conn, conn.RemoteAddr())

	req := make([]byte, 50*1024*1024)

	n, err := conn.Read(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error reading request: ", err.Error())
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

		writeResponse(conn, 200, "text/plain", []byte(abc))
		return
	}

	if strings.HasPrefix(httpReq.URL, "/files") {
		if httpReq.Method == "GET" {
			serveFile(conn, directory, httpReq)
		} else {
			writeFile(conn, directory, httpReq)
		}
		return
	}

	switch httpReq.URL {
	case "/":
		// conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
		writeResponse(conn, 200, "text/plain", []byte(""))
	case "/user-agent":
		userAgent := httpReq.Headers["User-Agent"]

		writeResponse(conn, 200, "text/plain", []byte(userAgent))

	default:
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	}

	conn.Close()
}

func writeFile(conn net.Conn, directory *string, httpReq Request) {
	filename := strings.TrimPrefix(httpReq.URL, "/files")

	var dirname string = "./not-existing"
	if directory != nil {
		dirname = *directory
	}

	w, err := os.Create(path.Join(dirname, filename))
	if err != nil {
		writeResponse(conn, 404, "text/plan", []byte("error opening file for writing"))
		return
	}

	defer w.Close()

	w.Write([]byte(httpReq.Body))

	writeResponse(conn, 201, "text/plan", []byte(""))
}

func serveFile(conn net.Conn, directory *string, httpReq Request) {
	filename := strings.TrimPrefix(httpReq.URL, "/files")

	var dirname string = "./not-existing"
	if directory != nil {
		dirname = *directory
	}

	r, err := os.Open(path.Join(dirname, filename))
	if err != nil {
		writeResponse(conn, 404, "text/plan", []byte("error opening file"))
		return
	}

	defer r.Close()

	body, err := io.ReadAll(r)
	if err != nil {
		writeResponse(conn, 500, "text/plan", []byte("error reading file"))
		return
	}

	writeResponse(conn, 200, "application/octet-stream", body)
}

var (
	statusText = map[int]string{
		200: "OK",
		201: "Created",
		404: "Not Found",
		500: "Server Error",
	}
)

func writeResponse(conn net.Conn, status int, contentType string, body []byte) {
	contentLength := len(body)

	response := fmt.Sprintf("HTTP/1.1 %d %s\r\nContent-Type: %s\r\nContent-Length: %d\r\n\r\n%s",
		status,
		statusText[status],
		contentType,
		contentLength,
		body)

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

	stage := "INITIAL"

	for _, line := range lines {
		if line == "\r\n" && stage == "HEADERS" {
			stage = "BODY"
		}

		switch stage {
		case "INITIAL":
			matches := methodUrlRe.FindStringSubmatch(line)
			fmt.Printf("matches: %#v\n", matches)
			if len(matches) >= 2 {
				// 0 = full match
				r.Method = matches[1]
				r.URL = matches[2]

				stage = "HEADERS"
				continue
			}
		case "HEADERS":
			headerMatches := headerRe.FindStringSubmatch(line)
			fmt.Printf("headers matches: %#v\n", headerMatches)
			if len(headerMatches) > 1 {
				r.Headers[headerMatches[1]] = headerMatches[2]
			}

		case "BODY":
			r.Body = append(r.Body, []byte(line)...)
		}
	}

	return r, nil
}
