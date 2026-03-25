package companion

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type stateStore interface {
	SaveJSON(ctx context.Context, key string, value any) error
	Status() string
}

type noopStateStore struct {
	status string
}

func (s *noopStateStore) SaveJSON(_ context.Context, _ string, _ any) error {
	return nil
}

func (s *noopStateStore) Status() string {
	return s.status
}

type redisStateStore struct {
	addr      string
	password  string
	db        int
	namespace string
	timeout   time.Duration
	status    string
}

func newStateStore(ctx context.Context, cfg RedisConfig) stateStore {
	if !cfg.Enabled {
		return &noopStateStore{status: "redis disabled"}
	}

	store, err := newRedisStateStore(cfg)
	if err == nil {
		if pingErr := store.ping(ctx); pingErr == nil {
			store.status = "redis connected"
			return store
		}
	}

	if cfg.AutoStartContainer {
		_ = ensureRedisContainer(ctx, cfg)
		store, err = newRedisStateStore(cfg)
		if err == nil {
			if pingErr := store.ping(ctx); pingErr == nil {
				store.status = "redis connected via docker"
				return store
			}
		}
	}

	status := "redis unavailable, using in-memory state"
	if err != nil {
		status = status + ": " + err.Error()
	}
	return &noopStateStore{status: status}
}

func newRedisStateStore(cfg RedisConfig) (*redisStateStore, error) {
	rawURL := strings.TrimSpace(cfg.URL)
	if rawURL == "" {
		rawURL = "redis://127.0.0.1:6379/0"
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	addr := parsed.Host
	if !strings.Contains(addr, ":") {
		addr += ":6379"
	}

	password := ""
	if parsed.User != nil {
		password, _ = parsed.User.Password()
	}

	db := 0
	if parsed.Path != "" && parsed.Path != "/" {
		db, _ = strconv.Atoi(strings.TrimPrefix(parsed.Path, "/"))
	}

	return &redisStateStore{
		addr:      addr,
		password:  password,
		db:        db,
		namespace: strings.TrimSpace(cfg.Namespace),
		timeout:   2 * time.Second,
	}, nil
}

func (s *redisStateStore) Status() string {
	return s.status
}

func (s *redisStateStore) SaveJSON(ctx context.Context, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.run(ctx, []string{"SET", s.prefixed(key), string(data)})
}

func (s *redisStateStore) ping(ctx context.Context) error {
	return s.run(ctx, []string{"PING"})
}

func (s *redisStateStore) prefixed(key string) string {
	if s.namespace == "" {
		return key
	}
	return s.namespace + ":" + key
}

func (s *redisStateStore) run(ctx context.Context, parts []string) error {
	dialer := net.Dialer{Timeout: s.timeout}
	conn, err := dialer.DialContext(ctx, "tcp", s.addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(s.timeout))

	if s.password != "" {
		if _, err := conn.Write([]byte(respArray("AUTH", s.password))); err != nil {
			return err
		}
		if _, err := readRESP(conn); err != nil {
			return err
		}
	}

	if s.db > 0 {
		if _, err := conn.Write([]byte(respArray("SELECT", strconv.Itoa(s.db)))); err != nil {
			return err
		}
		if _, err := readRESP(conn); err != nil {
			return err
		}
	}

	if _, err := conn.Write([]byte(respArray(parts...))); err != nil {
		return err
	}
	_, err = readRESP(conn)
	return err
}

func respArray(parts ...string) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("*%d\r\n", len(parts)))
	for _, part := range parts {
		builder.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(part), part))
	}
	return builder.String()
}

func readRESP(conn net.Conn) (string, error) {
	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil {
		return "", err
	}
	reply := string(buffer[:n])
	if strings.HasPrefix(reply, "-") {
		return reply, fmt.Errorf("%s", strings.TrimSpace(strings.TrimPrefix(reply, "-")))
	}
	return reply, nil
}

func ensureRedisContainer(ctx context.Context, cfg RedisConfig) error {
	containerName := strings.TrimSpace(cfg.ContainerName)
	if containerName == "" {
		containerName = "desk-companion-redis"
	}

	hostPort := "6379"
	if parsed, err := url.Parse(cfg.URL); err == nil && parsed.Host != "" {
		if _, port, splitErr := net.SplitHostPort(parsed.Host); splitErr == nil && port != "" {
			hostPort = port
		}
	}

	checkCmd := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Running}}", containerName)
	output, err := checkCmd.CombinedOutput()
	if err == nil {
		if strings.TrimSpace(string(output)) == "true" {
			return nil
		}
		startCmd := exec.CommandContext(ctx, "docker", "start", containerName)
		_, _ = startCmd.CombinedOutput()
		return nil
	}

	runCmd := exec.CommandContext(
		ctx,
		"docker", "run", "-d",
		"--name", containerName,
		"-p", hostPort+":6379",
		"redis:7-alpine",
	)
	_, runErr := runCmd.CombinedOutput()
	return runErr
}
