package main

import (
	"context"
	"github.com/botless/commands/pkg/commands"
	"github.com/cloudevents/sdk-go/pkg/cloudevents/client"
	"github.com/kelseyhightower/envconfig"
	"log"
)

type envConfig struct {
	// Port is server port to be listened.
	Port int `envconfig:"USER_PORT" default:"8080"`

	// StrictType is the type this function will only handle.
	StrictType string `envconfig:"STRICT_TYPE" default:""`
}

func main() {
	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		log.Fatalf("[ERROR] Failed to process env var: %s", err)
	}

	c, err := client.NewDefault()
	if err != nil {
		log.Fatalf("Failed to create client: %s", err.Error())
	}

	cmds := &commands.Commands{
		Ce:         c,
		StrictType: env.StrictType,
	}

	ctx := context.Background()
	log.Fatal(c.StartReceiver(ctx, cmds.Receive))
}
