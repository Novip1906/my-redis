package network

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/Novip1906/my-redis/internal/compute"
)

type TCPServer struct {
	wg       sync.WaitGroup
	port     string
	parser   *compute.Parser
	log      *slog.Logger
	listener net.Listener
}

func NewTCPServer(port string, parser *compute.Parser, log *slog.Logger) *TCPServer {
	return &TCPServer{
		port:   port,
		parser: parser,
		log:    log,
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

		response := s.parser.ProcessCommand(commandLine + "\n")

		response += "\n"

		if strings.HasPrefix(strings.ToUpper(commandLine), "QUIT") {
			conn.Write([]byte(response))
			return
		}

		conn.Write([]byte(response))
	}

	log.Info("Connection closed")
}
