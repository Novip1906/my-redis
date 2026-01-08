package network

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Storage interface {
	Set(key, value string)
	Get(key string) (string, bool)
	Delete(key string)
	SetTTL(key string, seconds int64) bool
	GetTTL(key string) int64
	Increment(key string) (int64, error)
}

type TCPServer struct {
	wg       sync.WaitGroup
	port     string
	storage  Storage
	log      *slog.Logger
	listener net.Listener
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
	s.listener = listener

	s.log.Info("TCP Server started", "port", s.port)

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				s.log.Info("Server stopped accepting new connections")
				return nil
			}
			s.log.Error("Connection error", "error", err)
			continue
		}

		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

func (s *TCPServer) Stop() {
	s.listener.Close()
	s.wg.Wait()
}

func (s *TCPServer) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	remoteAddr := conn.RemoteAddr().String()

	log := s.log.With("client", remoteAddr)
	log.Info("New connection")

	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {
		conn.SetReadDeadline(time.Now().Add(5 * time.Minute))

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
				break
			}
			key := parts[1]
			val := strings.Join(parts[2:], " ")
			s.storage.Set(key, val)
			response = "OK\n"

		case "GET":
			if len(parts) != 2 {
				response = "(error) ERR wrong number of arguments for 'get'\n"
				break
			}
			key := parts[1]
			val, ok := s.storage.Get(key)
			if !ok {
				response = "(nil)\n"
			} else {
				response = val + "\n"
			}

		case "DEL":
			if len(parts) != 2 {
				response = "(error) ERR wrong number of arguments for 'del'\n"
				break
			}
			key := parts[1]
			s.storage.Delete(key)
			response = "OK\n"

		case "EXPIRE":
			if len(parts) != 3 {
				response = "(error) ERR wrong number of arguments for 'expire'\n"
				break
			}
			key := parts[1]
			seconds, err := strconv.Atoi(parts[2])
			if err != nil {
				response = "(error) ERR value is not an integer or out of range\n"
			}

			ok := s.storage.SetTTL(key, int64(seconds))
			if ok {
				response = "1\n"
			} else {
				response = "0\n"
			}

		case "TTL":
			if len(parts) != 2 {
				response = "(error) ERR wrong number of arguments for 'ttl'\n"
				break
			}
			key := parts[1]
			seconds := s.storage.GetTTL(key)
			response = fmt.Sprintf("%d\n", seconds)

		case "INCR":
			if len(parts) != 2 {
				response = "(error) ERR wrong number of arguments for 'incr'\n"
				break
			}
			key := parts[1]
			val, err := s.storage.Increment(key)
			if err != nil {
				response = "(error) ERR value is not an integer or out of range\n"
				break
			}
			response = fmt.Sprintf("%d\n", val)

		case "QUIT":
			log.Info("Client QUIT")
			conn.Write([]byte("Bye!\n"))
			return

		default:
			response = fmt.Sprintf("(error) ERR unknown command '%s'\n", cmd)
		}

		conn.Write([]byte(response))
	}

	log.Info("Connection closed")
}
