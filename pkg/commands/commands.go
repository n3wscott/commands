package commands

import (
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
	Ce         client.Client
	StrictType string
}

func (c *Commands) Receive(event cloudevents.Event, resp *cloudevents.EventResponse) {
	if c.StrictType != "" && event.Type() != c.StrictType {
		return
	}
	var re *cloudevents.Event
	switch event.Type() {
	case "botless.bot.command.echo":
		re = c.Echo(event)
	case "botless.bot.command.caps":
		re = c.Caps(event)
	case "botless.bot.command.flip":
		re = c.Flip(event)
	default:
		// ignore
		log.Printf("botless command ignored event type %q", event.Type())
	}
	if re != nil {
		resp.RespondWith(200, re)
	}
}

func (c *Commands) Echo(parent cloudevents.Event) *cloudevents.Event {
	if parent.Type() != "botless.bot.command.echo" {
		return nil
	}
	cmd := &events.Command{}
	if err := parent.DataAs(cmd); err != nil {
		log.Printf("failed to get events.Command from %s", parent.Type())
		return nil
	}
	ec := parent.Context.AsV02()
	return &cloudevents.Event{
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
}

func (c *Commands) Caps(parent cloudevents.Event) *cloudevents.Event {
	if parent.Type() != "botless.bot.command.caps" {
		return nil
	}
	cmd := &events.Command{}
	if err := parent.DataAs(cmd); err != nil {
		log.Printf("failed to get events.Command from %s", parent.Type())
		return nil
	}
	ec := parent.Context.AsV02()
	return &cloudevents.Event{
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
}

func (c *Commands) Flip(parent cloudevents.Event) *cloudevents.Event {
	if parent.Type() != "botless.bot.command.flip" {
		return nil
	}
	cmd := &events.Command{}
	if err := parent.DataAs(cmd); err != nil {
		log.Printf("failed to get events.Command from %s", parent.Type())
		return nil
	}
	ec := parent.Context.AsV02()
	return &cloudevents.Event{
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
}
