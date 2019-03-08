package main

import (
	"context"
	"github.com/botless/commands/pkg/commands"
	"github.com/cloudevents/sdk-go/pkg/cloudevents/client"
	clienthttp "github.com/cloudevents/sdk-go/pkg/cloudevents/client/http"
	"github.com/cloudevents/sdk-go/pkg/cloudevents/transport/http"
	"github.com/kelseyhightower/envconfig"
	"log"
	"os"
)

type envConfig struct {
	// Port is server port to be listened.
	Port int `envconfig:"USER_PORT" default:"8080"`

	// Target is the endpoint to receive cloudevents.
	Target string `envconfig:"TARGET" required:"true"`
}

func main() {
	os.Exit(_main(os.Args[1:]))
}

func _main(args []string) int {
	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		log.Printf("[ERROR] Failed to process env var: %s", err)
		return 1
	}

	c, err := clienthttp.New(
		http.WithTarget(env.Target),
		http.WithPort(env.Port),
		http.WithBinaryEncoding(),
		client.WithTimeNow(),
		client.WithUUIDs(),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %s", err.Error())
	}

	cmds := &commands.Commands{Ce: c}

	ctx := context.Background()
	if err := c.StartReceiver(ctx, cmds.Receive); err != nil {
		log.Fatalf("Failed to start reveiver client: %s", err.Error())
	}
	log.Printf("default commands listening on :%d", env.Port)
	<-ctx.Done()
	log.Printf("default commands done")

	return 0
}
