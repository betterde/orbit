package checker

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"github.com/hashicorp/go-cleanhttp"
	"go.uber.org/zap"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	// MinInterval is the minimal interval between
	// two checks. Do not allow for an interval below this value.
	// Otherwise, we risk fork bombing a system.
	MinInterval = time.Second

	// DefaultBufSize is the maximum size of the captured
	// check output by default. Prevents an enormous buffer
	// from being captured
	DefaultBufSize = 4 * 1024 // 4KB

	// UserAgent is the value of the User-Agent header
	// for HTTP health checks.
	UserAgent = "Orbit Health Checker"
)

type CheckNotifier interface {
	UpdateCheck(status, output string)
}

type CheckHTTP struct {
	HTTP             string
	Header           map[string][]string
	Method           string
	Body             string
	Interval         time.Duration
	Timeout          time.Duration
	Logger           *zap.SugaredLogger
	TLSClientConfig  *tls.Config
	OutputMaxSize    int
	StatusHandler    *StatusHandler
	DisableRedirects bool

	httpClient *http.Client
	stop       bool
	stopCh     chan struct{}
	stopLock   sync.Mutex
	stopWg     sync.WaitGroup

	// Set if checks are exposed through Connect proxies
	// If set, this is the target of check()
	ProxyHTTP string
}

func (c *CheckHTTP) CheckType() CheckType {
	return CheckType{
		HTTP:          c.HTTP,
		Method:        c.Method,
		Body:          c.Body,
		Header:        c.Header,
		Interval:      c.Interval,
		ProxyHTTP:     c.ProxyHTTP,
		Timeout:       c.Timeout,
		OutputMaxSize: c.OutputMaxSize,
	}
}

// Start is used to start an HTTP check.
// The check runs until stop is called
func (c *CheckHTTP) Start() {
	c.stopLock.Lock()
	defer c.stopLock.Unlock()

	if c.httpClient == nil {
		// Create the transport. We disable HTTP Keep-Alive to prevent
		// failing checks due to the keepalive interval.
		trans := cleanhttp.DefaultTransport()
		trans.DisableKeepAlives = true

		// Take on the supplied TLS client config.
		trans.TLSClientConfig = c.TLSClientConfig

		// Create the HTTP client.
		c.httpClient = &http.Client{
			Timeout:   10 * time.Second,
			Transport: trans,
		}
		if c.DisableRedirects {
			c.httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			}
		}
		if c.Timeout > 0 {
			c.httpClient.Timeout = c.Timeout
		}

		if c.OutputMaxSize < 1 {
			c.OutputMaxSize = DefaultBufSize
		}
	}

	c.stop = false
	c.stopCh = make(chan struct{})
	c.stopWg.Add(1)
	go c.run()
}

// Stop is used to stop an HTTP check.
func (c *CheckHTTP) Stop() {
	c.stopLock.Lock()
	defer c.stopLock.Unlock()
	if !c.stop {
		c.stop = true
		close(c.stopCh)
	}

	// Wait for the c.run() goroutine to complete before returning.
	c.stopWg.Wait()
}

// run is invoked by a goroutine to run until Stop() is called
func (c *CheckHTTP) run() {
	defer c.stopWg.Done()
	// Get the randomized initial pause time
	initialPauseTime := RandomStagger(c.Interval)
	next := time.After(initialPauseTime)
	for {
		select {
		case <-next:
			c.check()
			next = time.After(c.Interval)
		case <-c.stopCh:
			return
		}
	}
}

// check is invoked periodically to perform the HTTP check
func (c *CheckHTTP) check() {
	method := c.Method
	if method == "" {
		method = "GET"
	}

	target := c.HTTP
	if c.ProxyHTTP != "" {
		target = c.ProxyHTTP
	}

	bodyReader := strings.NewReader(c.Body)
	req, err := http.NewRequest(method, target, bodyReader)
	if err != nil {
		c.StatusHandler.updateCheck(HealthCritical, err.Error())
		return
	}

	req.Header = http.Header(c.Header)

	// this happens during testing but not in prod
	if req.Header == nil {
		req.Header = make(http.Header)
	}

	if host := req.Header.Get("Host"); host != "" {
		req.Host = host
	}

	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", UserAgent)
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "text/plain, text/*, */*")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.StatusHandler.updateCheck(HealthCritical, err.Error())
		return
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			c.Logger.Errorw("An error occurred while closing I/O for reading the response content.", err)
		}
	}(resp.Body)

	// Read the response into a circular buffer to limit the size
	output, _ := NewBuffer(int64(c.OutputMaxSize))
	if _, err := io.Copy(output, resp.Body); err != nil {
		c.Logger.Warn("Check error while reading body", "error", err)
	}

	// Format the response body
	result := fmt.Sprintf("HTTP %s %s: %s Output: %s", method, target, resp.Status, output.String())

	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		// PASSING (2xx)
		c.StatusHandler.updateCheck(HealthPassing, result)
	} else if resp.StatusCode == 429 {
		// WARNING
		// 429 Too Many Requests (RFC 6585)
		// The user has sent too many requests in a given amount of time.
		c.StatusHandler.updateCheck(HealthWarning, result)
	} else {
		// CRITICAL
		c.StatusHandler.updateCheck(HealthCritical, result)
	}
}

// CheckTCP is used to periodically make a TCP connection to determine the
// health of a given check.
// The check is passing if the connection succeeds
// The check is critical if the connection returns an error
// Supports failures_before_critical and success_before_passing.
type CheckTCP struct {
	ServiceID       string
	TCP             string
	Interval        time.Duration
	Timeout         time.Duration
	Logger          *zap.SugaredLogger
	TLSClientConfig *tls.Config
	StatusHandler   *StatusHandler

	dialer   *net.Dialer
	stop     bool
	stopCh   chan struct{}
	stopLock sync.Mutex
}

// Start is used to start a TCP check.
// The check runs until stop is called
func (c *CheckTCP) Start() {
	c.stopLock.Lock()
	defer c.stopLock.Unlock()

	if c.dialer == nil {
		// Create the socket dialer
		c.dialer = &net.Dialer{
			Timeout: 10 * time.Second,
		}
		if c.Timeout > 0 {
			c.dialer.Timeout = c.Timeout
		}
	}

	c.stop = false
	c.stopCh = make(chan struct{})
	go c.run()
}

// Stop is used to stop a TCP check.
func (c *CheckTCP) Stop() {
	c.stopLock.Lock()
	defer c.stopLock.Unlock()
	if !c.stop {
		c.stop = true
		close(c.stopCh)
	}
}

// run is invoked by a goroutine to run until Stop() is called
func (c *CheckTCP) run() {
	// Get the randomized initial pause time
	initialPauseTime := RandomStagger(c.Interval)
	next := time.After(initialPauseTime)
	for {
		select {
		case <-next:
			c.check()
			next = time.After(c.Interval)
		case <-c.stopCh:
			return
		}
	}
}

// check is invoked periodically to perform the TCP check
func (c *CheckTCP) check() {
	var conn io.Closer
	var err error
	var checkType string

	if c.TLSClientConfig == nil {
		conn, err = c.dialer.Dial(`tcp`, c.TCP)
		checkType = "TCP"
	} else {
		conn, err = tls.DialWithDialer(c.dialer, `tcp`, c.TCP, c.TLSClientConfig)
		checkType = "TCP+TLS"
	}

	if err != nil {
		c.Logger.Warn(fmt.Sprintf("Check %s connection failed", checkType), "error", err)
		c.StatusHandler.updateCheck(HealthCritical, err.Error())
		return
	}

	err = conn.Close()
	if err != nil {
		c.Logger.Errorw("Error closing TCP connection.", err)
	}

	c.StatusHandler.updateCheck(HealthPassing, fmt.Sprintf("%s connect %s: Success", checkType, c.TCP))
}

// CheckUDP is used to periodically send a UDP datagram to determine the health of a given check.
// The check is passing if the connection succeeds, the response is bytes.Equal to the bytes passed
// in or if the error returned is a timeout error
// The check is critical if: the connection succeeds but the response is not equal to the bytes passed in,
// the connection succeeds but the error returned is not a timeout error or the connection fails
type CheckUDP struct {
	ServiceID     string
	UDP           string
	Message       string
	Interval      time.Duration
	Timeout       time.Duration
	Logger        *zap.SugaredLogger
	StatusHandler *StatusHandler

	dialer   *net.Dialer
	stop     bool
	stopCh   chan struct{}
	stopLock sync.Mutex
}

func (c *CheckUDP) Start() {
	c.stopLock.Lock()
	defer c.stopLock.Unlock()

	if c.dialer == nil {
		// Create the socket dialer
		c.dialer = &net.Dialer{
			Timeout: 10 * time.Second,
		}
		if c.Timeout > 0 {
			c.dialer.Timeout = c.Timeout
		}
	}

	c.stop = false
	c.stopCh = make(chan struct{})
	go c.run()
}

func (c *CheckUDP) Stop() {
	c.stopLock.Lock()
	defer c.stopLock.Unlock()
	if !c.stop {
		c.stop = true
		close(c.stopCh)
	}
}

func (c *CheckUDP) run() {
	// Get the randomized initial pause time
	initialPauseTime := RandomStagger(c.Interval)
	next := time.After(initialPauseTime)
	for {
		select {
		case <-next:
			c.check()
			next = time.After(c.Interval)
		case <-c.stopCh:
			return
		}
	}

}

func (c *CheckUDP) check() {

	conn, err := c.dialer.Dial(`udp`, c.UDP)

	if err != nil {
		if e, ok := err.(net.Error); ok && e.Timeout() {
			c.StatusHandler.updateCheck(HealthPassing, fmt.Sprintf("UDP connect %s: Success", c.UDP))
			return
		} else {
			c.Logger.Warn("Check socket connection failed", "error", err)
			c.StatusHandler.updateCheck(HealthCritical, err.Error())
			return
		}
	}
	defer conn.Close()

	n, err := fmt.Fprintf(conn, c.Message)
	if err != nil {
		c.Logger.Warn("Check socket write failed", "error", err)
		c.StatusHandler.updateCheck(HealthCritical, err.Error())
		return
	}

	if n != len(c.Message) {
		c.Logger.Warn("Check socket short write", "error", err)
		c.StatusHandler.updateCheck(HealthCritical, err.Error())
		return
	}

	if err != nil {
		c.Logger.Warn("Check socket write failed", "error", err)
		c.StatusHandler.updateCheck(HealthCritical, err.Error())
		return
	}
	_, err = bufio.NewReader(conn).Read(make([]byte, 1))
	if err != nil {
		if strings.Contains(err.Error(), "i/o timeout") {
			c.StatusHandler.updateCheck(HealthPassing, fmt.Sprintf("UDP connect %s: Success", c.UDP))
			return
		} else {
			c.Logger.Warn("Check socket read failed", "error", err)
			c.StatusHandler.updateCheck(HealthCritical, err.Error())
			return
		}
	} else if err == nil {
		c.StatusHandler.updateCheck(HealthPassing, fmt.Sprintf("UDP connect %s: Success", c.UDP))
	}
}

// RandomStagger returns an interval between 0 and the duration
func RandomStagger(interval time.Duration) time.Duration {
	if interval == 0 {
		return 0
	}

	return time.Duration(uint64(rand.Int63()) % uint64(interval))
}
