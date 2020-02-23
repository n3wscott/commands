package main

import (
	"context"
	"github.com/botless/commands/pkg/commands"
	cloudevents "github.com/cloudevents/sdk-go"
	"github.com/kelseyhightower/envconfig"
	"log"
	"os"
)

type envConfig struct {
	// Port is server port to be listened.
	Port int `envconfig:"USER_PORT" default:"8080"`

	// Target is the endpoint to receive cloudevents.
	Target string `envconfig:"TARGET" required:"true"`

	// StrictType is the type this function will only handle.
	StrictType string `envconfig:"STRICT_TYPE" default:""`
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

	c, err := cloudevents.NewDefaultClient()
	if err != nil {
		log.Fatalf("Failed to create client: %s", err.Error())
	}

	cmds := &commands.Commands{
		Ce:         c,
		StrictType: env.StrictType,
	}

	ctx := context.Background()
	if err := c.StartReceiver(ctx, cmds.Receive); err != nil {
		log.Fatalf("Failed to start reveiver client: %s", err.Error())
	}
	log.Printf("core commands listening on :%d", env.Port)
	<-ctx.Done()
	log.Printf("core commands done")

	return 0
}
