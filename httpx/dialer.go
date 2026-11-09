package httpx

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"golang.org/x/sync/singleflight"
)

type dnsCacheEntry struct {
	ips    []net.IP
	expire time.Time
}

var (
	dnsCacheMap sync.Map
	dnsSF       singleflight.Group
	cacheTTL    = 10 * time.Second
	dnsServers  = []string{"119.29.29.29:53"} // 固定 IPv4 DNS，可加备用
)

func LookupIPWithCache(ctx context.Context, host string) ([]net.IP, error) {
	// 1. 读缓存
	entryRaw, ok := dnsCacheMap.Load(host)
	entry := entryRaw.(dnsCacheEntry)
	if ok && time.Now().Before(entry.expire) {
		return entry.ips, nil
	} else {
		dnsCacheMap.Delete(host)
	}

	// 2. singleflight 防止高并发重复查询
	v, err, _ := dnsSF.Do(host, func() (interface{}, error) {
		var lastErr error
		for _, server := range dnsServers {
			r := &net.Resolver{
				PreferGo: true,
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					d := &net.Dialer{Timeout: 2 * time.Second}
					return d.DialContext(ctx, "udp4", server)
				},
			}

			ips, err := r.LookupIP(ctx, "ip4", host) // 禁用 IPv6
			if err == nil && len(ips) > 0 {
				// 保存缓存
				dnsCacheMap.Store(host, dnsCacheEntry{
					ips:    ips,
					expire: time.Now().Add(cacheTTL),
				})
				return ips, nil
			}
			lastErr = err
		}
		if lastErr == nil {
			lastErr = errors.New("DNS lookup returned no IP")
		}
		return nil, lastErr
	})

	if err != nil {
		return nil, err
	}
	return v.([]net.IP), nil
}

func init() {
	return
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			// 这里用 LookupIPWithCache 返回的 IP 进行 TCP 连接
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				hlog.Debugf("=====foo111===%v", err)
				return nil, fmt.Errorf("httpx_err.0:%v", err)
			}

			ips, err := LookupIPWithCache(ctx, host)
			if err != nil || len(ips) == 0 {
				hlog.Debugf("=====foo11===%v", err)
				return nil, fmt.Errorf("httpx_err.1:%v", err)
			}

			d := &net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}

			var lastErr error
			for _, ip := range ips {
				conn, err := d.DialContext(ctx, "tcp4", net.JoinHostPort(ip.String(), port))
				if err == nil {
					hlog.Debugf("=====foo===%s[%v][%v]", net.JoinHostPort(ip.String(), port), conn.RemoteAddr(), conn.LocalAddr())
					return conn, nil
				}
				lastErr = err
			}
			hlog.Debugf("=====foo1===%v", lastErr)
			return nil, fmt.Errorf("httpx_err.2:%v", lastErr)
		},
	}
}
