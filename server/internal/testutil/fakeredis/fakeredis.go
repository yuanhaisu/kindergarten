// Package fakeredis 提供进程内的最小 Redis 替身，供认证、限流与平台会话测试使用。
//
// 为什么不用真实 Redis：这些测试验证的是「键怎么写、TTL 是否带上、存储故障
// 时是否失败关闭」，不需要外部依赖，也不应污染开发库里真实的会话与黑名单。
//
// 只实现测试用到的命令：PING / SET(EX|PX) / GET / GETDEL / DEL / EXISTS /
// TTL / INCR / EXPIRE。未知命令统一回 +OK，让 go-redis 的连接握手
// （CLIENT SETINFO 等）不至于失败。
//
// 未知命令回 +OK 意味着「忘记实现」会静默变成错误语义——新增命令必须补这里。
package fakeredis

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

type value struct {
	data      string
	expiresAt time.Time // 零值表示永不过期
}

// Server 是一个监听随机端口的假 Redis 实例。
type Server struct {
	ln         net.Listener
	mu         sync.Mutex
	data       map[string]value
	failReads  bool
	failWrites bool
}

// New 启动一个假 Redis，测试结束时自动关闭。
func New(t testing.TB) *Server {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("fakeredis: listen: %v", err)
	}
	s := &Server{ln: ln, data: map[string]value{}}
	go s.serve()
	t.Cleanup(func() { _ = ln.Close() })
	return s
}

// Addr 返回监听地址（host:port）。
func (s *Server) Addr() string { return s.ln.Addr().String() }

// Client 返回指向假 Redis 的 go-redis 客户端。
func (s *Server) Client(t testing.TB) *redis.Client {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{
		Addr:            s.Addr(),
		Protocol:        2,
		DisableIdentity: true,
		PoolSize:        1,
		MaxRetries:      -1,
		DialTimeout:     time.Second,
		ReadTimeout:     time.Second,
		WriteTimeout:    time.Second,
	})
	t.Cleanup(func() { _ = rdb.Close() })
	return rdb
}

// SetFailReads 让后续读命令返回错误，用于验证失败关闭。
func (s *Server) SetFailReads(v bool) {
	s.mu.Lock()
	s.failReads = v
	s.mu.Unlock()
}

// SetFailWrites 让后续写命令返回错误，用于验证失败关闭。
func (s *Server) SetFailWrites(v bool) {
	s.mu.Lock()
	s.failWrites = v
	s.mu.Unlock()
}

// Get 返回键的原始值，用于断言存储格式。
func (s *Server) Get(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.lookupLocked(key)
	if !ok {
		return "", false
	}
	return v.data, true
}

// Exists 返回键是否存在以及剩余 TTL（-1 表示永不过期）。
func (s *Server) Exists(key string) (bool, time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.lookupLocked(key)
	if !ok {
		return false, 0
	}
	if v.expiresAt.IsZero() {
		return true, -1
	}
	return true, time.Until(v.expiresAt)
}

func (s *Server) lookupLocked(key string) (value, bool) {
	v, ok := s.data[key]
	if !ok {
		return value{}, false
	}
	if !v.expiresAt.IsZero() && time.Now().After(v.expiresAt) {
		delete(s.data, key)
		return value{}, false
	}
	return v, true
}

func (s *Server) serve() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	r := bufio.NewReader(conn)
	for {
		cmd, err := readRESPArray(r)
		if err != nil {
			return
		}
		s.dispatch(conn, cmd)
	}
}

func (s *Server) dispatch(conn net.Conn, cmd []string) {
	if len(cmd) == 0 {
		_, _ = conn.Write([]byte("+OK\r\n"))
		return
	}
	switch strings.ToUpper(cmd[0]) {
	case "HELLO":
		// go-redis 只在非 RedisError 时判定连接失败；回 ERR 即退回 RESP2。
		_, _ = conn.Write([]byte("-ERR unknown command 'hello'\r\n"))
	case "PING":
		_, _ = conn.Write([]byte("+PONG\r\n"))
	case "SET":
		s.cmdSet(conn, cmd)
	case "GET":
		s.cmdGet(conn, cmd)
	case "GETDEL":
		s.cmdGetDel(conn, cmd)
	case "DEL":
		s.cmdDel(conn, cmd)
	case "EXISTS":
		s.cmdExists(conn, cmd)
	case "TTL":
		s.cmdTTL(conn, cmd)
	case "INCR":
		s.cmdIncr(conn, cmd)
	case "EXPIRE":
		s.cmdExpire(conn, cmd)
	default:
		_, _ = conn.Write([]byte("+OK\r\n"))
	}
}

func (s *Server) cmdSet(conn net.Conn, cmd []string) {
	if len(cmd) < 3 {
		_, _ = conn.Write([]byte("-ERR wrong number of arguments for 'set'\r\n"))
		return
	}
	s.mu.Lock()
	fail := s.failWrites
	s.mu.Unlock()
	if fail {
		_, _ = conn.Write([]byte("-ERR simulated write failure\r\n"))
		return
	}

	v := value{data: cmd[2]}
	now := time.Now()
	for i := 3; i+1 < len(cmd); i++ {
		n, err := strconv.Atoi(cmd[i+1])
		if err != nil {
			continue
		}
		switch strings.ToUpper(cmd[i]) {
		case "EX":
			v.expiresAt = now.Add(time.Duration(n) * time.Second)
		case "PX":
			v.expiresAt = now.Add(time.Duration(n) * time.Millisecond)
		}
		i++
	}

	s.mu.Lock()
	s.data[cmd[1]] = v
	s.mu.Unlock()
	_, _ = conn.Write([]byte("+OK\r\n"))
}

func (s *Server) cmdGet(conn net.Conn, cmd []string) {
	if len(cmd) < 2 {
		_, _ = conn.Write([]byte("-ERR wrong number of arguments for 'get'\r\n"))
		return
	}
	s.mu.Lock()
	fail := s.failReads
	v, ok := s.lookupLocked(cmd[1])
	s.mu.Unlock()
	if fail {
		_, _ = conn.Write([]byte("-ERR simulated read failure\r\n"))
		return
	}
	if !ok {
		_, _ = conn.Write([]byte("$-1\r\n"))
		return
	}
	_, _ = fmt.Fprintf(conn, "$%d\r\n%s\r\n", len(v.data), v.data)
}

// cmdGetDel 在同一把锁里取值并删除，模拟 Redis GETDEL 的原子性（#169）。
func (s *Server) cmdGetDel(conn net.Conn, cmd []string) {
	if len(cmd) < 2 {
		_, _ = conn.Write([]byte("-ERR wrong number of arguments for 'getdel'\r\n"))
		return
	}
	s.mu.Lock()
	fail := s.failReads || s.failWrites
	v, ok := s.lookupLocked(cmd[1])
	if ok && !fail {
		delete(s.data, cmd[1])
	}
	s.mu.Unlock()
	if fail {
		_, _ = conn.Write([]byte("-ERR simulated getdel failure\r\n"))
		return
	}
	if !ok {
		_, _ = conn.Write([]byte("$-1\r\n"))
		return
	}
	_, _ = fmt.Fprintf(conn, "$%d\r\n%s\r\n", len(v.data), v.data)
}

func (s *Server) cmdDel(conn net.Conn, cmd []string) {
	s.mu.Lock()
	fail := s.failWrites
	n := 0
	if !fail {
		for _, key := range cmd[1:] {
			if _, ok := s.lookupLocked(key); ok {
				delete(s.data, key)
				n++
			}
		}
	}
	s.mu.Unlock()
	if fail {
		_, _ = conn.Write([]byte("-ERR simulated write failure\r\n"))
		return
	}
	_, _ = fmt.Fprintf(conn, ":%d\r\n", n)
}

func (s *Server) cmdExists(conn net.Conn, cmd []string) {
	s.mu.Lock()
	fail := s.failReads
	n := 0
	if !fail {
		for _, key := range cmd[1:] {
			if _, ok := s.lookupLocked(key); ok {
				n++
			}
		}
	}
	s.mu.Unlock()
	if fail {
		_, _ = conn.Write([]byte("-ERR simulated read failure\r\n"))
		return
	}
	_, _ = fmt.Fprintf(conn, ":%d\r\n", n)
}

func (s *Server) cmdTTL(conn net.Conn, cmd []string) {
	if len(cmd) < 2 {
		_, _ = conn.Write([]byte("-ERR wrong number of arguments for 'ttl'\r\n"))
		return
	}
	s.mu.Lock()
	fail := s.failReads
	v, ok := s.lookupLocked(cmd[1])
	s.mu.Unlock()
	if fail {
		_, _ = conn.Write([]byte("-ERR simulated read failure\r\n"))
		return
	}
	switch {
	case !ok:
		_, _ = conn.Write([]byte(":-2\r\n"))
	case v.expiresAt.IsZero():
		_, _ = conn.Write([]byte(":-1\r\n"))
	default:
		_, _ = fmt.Fprintf(conn, ":%d\r\n", int64(time.Until(v.expiresAt)/time.Second))
	}
}

func (s *Server) cmdIncr(conn net.Conn, cmd []string) {
	if len(cmd) < 2 {
		_, _ = conn.Write([]byte("-ERR wrong number of arguments for 'incr'\r\n"))
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failWrites {
		_, _ = conn.Write([]byte("-ERR simulated write failure\r\n"))
		return
	}
	cur := int64(0)
	if v, ok := s.lookupLocked(cmd[1]); ok {
		n, err := strconv.ParseInt(v.data, 10, 64)
		if err != nil {
			_, _ = conn.Write([]byte("-ERR value is not an integer or out of range\r\n"))
			return
		}
		cur = n
	}
	cur++
	// 保留原有 TTL：计数窗口内续期请求不应重置窗口。
	next := value{data: strconv.FormatInt(cur, 10)}
	if v, ok := s.data[cmd[1]]; ok {
		next.expiresAt = v.expiresAt
	}
	s.data[cmd[1]] = next
	_, _ = fmt.Fprintf(conn, ":%d\r\n", cur)
}

func (s *Server) cmdExpire(conn net.Conn, cmd []string) {
	if len(cmd) < 3 {
		_, _ = conn.Write([]byte("-ERR wrong number of arguments for 'expire'\r\n"))
		return
	}
	sec, err := strconv.Atoi(cmd[2])
	if err != nil {
		_, _ = conn.Write([]byte("-ERR value is not an integer or out of range\r\n"))
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failWrites {
		_, _ = conn.Write([]byte("-ERR simulated write failure\r\n"))
		return
	}
	v, ok := s.lookupLocked(cmd[1])
	if !ok {
		_, _ = conn.Write([]byte(":0\r\n"))
		return
	}
	v.expiresAt = time.Now().Add(time.Duration(sec) * time.Second)
	s.data[cmd[1]] = v
	_, _ = conn.Write([]byte(":1\r\n"))
}

func readRESPArray(r *bufio.Reader) ([]string, error) {
	head, err := readCRLFLine(r)
	if err != nil {
		return nil, err
	}
	if len(head) == 0 || head[0] != '*' {
		return nil, fmt.Errorf("expected RESP array, got %q", head)
	}
	n, err := strconv.Atoi(head[1:])
	if err != nil {
		return nil, err
	}
	cmd := make([]string, 0, n)
	for i := 0; i < n; i++ {
		line, err := readCRLFLine(r)
		if err != nil {
			return nil, err
		}
		if len(line) == 0 || line[0] != '$' {
			return nil, fmt.Errorf("expected bulk string, got %q", line)
		}
		size, err := strconv.Atoi(line[1:])
		if err != nil {
			return nil, err
		}
		buf := make([]byte, size+2)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		cmd = append(cmd, string(buf[:size]))
	}
	return cmd, nil
}

func readCRLFLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), nil
}
