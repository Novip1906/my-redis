package network

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"strings"
)

type Storage interface {
	Set(key, value string)
	Get(key string) (string, bool)
	Delete(key string)
}

type TCPServer struct {
	port    string
	storage Storage
	log     *slog.Logger
}

func NewTCPServer(port string, storage Storage, log *slog.Logger) *TCPServer {
	return &TCPServer{
		port:    port,
		storage: storage,
		log:     log,
	}
}

func (s *TCPServer) Start() error {
	listener, err := net.Listen("tcp", s.port)
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}
	defer listener.Close()

	s.log.Info("TCP Server started", "port", s.port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			s.log.Error("Connection error", "error", err)
			continue
		}
		go s.handleConnection(conn)
	}
}

func (s *TCPServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	remoteAddr := conn.RemoteAddr().String()
	s.log.Info("New connection", "client", remoteAddr)

	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {
		commandLine := scanner.Text()
		parts := strings.Fields(commandLine)

		if len(parts) == 0 {
			continue
		}

		cmd := strings.ToUpper(parts[0])
		var response string

		switch cmd {
		case "SET":
			if len(parts) < 3 {
				response = "(error) ERR wrong number of arguments for 'set'\n"
			} else {
				key := parts[1]
				val := strings.Join(parts[2:], " ")
				s.storage.Set(key, val)
				response = "OK\n"
			}
		case "GET":
			if len(parts) != 2 {
				response = "(error) ERR wrong number of arguments for 'get'\n"
			} else {
				key := parts[1]
				val, ok := s.storage.Get(key)
				if !ok {
					response = "(nil)\n"
				} else {
					response = val + "\n"
				}
			}
		case "DEL":
			if len(parts) != 2 {
				response = "(error) ERR wrong number of arguments for 'del'\n"
			} else {
				key := parts[1]
				s.storage.Delete(key)
				response = "OK\n"
			}
		case "QUIT":
			conn.Write([]byte("Bye!\n"))
			return
		default:
			response = fmt.Sprintf("(error) ERR unknown command '%s'\n", cmd)
		}

		conn.Write([]byte(response))
	}

	slog.Info("Connection closed", "client", remoteAddr)
}
