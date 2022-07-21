package wrap

import (
	clickhousebuffer "github.com/zikwall/clickhouse-buffer/v3"
	"github.com/zikwall/clickhouse-buffer/v3/src/cx"
)

type BufferWrapper struct {
	conn cx.Clickhouse
}

func (w *BufferWrapper) Drop() error {
	return w.conn.Close()
}

func (w *BufferWrapper) DropMsg() string {
	return "close clickhouse buffer via wrapper"
}

func (w *BufferWrapper) Conn() cx.Clickhouse {
	return w.conn
}

func NewBufferWrapper(conn cx.Clickhouse) *BufferWrapper {
	return &BufferWrapper{
		conn: conn,
	}
}

type ClientWrapper struct {
	client clickhousebuffer.Client
}

func (c *ClientWrapper) Drop() error {
	c.client.Close()
	return nil
}

func (c *ClientWrapper) DropMsg() string {
	return "close clickhouse client via wrapper"
}

func (c *ClientWrapper) Client() clickhousebuffer.Client {
	return c.client
}

func NewClientWrapper(client clickhousebuffer.Client) *ClientWrapper {
	return &ClientWrapper{
		client: client,
	}
}
