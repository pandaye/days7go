package geerpc

type Call struct {
	Seq    uint64
	Method string
	Argv   interface{}
	Replyv interface{}
	Error  error
	Done   chan *Call
}

func (c *Call) done() {
	c.Done <- c
}
