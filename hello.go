package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
)

func main() {
	go startServer(8080)

	status, body, err := execute("http://localhost:8080")
	if err != nil {
		handleErr(err)
	}

	fmt.Printf("Status: %d\n", status)
	fmt.Printf("Body: %s\n", body)
}

func execute(url string) (int, string, error) {
	if !strings.HasPrefix(url, "http://") {
		return -1, "", errors.New("url must start with http://")
	}

	noProtocol := strings.Replace(url, "http://", "", 1)
	splitNoProtocol := strings.SplitN(noProtocol, "/", 2)
	domain := splitNoProtocol[0]

	path := "/"
	if len(splitNoProtocol) == 2 {
		path = splitNoProtocol[1]
	}

	splitDomain := strings.SplitN(domain, ":", 2)
	port := "80"
	domainNoPort := splitDomain[0]
	if len(splitDomain) == 2 {
		port = splitDomain[1]
	}

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", domainNoPort, port))
	if err != nil {
		handleErr(err)
	}
	fmt.Fprintf(conn, fmt.Sprintf("GET %s HTTP/1.0\r\n", path))
	fmt.Fprintf(conn, fmt.Sprintf("Host: %s\r\n", domainNoPort))
	fmt.Fprintf(conn, "\r\n\r\n")
	reader := bufio.NewReader(conn)
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		handleErr(err)
	}

	splitStatusLine := strings.Split(statusLine, " ")

	if len(splitStatusLine) != 3 {
		return -1, "", errors.New("invalid response status line")
	}
	status, err := strconv.Atoi(splitStatusLine[1])
	if err != nil {
		handleErr(err)
	}

	headers, err := readHeaders(reader)
	if err != nil {
		handleErr(err)
	}

	responseBody, err := readBody(reader, headers)
	if err != nil {
		handleErr(err)
	}

	return status, responseBody, nil
}

func readHeaders(reader *bufio.Reader) (map[string]string, error) {
	headers := make(map[string]string)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			handleErr(err)
		}

		if strings.TrimSpace(line) == "" {
			break
		}

		splitLine := strings.Split(line, ": ")
		if len(splitLine) != 2 {
			return nil, errors.New("Invalid header: " + line)
		}

		headers[strings.ToLower(splitLine[0])] = strings.TrimSpace(splitLine[1])
	}

	return headers, nil
}

func startServer(port int) {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))

	if err != nil {
		handleErr(err)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			handleErr(err)
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	requestLine, err := reader.ReadString('\n')
	if err != nil {
		handleErr(err)
	}

	splitRequestLine := strings.Split(requestLine, " ")
	if len(splitRequestLine) != 3 {
		writeResponse(conn, 500, "Invalid request!")
		return
	}

	method := splitRequestLine[0]
	path := splitRequestLine[1]
	httpVersion := strings.TrimSpace(splitRequestLine[2])

	headers, err := readHeaders(reader)
	if err != nil {
		handleErr(err)
	}

	requestBody, err := readBody(reader, headers)
	if err != nil {
		handleErr(err)
	}

	responseBody := fmt.Sprintf(
		"Method: %s, Path: %s, Http version: %s, Body: %s", method, path, httpVersion, requestBody)

	writeResponse(conn, 200, responseBody)
}

func readBody(reader *bufio.Reader, headers map[string]string) (string, error) {
	contentLengthString := headers["content-length"]
	if contentLengthString != "" {
		contentLength, err := strconv.Atoi(contentLengthString)
		if err != nil {
			handleErr(err)
		}

		buf := make([]byte, contentLength)
		readCount, err := io.ReadFull(reader, buf)
		if err != nil {
			handleErr(err)
		}
		if readCount != contentLength {
			return "", errors.New(fmt.Sprintf("Read invalid length %d, expected %d", readCount, contentLength))
		}

		return string(buf), nil
	}
	return "", nil
}

func writeResponse(conn net.Conn, status int, body string) {
	writer := bufio.NewWriter(conn)
	_, err := writer.WriteString(fmt.Sprintf("HTTP/1.0 %d Woot\r\n", status))
	if err != nil {
		handleErr(err)
	}

	if body != "" {
		writer.WriteString(fmt.Sprintf("Content-Length: %d\r\n", len(body)))
	}
	writer.WriteString("\r\n")
	writer.WriteString(body)

	writer.Flush()
}

func handleErr(err error) {
	panic(err)
}
