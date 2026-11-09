package httpx

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"strings"
	"time"

	eghelper "github.com/maliboot/mago/helper"

	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/app/client/retry"
	clientconf "github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/network/standard"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	config2 "github.com/hertz-contrib/http2/config"
	"github.com/hertz-contrib/http2/factory"
)

type Client struct {
	httpIns        *client.Client
	httpsIns       *client.Client
	https2Ins      *client.Client
	httpStreamIns  *client.Client
	httpsStreamIns *client.Client

	syncPool *eghelper.SyncBuf
}

func NewClient(buf *eghelper.SyncBuf) *Client {
	if buf == nil {
		buf = eghelper.NewSyncBuf()
	}
	ins := &Client{
		syncPool: buf,
	}
	ins.initialize()
	return ins
}

func (p *Client) initialize() {
	// 全局httpClient
	p.httpIns, _ = client.NewClient(
		client.WithDialTimeout(5*time.Second),
		client.WithMaxConnsPerHost(500),
		client.WithKeepAlive(true),
		client.WithMaxIdleConnDuration(time.Second*5),
		client.WithRetryConfig(
			retry.WithMaxAttemptTimes(2),
		),
		client.WithNoDefaultUserAgentHeader(true),
	)

	cipherSutieIds := make([]uint16, 0)
	for _, cipherSutie := range tls.CipherSuites() {
		cipherSutieIds = append(cipherSutieIds, cipherSutie.ID)
	}
	for _, insecureCipherSutie := range tls.InsecureCipherSuites() {
		cipherSutieIds = append(cipherSutieIds, insecureCipherSutie.ID)
	}

	p.httpsIns, _ = client.NewClient(
		client.WithTLSConfig(&tls.Config{
			InsecureSkipVerify: true,
			CipherSuites:       cipherSutieIds,
		}),
		client.WithDialer(standard.NewDialer()),
		client.WithDialTimeout(5*time.Second),
		client.WithMaxConnsPerHost(500),
		client.WithKeepAlive(true),
		client.WithMaxIdleConnDuration(time.Second*5),
		client.WithRetryConfig(
			retry.WithMaxAttemptTimes(2),
		),
		client.WithNoDefaultUserAgentHeader(true),
	)

	//f, _ := os.OpenFile("/Users/stone/wireshark_ssl.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	p.https2Ins, _ = client.NewClient()
	p.https2Ins.SetClientFactory(factory.NewClientFactory(
		config2.WithDialTimeout(5*time.Second),
		config2.WithReadIdleTimeout(1*time.Second),
		config2.WithWriteByteTimeout(time.Second),
		config2.WithPingTimeout(time.Minute),
		config2.WithMaxIdleConnDuration(5*time.Second),
		config2.WithClientDisableKeepAlive(true),     //Close Connection after each request
		config2.WithStrictMaxConcurrentStreams(true), // Set the server's SETTINGS_MAX_CONCURRENT_STREAMS to be respected globally.
		config2.WithDialer(standard.NewDialer()),     // You can customize dialer here
		config2.WithMaxHeaderListSize(0xffffffff),    // Set SETTINGS_MAX_HEADER_LIST_SIZE to unlimited.
		config2.WithMaxIdempotentCallAttempts(3),
		config2.WithRetryConfig(
			retry.WithMaxAttemptTimes(3),
			retry.WithInitDelay(2*time.Millisecond),
			retry.WithMaxDelay(200*time.Millisecond),
			retry.WithMaxJitter(30*time.Millisecond),
			retry.WithDelayPolicy(retry.FixedDelayPolicy),
		),
		config2.WithStrictMaxConcurrentStreams(true), // Set the server's SETTINGS_MAX_CONCURRENT_STREAMS to be respected globally.
		config2.WithTLSConfig(&tls.Config{
			InsecureSkipVerify: true,
			NextProtos:         []string{"h2"}, // 指明只协商 HTTP/2（ALPN）
			//KeyLogWriter:       f,              // 把 TLS secrets 写入文件，供 Wireshark 解密
			MinVersion:   tls.VersionTLS12,
			CipherSuites: cipherSutieIds,
		}),
	))

	// 全局httpStreamClient
	var clientOpts = []clientconf.ClientOption{
		client.WithMaxConnsPerHost(200),
		client.WithDialTimeout(5 * time.Second),
		client.WithResponseBodyStream(true),
		client.WithClientReadTimeout(time.Minute * 10),
		client.WithMaxConnWaitTimeout(10 * time.Second),
		client.WithRetryConfig(
			retry.WithMaxAttemptTimes(2),
		),
		client.WithNoDefaultUserAgentHeader(true),
	}
	p.httpStreamIns, _ = client.NewClient(clientOpts...)
	p.httpsStreamIns, _ = client.NewClient(append(clientOpts,
		client.WithTLSConfig(&tls.Config{InsecureSkipVerify: true, CipherSuites: cipherSutieIds}),
		client.WithDialer(standard.NewDialer()),
		client.WithRetryConfig(
			retry.WithMaxAttemptTimes(2),
		),
		client.WithNoDefaultUserAgentHeader(true),
	)...)
}

// Post
// 注意！protocol.ReleaseResponse(res) 外部控制
func (p *Client) Post(
	url string,
	headers map[string]string,
	body []byte,
	timeout time.Duration,
) (*protocol.Response, error) {
	return p.HttpRequest(url, consts.MethodPost, headers, body, timeout, 0, false)
}

// Get
// 注意！protocol.ReleaseResponse(res) 外部控制
func (p *Client) Get(
	url string,
	headers map[string]string,
	timeout time.Duration,
	maxRedirectsCount int,
) (*protocol.Response, error) {
	return p.HttpRequest(url, consts.MethodGet, headers, nil, timeout, maxRedirectsCount, false)
}

// Http2Post
// 注意！protocol.ReleaseResponse(res) 外部控制
func (p *Client) Http2Post(
	url string,
	headers map[string]string,
	body []byte,
	timeout time.Duration,
) (*protocol.Response, error) {
	return p.HttpRequest(url, consts.MethodPost, headers, body, timeout, 0, true)
}

// Http2Get
// 注意！protocol.ReleaseResponse(res) 外部控制
func (p *Client) Http2Get(
	url string,
	headers map[string]string,
	timeout time.Duration,
	maxRedirectsCount int,
) (*protocol.Response, error) {
	return p.HttpRequest(url, consts.MethodGet, headers, nil, timeout, maxRedirectsCount, true)
}

// HttpRequest
// 注意！protocol.ReleaseResponse(res) 外部控制
func (p *Client) HttpRequest(
	url string,
	method string,
	headers map[string]string,
	body []byte,
	timeout time.Duration,
	maxRedirectsCount int,
	isHttp2 bool,
) (*protocol.Response, error) {
	var httpClient *client.Client
	if strings.HasPrefix(url, "http://") {
		httpClient = p.httpIns
	} else {
		if isHttp2 {
			httpClient = p.https2Ins
		} else {
			httpClient = p.httpsIns
		}
	}

	if httpClient == nil {
		return nil, fmt.Errorf("http-io异常1: %s", url)
	}

	req := protocol.AcquireRequest()
	res := protocol.AcquireResponse()

	defer func() {
		protocol.ReleaseRequest(req)
		//protocol.ReleaseResponse(res) 外部控制
	}()

	req.Header.SetMethod(method)
	if len(headers) != 0 {
		req.SetHeaders(headers)
	}
	req.SetRequestURI(url)
	req.SetBody(body)

	if timeout != 0 {
		req.SetOptions(clientconf.WithRequestTimeout(timeout))
	}
	if timeout == 0 {
		timeout = time.Second * 30
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var err error
	if method == consts.MethodGet && maxRedirectsCount != 0 {
		err = httpClient.DoRedirects(ctx, req, res, maxRedirectsCount)
	} else {
		err = httpClient.Do(ctx, req, res)
	}
	if err != nil {
		return nil, fmt.Errorf("http-io[%s]异常2: %v", url, err)
	}

	hlog.Debugf("http-io完成: %s", url)
	return res, nil
}

func (p *Client) StreamGet(
	url string,
	headers map[string]string,
	timeout time.Duration,
	maxRedirectsCount int,
	dst io.Writer,
	streamBufferLimit uint,
) (*protocol.ResponseHeader, error) {
	return p.HttpStreamRequest(url, consts.MethodGet, headers, nil, timeout, maxRedirectsCount, dst, streamBufferLimit)
}

func (p *Client) StreamPost(
	url string,
	headers map[string]string,
	body []byte,
	timeout time.Duration,
	dst io.Writer,
	streamBufferLimit uint,
) (*protocol.ResponseHeader, error) {
	return p.HttpStreamRequest(url, consts.MethodPost, headers, body, timeout, 0, dst, streamBufferLimit)
}

func (p *Client) HttpStreamRequest(
	url string,
	method string,
	headers map[string]string,
	body []byte,
	timeout time.Duration,
	maxRedirectsCount int,
	dst io.Writer,
	streamBufferLimit uint,
) (*protocol.ResponseHeader, error) {
	var httpClient *client.Client
	if strings.Contains(url, "https://") {
		httpClient = p.httpsStreamIns
	} else {
		httpClient = p.httpStreamIns
	}
	if httpClient == nil {
		return nil, fmt.Errorf("http-io-stream异常.1: %s", url)
	}

	req := protocol.AcquireRequest()
	res := protocol.AcquireResponse()
	defer func() {
		protocol.ReleaseRequest(req)
		//protocol.ReleaseResponse(res) // 外部控制
	}()

	req.Header.SetMethod(method)
	if len(headers) != 0 {
		req.SetHeaders(headers)
	}
	req.SetRequestURI(url)
	req.SetBody(body)

	if timeout != 0 {
		req.SetOptions(clientconf.WithRequestTimeout(timeout))
	}
	if timeout == 0 {
		timeout = time.Second * 30
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var err error
	if method == consts.MethodGet && maxRedirectsCount != 0 {
		err = httpClient.DoRedirects(ctx, req, res, maxRedirectsCount)
	} else {
		err = httpClient.Do(ctx, req, res)
	}
	if err != nil {
		return nil, fmt.Errorf("httpStream-io[%s]异常.2: %v", url, err)
	}

	if dst != nil {
		if streamBufferLimit == 0 {
			streamBufferLimit = 1024 * 1024 * 1 // 默认1M
		}
		streamBuffer := p.syncPool.Get(int64(streamBufferLimit))
		defer p.syncPool.Put(streamBuffer)

		written, err := io.CopyBuffer(dst, res.BodyStream(), streamBuffer)
		if err != nil {
			return nil, fmt.Errorf("httpStream-io[%s]异常.3: %v", url, err)
		}
		hlog.Debugf("httpStream-io[%s]完成, 写入字节: %d(%.2f MB)", url, written, float64(written)/1024/1024)
	}

	return &res.Header, nil
}
