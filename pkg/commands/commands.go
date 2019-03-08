package commands

import (
	"context"
	"fmt"
	"github.com/botless/events/pkg/events"
	"github.com/cloudevents/sdk-go/pkg/cloudevents"
	"github.com/cloudevents/sdk-go/pkg/cloudevents/client"
	"github.com/cloudevents/sdk-go/pkg/cloudevents/types"
	"log"
	"net/url"
	"strings"
)

type Commands struct {
	Ce client.Client
}

func (c *Commands) Receive(event cloudevents.Event) {
	// don't block the caller.
	go c.receive(event)
}

func (c *Commands) receive(event cloudevents.Event) {
	switch event.Type() {
	case "botless.bot.command":
		cmd := &events.Command{}
		if err := event.DataAs(cmd); err != nil {
			log.Printf("failed to get events.Command from %s", event.Type())
			return
		}
		c.command(event, *cmd)
	default:
		// ignore
		log.Printf("botless command ignored event type %q", event.Type())
	}
}

func (c *Commands) command(event cloudevents.Event, cmd events.Command) {
	switch cmd.Cmd {
	case "echo":
		c.echo(event, cmd)
	case "caps":
		c.caps(event, cmd)
	case "flip":
		c.flip(event, cmd)
	default:
		log.Printf("botless commnad ignored cmd %q", cmd.Cmd)
	}
}

func (c *Commands) echo(parent cloudevents.Event, cmd events.Command) {
	ec := parent.Context.AsV02()
	event := cloudevents.Event{
		Context: cloudevents.EventContextV02{
			Type:       events.Bot.Type("response"),
			Source:     *types.ParseURLRef("//botless/command/echo"),
			Extensions: ec.Extensions,
		}.AsV02(),
		Data: events.Message{
			Channel: cmd.Channel,
			Text:    cmd.Args,
		},
	}
	if _, err := c.Ce.Send(context.TODO(), event); err != nil {
		log.Printf("failed to send cloudevent: %s\n", err)
	} else {
		log.Printf("echo sent %s", cmd.Args)
	}
}

func (c *Commands) caps(parent cloudevents.Event, cmd events.Command) {
	ec := parent.Context.AsV02()
	event := cloudevents.Event{
		Context: cloudevents.EventContextV02{
			Type:       events.Bot.Type("response"),
			Source:     *types.ParseURLRef("//botless/command/caps"),
			Extensions: ec.Extensions,
		}.AsV02(),
		Data: events.Message{
			Channel: cmd.Channel,
			Text:    strings.ToUpper(cmd.Args),
		},
	}
	if _, err := c.Ce.Send(context.TODO(), event); err != nil {
		log.Printf("failed to send cloudevent: %s\n", err)
	} else {
		log.Printf("upper sent %s", cmd.Args)
	}
}

func (c *Commands) flip(parent cloudevents.Event, cmd events.Command) {
	ec := parent.Context.AsV02()
	event := cloudevents.Event{
		Context: cloudevents.EventContextV02{
			Type:       events.Bot.Type("response"),
			Source:     *types.ParseURLRef("//botless/command/flip"),
			Extensions: ec.Extensions,
		}.AsV02(),
		Data: events.Message{
			Channel: cmd.Channel,
			Text:    fmt.Sprintf("https://tableflip.dev/?flip=%s", url.QueryEscape(cmd.Args)),
		},
	}
	if _, err := c.Ce.Send(context.TODO(), event); err != nil {
		log.Printf("failed to send cloudevent: %s\n", err)
	} else {
		log.Printf("flip sent %s", cmd.Args)
	}
}
