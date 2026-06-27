package system

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestDeviceServiceNoopsWhenRedisUnavailable(t *testing.T) {
	originalRedis := global.GVA_REDIS
	global.GVA_REDIS = nil
	t.Cleanup(func() {
		global.GVA_REDIS = originalRedis
	})

	require.NoError(t, (&DeviceService{}).MarkHeartbeat("9130dbc0"))
	require.NoError(t, (&DeviceService{}).MarkBusy("9130dbc0", "phone_register"))
	require.NoError(t, (&DeviceService{}).ClearBusy("9130dbc0"))
	require.NoError(t, (&DeviceService{}).MarkOffline("9130dbc0"))
	require.Empty(t, (&DeviceService{}).ListOnlineDeviceIDs())
	require.Empty(t, (&DeviceService{}).ListBusyDeviceIDs())
}

func TestListOnlineDeviceIDsIgnoresStaleHeartbeatTimestamps(t *testing.T) {
	now := time.Now()
	server := newFakeRedisServer(t, map[string]string{
		deviceHeartbeatKey("fresh-device"): strconv.FormatInt(now.Unix(), 10),
		deviceHeartbeatKey("stale-device"): strconv.FormatInt(now.Add(-time.Hour).Unix(), 10),
	})
	originalRedis := global.GVA_REDIS
	global.GVA_REDIS = redis.NewClient(&redis.Options{Addr: server.addr, Protocol: 2})
	t.Cleanup(func() {
		_ = global.GVA_REDIS.Close()
		global.GVA_REDIS = originalRedis
		server.close()
	})

	devices := (&DeviceService{}).ListOnlineDeviceIDs()

	require.ElementsMatch(t, []string{"fresh-device"}, devices)
}

func TestUpdateBusyIfMatchingReturnsErrorWhenBusyValueChanged(t *testing.T) {
	server := newFakeRedisServer(t, map[string]string{
		deviceBusyKey("9130dbc0"): "other_business",
	})
	originalRedis := global.GVA_REDIS
	global.GVA_REDIS = redis.NewClient(&redis.Options{Addr: server.addr, Protocol: 2})
	t.Cleanup(func() {
		_ = global.GVA_REDIS.Close()
		global.GVA_REDIS = originalRedis
		server.close()
	})

	err := (&DeviceService{}).UpdateBusyIfMatching("9130dbc0", "reservation_token", "phone_register_reserved:1", time.Minute)

	require.Error(t, err)
	require.Contains(t, err.Error(), "busy状态已变更")
	require.Equal(t, 0, server.count("set"))
}

func TestDeviceStateChangeInvalidatesPhoneRegisterDeviceStatsCache(t *testing.T) {
	server := newFakeRedisServer(t, nil)
	originalRedis := global.GVA_REDIS
	global.GVA_REDIS = redis.NewClient(&redis.Options{Addr: server.addr, Protocol: 2})
	t.Cleanup(func() {
		_ = global.GVA_REDIS.Close()
		global.GVA_REDIS = originalRedis
		server.close()
		resetPhoneRegisterDeviceStatsCache()
	})

	phoneRegisterDeviceStatsCache.Lock()
	phoneRegisterDeviceStatsCache.stats = phoneRegisterDeviceStats{Online: 2, Idle: 1}
	phoneRegisterDeviceStatsCache.expiresAt = time.Now().Add(time.Minute)
	phoneRegisterDeviceStatsCache.Unlock()

	require.NoError(t, (&DeviceService{}).MarkBusy("9130dbc0", "phone_register"))

	phoneRegisterDeviceStatsCache.Lock()
	expiresAt := phoneRegisterDeviceStatsCache.expiresAt
	phoneRegisterDeviceStatsCache.Unlock()
	require.True(t, expiresAt.IsZero())
}

type fakeRedisServer struct {
	addr   string
	ln     net.Listener
	mu     sync.Mutex
	values map[string]string
	counts chan string
}

func newFakeRedisServer(t *testing.T, values map[string]string) *fakeRedisServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	s := &fakeRedisServer{
		addr:   ln.Addr().String(),
		ln:     ln,
		values: values,
		counts: make(chan string, 16),
	}
	if s.values == nil {
		s.values = map[string]string{}
	}
	go s.serve()
	return s
}

func (s *fakeRedisServer) close() {
	_ = s.ln.Close()
}

func (s *fakeRedisServer) count(command string) int {
	total := 0
	for {
		select {
		case got := <-s.counts:
			if got == command {
				total++
			}
		default:
			return total
		}
	}
}

func (s *fakeRedisServer) serve() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handle(conn)
	}
}

func (s *fakeRedisServer) handle(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		args, err := readRESPArray(reader)
		if err != nil {
			return
		}
		if len(args) == 0 {
			_, _ = conn.Write([]byte("-ERR empty command\r\n"))
			continue
		}
		cmd := strings.ToLower(args[0])
		select {
		case s.counts <- cmd:
		default:
		}
		switch cmd {
		case "hello":
			_, _ = conn.Write([]byte("%7\r\n+server\r\n+redis\r\n+version\r\n+7.0.0\r\n+proto\r\n:3\r\n+id\r\n:1\r\n+mode\r\n+standalone\r\n+role\r\n+master\r\n+modules\r\n*0\r\n"))
		case "get":
			s.mu.Lock()
			value, ok := s.values[args[1]]
			s.mu.Unlock()
			if !ok {
				_, _ = conn.Write([]byte("$-1\r\n"))
				continue
			}
			_, _ = fmt.Fprintf(conn, "$%d\r\n%s\r\n", len(value), value)
		case "mget":
			s.mu.Lock()
			values := make([]*string, 0, len(args)-1)
			for _, key := range args[1:] {
				if value, ok := s.values[key]; ok {
					v := value
					values = append(values, &v)
				} else {
					values = append(values, nil)
				}
			}
			s.mu.Unlock()
			_, _ = fmt.Fprintf(conn, "*%d\r\n", len(values))
			for _, value := range values {
				if value == nil {
					_, _ = conn.Write([]byte("$-1\r\n"))
					continue
				}
				_, _ = fmt.Fprintf(conn, "$%d\r\n%s\r\n", len(*value), *value)
			}
		case "set":
			if len(args) >= 3 {
				s.mu.Lock()
				s.values[args[1]] = args[2]
				s.mu.Unlock()
			}
			_, _ = conn.Write([]byte("+OK\r\n"))
		case "del":
			var deleted int
			if len(args) > 1 {
				s.mu.Lock()
				for _, key := range args[1:] {
					if _, ok := s.values[key]; ok {
						deleted++
						delete(s.values, key)
					}
				}
				s.mu.Unlock()
			}
			_, _ = fmt.Fprintf(conn, ":%d\r\n", deleted)
		case "scan":
			pattern := ""
			for i := 2; i+1 < len(args); i += 2 {
				if strings.EqualFold(args[i], "match") {
					pattern = args[i+1]
					break
				}
			}
			prefix := strings.TrimSuffix(pattern, "*")
			s.mu.Lock()
			keys := make([]string, 0, len(s.values))
			for key := range s.values {
				if prefix == "" || strings.HasPrefix(key, prefix) {
					keys = append(keys, key)
				}
			}
			s.mu.Unlock()
			sort.Strings(keys)
			_, _ = conn.Write([]byte("*2\r\n$1\r\n0\r\n"))
			_, _ = fmt.Fprintf(conn, "*%d\r\n", len(keys))
			for _, key := range keys {
				_, _ = fmt.Fprintf(conn, "$%d\r\n%s\r\n", len(key), key)
			}
		case "client":
			_, _ = conn.Write([]byte("+OK\r\n"))
		default:
			_, _ = conn.Write([]byte("+OK\r\n"))
		}
	}
}

func readRESPArray(reader *bufio.Reader) ([]string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(line, "*") {
		return nil, fmt.Errorf("unexpected RESP line %q", line)
	}
	var count int
	if _, err := fmt.Sscanf(strings.TrimSpace(line), "*%d", &count); err != nil {
		return nil, err
	}
	args := make([]string, 0, count)
	for i := 0; i < count; i++ {
		bulkHeader, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		var length int
		if _, err := fmt.Sscanf(strings.TrimSpace(bulkHeader), "$%d", &length); err != nil {
			return nil, err
		}
		buf := make([]byte, length+2)
		if _, err := io.ReadFull(reader, buf); err != nil {
			return nil, err
		}
		args = append(args, string(buf[:length]))
	}
	return args, nil
}
