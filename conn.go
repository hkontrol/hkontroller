package hkontroller

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/brutella/dnssd"
	"github.com/hkontrol/hkontroller/log"
	"io"
	"net"
	"net/http"
	"sort"
	"sync"
	"time"
)

// dialServiceInstance lookup dnssd service and make tcp connection
func dialServiceInstance(ctx context.Context, e *dnssd.BrowseEntry, dialTimeout time.Duration) (net.Conn, error) {

	IPs := e.IPs
	sort.Slice(IPs, func(i, j int) bool {
		return IPs[i].To4() != nil // ip4 first
	})

	var firstEstablishedConn net.Conn
	mu := sync.Mutex{}

	// common context
	cc, cancel := context.WithCancel(ctx)
	defer cancel()

	// probe every ip tcpAddr
	wg := sync.WaitGroup{}
	for _, ip := range IPs {
		var tcpAddr string
		if ip.To4() == nil {
			// ipv6 tcpAddr in square brackets
			// [fe80::...%wlp2s0]:51510
			tcpAddr = fmt.Sprintf("[%s%%%s]:%d", ip.String(), e.IfaceName, e.Port)
		} else {
			tcpAddr = fmt.Sprintf("%s:%d", ip.String(), e.Port)
		}

		log.Debug.Println("dialing: ", tcpAddr)

		wg.Add(1)
		go func(tcpAddr string) {
			defer wg.Done()
			// use dialer with parent context to be able to cancel connect
			d := net.Dialer{Timeout: dialTimeout}
			tcpConn, err := d.DialContext(cc, "tcp", tcpAddr)
			if err != nil {
				log.Debug.Println("dial err: ", err)
				return
			}
			log.Debug.Println("dial good")
			mu.Lock()
			defer mu.Unlock()
			if firstEstablishedConn == nil {
				firstEstablishedConn = tcpConn
				cancel() // cancel all other attempts
			}
		}(tcpAddr)
	}

	wg.Wait()

	if firstEstablishedConn == nil {
		return nil, errors.New("no connection available")
	} else {
		return firstEstablishedConn, nil
	}
}

const eventHeader = "EVENT"
const httpHeader = "HTTP"

// to transform EVENT into valid HTTP response
type eventTransformer struct {
	rr io.Reader

	skipped     bool // indicates that rr.ReadWithTransform("EVENT") was performed
	transformed bool // indicates that EVENT proto was replaced with HTTP
	readIndex   int
}

func newEventTransformer(cc *conn, r io.Reader) *eventTransformer {

	return &eventTransformer{
		rr:          r,
		readIndex:   0,
		skipped:     false,
		transformed: false,
	}
}

// Read reads data and in case of EVENT it replaces EVENT header
// so http.ReadResponse may be invoked
func (t *eventTransformer) Read(p []byte) (n int, err error) {

	if !t.skipped {
		toReplace := make([]byte, len(eventHeader))
		n := 0
		for n < len(eventHeader) {
			k, err := t.rr.Read(toReplace[n:])
			if err != nil {
				return 0, err
			}
			n += k
		}
		t.skipped = true
	}

	for !t.transformed {
		d := []byte(httpHeader)
		n = copy(p, d[t.readIndex:])
		t.readIndex += n
		if t.readIndex >= len(d) {
			t.transformed = true
		}
	}

	nn, err := t.rr.Read(p[t.readIndex:])
	return t.readIndex + nn, err
}

type conn struct {
	net.Conn

	closed bool

	// s and ss are used to encrypt data. s is used to temporarily store the session.
	// After the next read, ss becomes s and the session is encrypted from then on.
	// ------------------------------------------------------------------------------------
	// 2022-02-17 (mah) This workaround is needed because switching to encryption is done
	// after sending a response. But Write() on http.ResponseWriter is not immediate.
	// So therefore we wait until the next read.
	smu sync.Mutex
	ss  *session

	readBuf   io.Reader
	bufReader *bufio.Reader

	inBackground   bool
	backgroundStop chan interface{}

	onEvent func(*http.Response) // EVENT callback, when characteristic value updated

	// indicates if response channel is open. it may be closed if request context was done
	wantResponse bool
	response     chan *http.Response // assume that one request wants one response
	resError     chan error
}

func newConn(c net.Conn) *conn {
	cc := &conn{
		Conn:     c,
		smu:      sync.Mutex{},
		response: make(chan *http.Response),
		resError: make(chan error),
		closed:   false,
	}

	return cc
}

func (c *conn) close() {
	c.closed = true
	c.Conn.Close()
}

func (c *conn) SetEventCallback(cb func(*http.Response)) {
	c.onEvent = cb
}

func (c *conn) UpgradeEnc(s *session) {
	c.smu.Lock()
	c.ss = s
	c.smu.Unlock()
}

// Write writes bytes to the connection.
// The written bytes are encrypted when possible.
func (c *conn) Write(b []byte) (int, error) {
	if c.ss == nil {
		n, err := c.Conn.Write(b)
		if err != nil {
			c.close()
			return n, err
		}
		return n, err
	}

	var buf bytes.Buffer
	buf.Write(b)
	enc, err := c.ss.Encrypt(&buf)

	if err != nil {
		c.close()
		return 0, err
	}

	encB, err := io.ReadAll(enc)
	n, err := c.Conn.Write(encB)

	return n, err
}

const (
	packetSize = 0x400
)

// Read reads bytes from the connection.
// The read bytes are decrypted when possible.
func (c *conn) Read(b []byte) (int, error) {
	if c.ss == nil {
		n, err := c.Conn.Read(b)
		if err != nil {
			c.close()
		}
		return n, err
	}

	if c.bufReader == nil {
		c.bufReader = bufio.NewReader(c.Conn)
	}
	if c.readBuf == nil {
		buf, err := c.ss.Decrypt(c.bufReader)
		if err != nil {
			c.close()
			return 0, err
		}

		c.readBuf = buf
	}

	n, err := c.readBuf.Read(b)

	if n < len(b) || err == io.EOF {
		c.readBuf = nil
	}

	return n, err
}

func (c *conn) loop() {
	c.backgroundStop = make(chan interface{})
	defer func() {
		c.backgroundStop <- struct {
		}{}
		close(c.backgroundStop)
	}()
	c.inBackground = true
	defer func() {
		c.inBackground = false
	}()
	rd := bufio.NewReader(c)
	for !c.closed {
		b, err := rd.Peek(len(eventHeader)) // len of EVENT string
		if err != nil {
			log.Debug.Println("error reading from connection: ", err)
			return
		}
		//fmt.Println("---")
		//fmt.Println("rd.Peek: ", string(b))
		if string(b) == eventHeader {
			rt := newEventTransformer(c, rd)
			rb := bufio.NewReader(rt)

			res, err := http.ReadResponse(rb, nil)
			if err != nil {
				return
			}

			all, err := io.ReadAll(res.Body)
			res.Body.Close()
			if err != nil {
				continue
			}

			// then assign new res.Body
			res.Body = io.NopCloser(bytes.NewReader(all))

			if c.onEvent != nil {
				c.onEvent(res)
			}

			continue
		} else {
			res, err := http.ReadResponse(rd, nil)
			if err != nil {
				log.Debug.Println("response error: ", err)
				if c.wantResponse {
					c.resError <- err
				}
				continue
			}
			//dump, err := httputil.DumpResponse(res, false)
			//fmt.Println(string(dump), err)

			// ReadAll here because if response is chunked then on next loop there will be error
			//fmt.Println("reading HTTP response body")

			all, err := io.ReadAll(res.Body)
			//fmt.Println("reading HTTP response body done. err: ", err)
			res.Body.Close()
			if err != nil {
				log.Debug.Println("response body read error: ", err)
				if c.wantResponse {
					c.resError <- err
				}
				continue
			}

			// then assign new res.Body
			res.Body = io.NopCloser(bytes.NewReader(all))

			if c.wantResponse {
				c.response <- res
			}
		}
	}
}
