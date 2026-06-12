package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	serverdataResponseValue = 0
	serverdataExecCommand   = 2
	serverdataAuthResponse  = 2
	serverdataAuth          = 3
)

type iniConfig struct {
	Language string
	Host     string
	Port     int
	Password string
}

type packet struct {
	id   int32
	typ  int32
	body []byte
}

type rconClient struct {
	conn        net.Conn
	nextIDValue int32
	tailTimeout time.Duration
}

type repeatedCommands []string

func (v *repeatedCommands) String() string {
	return strings.Join(*v, ";")
}

func (v *repeatedCommands) Set(value string) error {
	command := strings.TrimSpace(value)
	if command != "" {
		*v = append(*v, command)
	}
	return nil
}

func main() {
	os.Exit(run())
}

func run() int {
	configFlag := flag.String("config", "", "path to scum_rcon.ini")
	commandFlag := flag.String("command", "", "command to send")
	commandsFlag := flag.String("commands", "", "commands separated by semicolon or newline")
	var cmdFlags repeatedCommands
	flag.Var(&cmdFlags, "cmd", "command to send; repeat for multiple commands")
	hostFlag := flag.String("host", "", "override host")
	portFlag := flag.Int("port", 0, "override port")
	passwordFlag := flag.String("password", "", "override password")
	timeoutMS := flag.Int("timeout-ms", 3000, "connect/auth timeout")
	tailMS := flag.Int("tail-timeout-ms", 750, "response tail timeout")
	flag.Parse()

	exePath, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR|exe|"+err.Error())
		return 1
	}
	exeDir := filepath.Dir(exePath)
	configPath := strings.TrimSpace(*configFlag)
	if configPath == "" {
		configPath = filepath.Join(exeDir, "ini", "scum_rcon.ini")
	}

	cfg, err := loadOrCreateConfig(configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR|config|"+err.Error())
		return 2
	}

	if strings.TrimSpace(*hostFlag) != "" {
		cfg.Host = strings.TrimSpace(*hostFlag)
	}
	if *portFlag != 0 {
		cfg.Port = *portFlag
	}
	if strings.TrimSpace(*passwordFlag) != "" {
		cfg.Password = *passwordFlag
	}

	commands, err := collectCommands(*commandFlag, *commandsFlag, cmdFlags, flag.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR|input|"+err.Error())
		return 2
	}
	if len(commands) == 0 {
		commands, err = readCommandsFromStdin()
		if err != nil {
			fmt.Fprintln(os.Stderr, "ERROR|stdin|"+err.Error())
			return 2
		}
	}
	if len(commands) == 0 {
		return interactiveSession(cfg, time.Duration(*timeoutMS)*time.Millisecond, time.Duration(*tailMS)*time.Millisecond)
	}

	client, err := dialRcon(cfg.Host, cfg.Port, cfg.Password, time.Duration(*timeoutMS)*time.Millisecond, time.Duration(*tailMS)*time.Millisecond)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR|rcon|"+err.Error())
		return 1
	}
	defer client.close()

	for _, cmd := range commands {
		out, err := client.command(cmd)
		if err != nil {
			fmt.Fprintln(os.Stderr, "ERROR|command|"+err.Error())
			return 1
		}
		if out != "" {
			fmt.Print(out)
			if !strings.HasSuffix(out, "\n") {
				fmt.Println()
			}
		}
	}

	return 0
}

func interactiveSession(cfg iniConfig, timeout, tailTimeout time.Duration) int {
	client, err := dialRcon(cfg.Host, cfg.Port, cfg.Password, timeout, tailTimeout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR|rcon|"+err.Error())
		return 1
	}
	defer client.close()

	fmt.Printf("Connected to %s:%d\n", cfg.Host, cfg.Port)
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("rcon> ")
		if !scanner.Scan() {
			fmt.Println()
			return 0
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		switch strings.ToLower(line) {
		case "exit", "quit", "q":
			return 0
		}

		out, err := client.command(line)
		if err != nil {
			fmt.Fprintln(os.Stderr, "ERROR|command|"+err.Error())
			return 1
		}
		if out != "" {
			fmt.Print(out)
			if !strings.HasSuffix(out, "\n") {
				fmt.Println()
			}
		}
	}
}

func collectCommands(command string, commandsText string, cmdFlags repeatedCommands, args []string) ([]string, error) {
	var commands []string
	for _, cmd := range cmdFlags {
		if cmd = strings.TrimSpace(cmd); cmd != "" {
			commands = append(commands, cmd)
		}
	}
	commands = append(commands, splitCommandList(commandsText)...)
	if cmd := strings.TrimSpace(command); cmd != "" {
		commands = append(commands, cmd)
	}

	if len(commands) > 0 {
		return commands, nil
	}
	if len(args) > 0 {
		if cmd := strings.TrimSpace(strings.Join(args, " ")); cmd != "" {
			if commands := splitCommandList(cmd); len(commands) > 0 {
				return commands, nil
			}
			return []string{cmd}, nil
		}
	}
	return nil, nil
}

func splitCommandList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == ';' || r == '\n' || r == '\r'
	})
	commands := make([]string, 0, len(fields))
	for _, field := range fields {
		if cmd := strings.TrimSpace(field); cmd != "" {
			commands = append(commands, cmd)
		}
	}
	return commands
}

func readCommandsFromStdin() ([]string, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}
	if stat.Mode()&os.ModeCharDevice != 0 {
		return nil, nil
	}

	var commands []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			commands = append(commands, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return commands, nil
}

func loadOrCreateConfig(path string) (iniConfig, error) {
	cfg := iniConfig{
		Language: "en",
		Host:     "SET_IP",
		Port:     9010,
		Password: "SET_PASSWORD",
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return cfg, err
			}
			if err := os.WriteFile(path, []byte(defaultINI()), 0o644); err != nil {
				return cfg, err
			}
			return cfg, fmt.Errorf("created template at %s", path)
		}
		return cfg, err
	}

	section := ""
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = trimInlineComment(strings.TrimSpace(value))

		switch section {
		case "ui":
			if key == "language" && value != "" {
				cfg.Language = value
			}
		case "server":
			switch key {
			case "host":
				cfg.Host = value
			case "port":
				if n, err := strconv.Atoi(value); err == nil && n > 0 {
					cfg.Port = n
				}
			case "password":
				cfg.Password = value
			}
		}
	}

	if isPlaceholder(cfg.Host) || cfg.Port <= 0 || isPlaceholder(cfg.Password) {
		return cfg, fmt.Errorf("set host, port, and password in %s", path)
	}
	return cfg, nil
}

func defaultINI() string {
	return strings.Join([]string{
		"# SCUM RCON configuration",
		"",
		"[ui]",
		"language = en",
		"",
		"[server]",
		"host = SET_IP",
		"port = 9010",
		"password = SET_PASSWORD",
		"",
	}, "\n")
}

func trimInlineComment(value string) string {
	for _, marker := range []string{" #", " ;", "\t#", "\t;"} {
		if idx := strings.Index(value, marker); idx >= 0 {
			value = strings.TrimSpace(value[:idx])
		}
	}
	return strings.TrimSpace(value)
}

func isPlaceholder(value string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	return normalized == "" || strings.HasPrefix(normalized, "SET_")
}

func dialRcon(host string, port int, password string, timeout, tailTimeout time.Duration) (*rconClient, error) {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), timeout)
	if err != nil {
		return nil, err
	}

	client := &rconClient{
		conn:        conn,
		nextIDValue: int32(rand.Intn(1000) + 100),
		tailTimeout: tailTimeout,
	}
	_ = conn.SetDeadline(time.Now().Add(timeout))
	authID := client.nextID()
	if err := client.writePacket(authID, serverdataAuth, password); err != nil {
		conn.Close()
		return nil, err
	}

	for {
		pkt, err := client.readPacket()
		if err != nil {
			conn.Close()
			return nil, err
		}
		if pkt.typ != serverdataAuthResponse {
			continue
		}
		if pkt.id == -1 {
			conn.Close()
			return nil, errors.New("RCON authentication failed")
		}
		if pkt.id == authID {
			_ = conn.SetDeadline(time.Time{})
			return client, nil
		}
	}
}

func (c *rconClient) nextID() int32 {
	c.nextIDValue++
	return c.nextIDValue
}

func (c *rconClient) close() {
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

func (c *rconClient) writePacket(id, typ int32, text string) error {
	body := append([]byte(text), 0, 0)
	size := int32(len(body) + 8)

	var buf bytes.Buffer
	for _, value := range []int32{size, id, typ} {
		if err := binary.Write(&buf, binary.LittleEndian, value); err != nil {
			return err
		}
	}
	buf.Write(body)

	_, err := c.conn.Write(buf.Bytes())
	return err
}

func (c *rconClient) readPacket() (*packet, error) {
	var size int32
	if err := binary.Read(c.conn, binary.LittleEndian, &size); err != nil {
		return nil, err
	}
	if size < 10 {
		return nil, fmt.Errorf("invalid RCON packet size: %d", size)
	}

	payload := make([]byte, size)
	if _, err := io.ReadFull(c.conn, payload); err != nil {
		return nil, err
	}

	return &packet{
		id:   int32(binary.LittleEndian.Uint32(payload[0:4])),
		typ:  int32(binary.LittleEndian.Uint32(payload[4:8])),
		body: payload[8 : len(payload)-2],
	}, nil
}

func (c *rconClient) command(command string) (string, error) {
	commandID := c.nextID()
	markerID := c.nextID()

	if err := c.writePacket(commandID, serverdataExecCommand, command); err != nil {
		return "", err
	}
	if err := c.writePacket(markerID, serverdataResponseValue, ""); err != nil {
		return "", err
	}

	_ = c.conn.SetReadDeadline(time.Now().Add(c.tailTimeout))
	var chunks []string
	for {
		pkt, err := c.readPacket()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				break
			}
			return strings.TrimRight(strings.Join(chunks, ""), "\r\n\t "), err
		}
		if pkt.id == markerID {
			break
		}
		if pkt.id == commandID || pkt.id == 0 || pkt.id == 1 {
			chunks = append(chunks, string(pkt.body))
		}
	}
	_ = c.conn.SetReadDeadline(time.Time{})
	return strings.TrimRight(strings.Join(chunks, ""), "\r\n\t "), nil
}
