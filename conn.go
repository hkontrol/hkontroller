package hkontroller

import (
	"fmt"
	"net/http"
	"sync"

	"bufio"
	"bytes"
	"errors"
	"io"
	"net"
)

const eventHeader = "EVENT"

type conn struct {
	net.Conn

	// s and ss are used to encrypt data. s is used to temporarily store the session.
	// After the next read, ss becomes s and the session is encrypted from then on.
	// ------------------------------------------------------------------------------------
	// 2022-02-17 (mah) This workaround is needed because switching to encryption is done
	// after sending a response. But Write() on http.ResponseWriter is not immediate.
	// So therefore we wait until the next read.
	smu sync.Mutex
	ss  *session

	readBuf io.Reader

	inBackground bool

	emu      sync.Mutex
	onEvent  func(*http.Response) // EVENT callback, when characteristic value updated
	response chan *http.Response  // assume that one request wants one response
}

func newConn(c net.Conn) *conn {
	cc := &conn{
		Conn:     c,
		smu:      sync.Mutex{},
		emu:      sync.Mutex{},
		response: make(chan *http.Response),
	}

	return cc
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
		return c.Conn.Write(b)
	}

	var buf bytes.Buffer
	buf.Write(b)
	enc, err := c.ss.Encrypt(&buf)

	if err != nil {
		err = c.Conn.Close()
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
		return c.Conn.Read(b)
	}

	if c.readBuf == nil {
		r := bufio.NewReader(c.Conn)
		buf, err := c.ss.Decrypt(r)
		if err != nil {
			if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
				// Ignore timeout error #77
			} else if errors.Is(err, net.ErrClosed) {
				// Ignore close errors
			} else {
				c.Conn.Close()
			}
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

// to transform EVENT into valid HTTP response
type eventTransformer struct {
	rr io.Reader
	cc *conn

	skipped     bool // indicates that rr.ReadWithTransform("EVENT") was performed
	transformed bool // indicates that EVENT proto was replaced with HTTP
	readIndex   int
}

func newEventTransformer(cc *conn, r io.Reader) *eventTransformer {

	return &eventTransformer{
		rr: r,
		cc: cc,
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

	if !t.transformed {
		d := []byte("HTTP")
		n = copy(p, d[t.readIndex:])
		t.readIndex += n
		if t.readIndex >= len(d) {
			t.transformed = true
			return n, nil
		}
	}

	return t.rr.Read(p)
}

// RoundTrip implementation to be able to use with http.Client

func (c *conn) StartBackgroundRead() {
	go c.backgroundRead()
	c.inBackground = true
}

func (c *conn) backgroundRead() {

	rd := bufio.NewReader(c)

	for {
		b, err := rd.Peek(5) // len of EVENT string
		if err != nil {
			fmt.Println(err)
			if errors.Is(err, io.EOF) {
				return
			}
			continue
		}
		if string(b) == eventHeader {
			rt := newEventTransformer(c, rd)
			rb := bufio.NewReader(rt)

			res, err := http.ReadResponse(rb, nil)
			if err != nil {
				fmt.Println(err)
				continue
			}

			all, err := io.ReadAll(res.Body)
			if err != nil {
				fmt.Println(err)
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
				fmt.Println(err)
				continue
			}
			// ReadAll here because if response is chunked then on next loop there will be error
			all, err := io.ReadAll(res.Body)
			if err != nil {
				fmt.Println("err: ", err)
				return
			}

			// then assign new res.Body
			res.Body = io.NopCloser(bytes.NewReader(all))

			c.response <- res
		}
	}
}

func (c *conn) RoundTrip(req *http.Request) (*http.Response, error) {
	err := req.Write(c)
	if err != nil {
		return nil, err
	}
	if c.inBackground {
		res := <-c.response
		return res, nil
	}

	rd := bufio.NewReader(c)
	res, err := http.ReadResponse(rd, nil)
	if err != nil {
		return nil, err
	}

	return res, nil
}
