package router

func BasicLogger(c *Context) {
	c.Next()
}

func Recovery(c *Context) {
	defer func() {
		if err := recover(); err != nil {
			c.Send500()
		}
	}()
	c.Next()
}
