package redis

import (
	"strconv"
)

//* Client

// Client manages the access to a database.
type Client struct {
	config Config
	//	pool   *connPool
	co *conn
}

// NewClient creates a new Client.
func NewClient(config Config) (*Client, error) {
	c := new(Client)
	c.config = config
	if conn, err := newConn(&config); err != nil {
		return nil, err
	} else {
		c.co = conn
	}
	//c.pool = newConnPool(&c.config)
	return c, nil
}

func NewBroomClient(config Config) (*Client, error) {
	c := new(Client)
	c.config = config
	if conn, err := newBroomConn(&config); err != nil {
		return nil, err
	} else {
		c.co = conn
	}
	//c.pool = newConnPool(&c.config)
	return c, nil
}

// Close closes all connections of the client.
func (c *Client) Close() {
	c.co.close()
}

func (c *Client) call(cmd Cmd, args ...interface{}) *Reply {
	/*
		// Connection handling
		conn, err := c.pool.pull()
		if err != nil {
			return &Reply{Type: ReplyError, Err: err}
		}

		defer c.pool.push(conn)
	*/
	return c.co.call(Cmd(cmd), args...)
}

// Call calls the given Redis command.
func (c *Client) Call(cmd string, args ...interface{}) *Reply {
	return c.call(Cmd(cmd), args...)
}

func (c *Client) asyncCall(cmd Cmd, args ...interface{}) Future {
	f := newFuture()

	go func() {
		f <- c.call(cmd, args...)
	}()

	return f
}

// AsyncCall calls the given Redis command asynchronously.
func (c *Client) AsyncCall(cmd string, args ...interface{}) Future {
	return c.asyncCall(Cmd(cmd), args...)
}

// InfoMap calls the INFO command, parses and returns the results as a map[string]string or an error.
// Use Info method for fetching the unparsed INFO results.
func (c *Client) InfoMap() (map[string]string, error) {
	/*
		// Connection handling
		conn, err := c.pool.pull()
		if err != nil {
			return nil, err
		}

		defer c.pool.push(conn)
	*/
	return c.co.infoMap()

}

func (c *Client) multiCall(transaction bool, f func(*MultiCall)) *Reply {
	/*
		// Connection handling
		conn, err := c.pool.pull()

		if err != nil {
			return &Reply{Type: ReplyError, Err: err}
		}

		defer c.pool.push(conn)
	*/
	return newMultiCall(transaction, c.co).process(f)
}

// MultiCall executes the given MultiCall.
// Multicall reply is guaranteed to have the same number of sub-replies as calls, if it succeeds.
func (c *Client) MultiCall(f func(*MultiCall)) *Reply {
	return c.multiCall(false, f)
}

// Transaction performs a simple transaction.
// Simple transaction is a multi command that is wrapped in a MULTI-EXEC block.
// For complex transactions with WATCH, UNWATCH or DISCARD commands use MultiCall.
// Transaction reply is guaranteed to have the same number of sub-replies as calls, if it succeeds.
func (c *Client) Transaction(f func(*MultiCall)) *Reply {
	return c.multiCall(true, f)
}

// AsyncMultiCall calls an asynchronous MultiCall.
func (c *Client) AsyncMultiCall(mc func(*MultiCall)) Future {
	f := newFuture()

	go func() {
		f <- c.MultiCall(mc)
	}()

	return f
}

// AsyncTransaction performs a simple asynchronous transaction.
func (c *Client) AsyncTransaction(mc func(*MultiCall)) Future {
	f := newFuture()

	go func() {
		f <- c.Transaction(mc)
	}()

	return f
}

//* PubSub

// Subscription returns a new Subscription instance with the given message handler callback or
// an error. The message handler is called whenever a new message arrives.
// Subscriptions create their own dedicated connections,
// they do not pull connections from the connection pool.
func (c *Client) Subscription(msgHdlr func(msg *Message)) (*Subscription, *Error) {
	if msgHdlr == nil {
		panic(errmsg("message handler must not be nil"))
	}

	sub, err := newSubscription(&c.config, msgHdlr)
	if err != nil {
		return nil, err
	}

	return sub, nil
}

func (c *Client) Bytes() []byte {
	return c.co.bytes()
}

var rpush = []byte("*3\r\n$5\r\nRPUSH\r\n$3\r\ndbq\r\n$")

func (c *Client) DirectRpush(info []byte) {
	msg := append(rpush, []byte(strconv.Itoa(len(info)))...)
	msg = append(msg, delim...)
	msg = append(msg, info...)
	msg = append(msg, delim...)

	reply := c.co.DirectCall(msg)
	if reply.Err != nil {
		panic(reply.Err.Error())
	}
}
