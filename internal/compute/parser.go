package compute

import (
	"fmt"
	"strconv"
	"strings"
)

type Storage interface {
	Set(key, value string)
	Get(key string) (string, bool)
	Delete(key string)
	SetTTL(key string, seconds int64) bool
	GetTTL(key string) int64
	Increment(key string) (int64, error)
	Flush()
}

type Parser struct {
	storage Storage
}

func NewParser(storage Storage) *Parser {
	return &Parser{
		storage: storage,
	}
}

func (p *Parser) ProcessCommand(commandLine string) (response string) {
	parts := strings.Fields(commandLine)
	if len(parts) == 0 {
		return ""
	}

	cmd := strings.ToUpper(parts[0])

	switch cmd {
	case "SET":
		if len(parts) < 3 {
			return "(error) ERR wrong number of arguments for 'set'"
		}
		key := parts[1]
		val := strings.Join(parts[2:], " ")
		p.storage.Set(key, val)
		return "OK"

	case "GET":
		if len(parts) != 2 {
			return "(error) ERR wrong number of arguments for 'get'"
		}
		key := parts[1]
		val, ok := p.storage.Get(key)
		if !ok {
			return "(nil)"
		} else {
			return val
		}

	case "DEL":
		if len(parts) != 2 {
			return "(error) ERR wrong number of arguments for 'del'"
		}
		key := parts[1]
		p.storage.Delete(key)
		return "OK"

	case "EXPIRE":
		if len(parts) != 3 {
			return "(error) ERR wrong number of arguments for 'expire'"
		}
		key := parts[1]
		seconds, err := strconv.Atoi(parts[2])
		if err != nil {
			return "(error) ERR value is not an integer or out of range"
		}

		ok := p.storage.SetTTL(key, int64(seconds))
		if ok {
			return "1"
		} else {
			return "0"
		}

	case "TTL":
		if len(parts) != 2 {
			return "(error) ERR wrong number of arguments for 'ttl'"
		}
		key := parts[1]
		seconds := p.storage.GetTTL(key)
		return fmt.Sprintf("%d", seconds)

	case "INCR":
		if len(parts) != 2 {
			return "(error) ERR wrong number of arguments for 'incr'"
		}
		key := parts[1]
		val, err := p.storage.Increment(key)
		if err != nil {
			return "(error) ERR value is not an integer or out of range"
		}
		return fmt.Sprintf("%d", val)

	case "FLUSH":
		p.storage.Flush()
		return "OK"

	case "QUIT":
		return "Bye!"

	default:
		return fmt.Sprintf("(error) ERR unknown command '%s'\n", cmd)
	}

}
