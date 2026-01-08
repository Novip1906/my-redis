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

func (p *Parser) ProcessCommand(commandLine string) (response string, saveToAOF bool) {
	parts := strings.Fields(commandLine)
	if len(parts) == 0 {
		return "", false
	}

	cmd := strings.ToUpper(parts[0])

	switch cmd {
	case "SET":
		if len(parts) < 3 {
			return "(error) ERR wrong number of arguments for 'set'", false
		}
		key := parts[1]
		val := strings.Join(parts[2:], " ")
		p.storage.Set(key, val)
		return "OK", true

	case "GET":
		if len(parts) != 2 {
			return "(error) ERR wrong number of arguments for 'get'", false
		}
		key := parts[1]
		val, ok := p.storage.Get(key)
		if !ok {
			return "(nil)", false
		} else {
			return val, false
		}

	case "DEL":
		if len(parts) != 2 {
			return "(error) ERR wrong number of arguments for 'del'", false
		}
		key := parts[1]
		p.storage.Delete(key)
		return "OK", true

	case "EXPIRE":
		if len(parts) != 3 {
			return "(error) ERR wrong number of arguments for 'expire'", false
		}
		key := parts[1]
		seconds, err := strconv.Atoi(parts[2])
		if err != nil {
			return "(error) ERR value is not an integer or out of range", false
		}

		ok := p.storage.SetTTL(key, int64(seconds))
		if ok {
			return "1", true
		} else {
			return "0", false
		}

	case "TTL":
		if len(parts) != 2 {
			return "(error) ERR wrong number of arguments for 'ttl'", false
		}
		key := parts[1]
		seconds := p.storage.GetTTL(key)
		return fmt.Sprintf("%d", seconds), false

	case "INCR":
		if len(parts) != 2 {
			return "(error) ERR wrong number of arguments for 'incr'", false
		}
		key := parts[1]
		val, err := p.storage.Increment(key)
		if err != nil {
			return "(error) ERR value is not an integer or out of range", false
		}
		return fmt.Sprintf("%d", val), true

	case "FLUSH":
		p.storage.Flush()
		return "OK", true

	case "QUIT":
		return "Bye!", false

	default:
		return fmt.Sprintf("(error) ERR unknown command '%s'\n", cmd), false
	}

}
