package util

import (
	"cmp"
	"context"
	"crypto/md5"
	"fmt"
	"iter"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

const MacOSChromeUA = "nezha-agent/1.0"

func IsWindows() bool {
	return os.PathSeparator == '\\' && os.PathListSeparator == ';'
}

func BrowserHeaders() http.Header {
	return http.Header{
		"Accept":          {"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8"},
		"Accept-Language": {"en,zh-CN;q=0.9,zh;q=0.8"},
		"User-Agent":      {MacOSChromeUA},
	}
}

func ContainsStr(slice []string, str string) bool {
	if str != "" {
		for _, item := range slice {
			if strings.Contains(str, item) {
				return true
			}
		}
	}
	return false
}

func RemoveDuplicate[S ~[]E, E cmp.Ordered](list S) S {
	if list == nil {
		return nil
	}
	out := make([]E, len(list))
	copy(out, list)
	slices.Sort(out)
	return slices.Compact(out)
}

func RotateQueue1(start, i, size int) int {
	return (start + i) % size
}

func RangeRnd[S ~[]E, E any](s S) iter.Seq2[int, E] {
	index := int(time.Now().Unix()) % len(s)
	return func(yield func(int, E) bool) {
		for i := range len(s) {
			r := RotateQueue1(index, i, len(s))
			if !yield(r, s[r]) {
				break
			}
		}
	}
}

// LookupIP looks up host using the local resolver.
// It returns a slice of that host's IPv4 and IPv6 addresses.
func LookupIP(host string) ([]net.IP, error) {
	defaultResolver := net.Resolver{PreferGo: true}
	addrs, err := defaultResolver.LookupIPAddr(context.Background(), host)
	if err != nil {
		return nil, err
	}
	ips := make([]net.IP, len(addrs))
	for i, ia := range addrs {
		ips[i] = ia.IP
	}
	return ips, nil
}

func SubUintChecked[T Unsigned](a, b T) T {
	if a < b {
		return 0
	}

	return a - b
}

func MD5Sum(str string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(str)))
}

type Unsigned interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

func FindProcessByCmd(executablePath string) []int32 {
	if executablePath == "" {
		return nil
	}

	target := normalizeExePath(executablePath)
	self := int32(os.Getpid())

	procs, err := process.Processes()
	if err != nil {
		return nil
	}

	var pids []int32
	for _, p := range procs {
		if p == nil {
			continue
		}
		pid := p.Pid
		if pid == self {
			continue
		}
		exe, err := p.Exe()
		if err != nil || exe == "" {
			continue
		}
		if normalizeExePath(exe) == target {
			pids = append(pids, pid)
		}
	}

	return pids
}

func KillProcesses(pids []int32) {
	for _, pid := range pids {
		if pid <= 0 {
			continue
		}
		if int(pid) == os.Getpid() {
			continue
		}
		p, err := os.FindProcess(int(pid))
		if err != nil {
			continue
		}
		_ = p.Kill()
	}
}

func normalizeExePath(p string) string {
	if p == "" {
		return ""
	}
	p = filepath.Clean(p)
	if IsWindows() {
		p = strings.ToLower(p)
	}
	return p
}
