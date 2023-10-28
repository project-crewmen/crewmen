package task

import (
	"github.com/docker/docker/client"
)

func main() {
	c := Config{
		Name: "test-container-1",
		Image: "strm/helloworld-http",
	}

	dc, _ := client.NewClientWithOpts(client.FromEnv)
	d := Docker{
		Client: dc,
		Config: c,
	}

	d.Run()
}