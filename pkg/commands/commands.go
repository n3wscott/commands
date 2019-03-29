package commands

import (
	"fmt"
	"github.com/botless/events/pkg/events"
	"github.com/cloudevents/sdk-go/pkg/cloudevents"
	"github.com/cloudevents/sdk-go/pkg/cloudevents/client"
	"github.com/cloudevents/sdk-go/pkg/cloudevents/types"
	"log"
	"net/url"
	"strconv"
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
	case "botless.bot.command.square":
		re = c.Square(event)
	case "botless.bot.command.fib":
		re = c.Fibonacci(event)
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
		log.Printf("responding with \n%s", re)
		resp.RespondWith(200, re)
	}
}

func (c *Commands) Square(parent cloudevents.Event) *cloudevents.Event {
	if parent.Type() != "botless.bot.command.square" {
		return nil
	}
	cmd := &events.Command{}
	if err := parent.DataAs(cmd); err != nil {
		log.Printf("failed to get events.Command from %s", parent.Type())
		return nil
	}
	ec := parent.Context.AsV02()

	var result string
	if x, err := strconv.Atoi(cmd.Args); err != nil {
		result = "bad number"
	} else {
		result = fmt.Sprintf("%d", x*x)
	}

	return &cloudevents.Event{
		Context: cloudevents.EventContextV02{
			Type:       events.Bot.Type("response"),
			Source:     *types.ParseURLRef("//botless/command/square"),
			Extensions: ec.Extensions,
		}.AsV02(),
		Data: events.Message{
			Channel: cmd.Channel,
			Text:    result,
		},
	}
}

func (c *Commands) Fibonacci(parent cloudevents.Event) *cloudevents.Event {
	if parent.Type() != "botless.bot.command.fib" {
		return nil
	}
	cmd := &events.Command{}
	if err := parent.DataAs(cmd); err != nil {
		log.Printf("failed to get events.Command from %s", parent.Type())
		return nil
	}
	ec := parent.Context.AsV02()

	var result string
	var keepFib bool

	// f(0) = 0
	// f(1) = 1
	// f(n) = f(n-1)+f(n-2)

	n := strings.Split(cmd.Args, " ")
	if len(n) == 1 {
		if x, err := strconv.Atoi(cmd.Args); err != nil || x > 30 {
			result = "bad number, n < 30"
		} else if x >= 1 {
			// prime it
			result = fmt.Sprintf("0 1 %d", x-1)
			keepFib = true
		} else if x == 0 {
			result = "0"
		}
	} else if len(n) == 3 {
		var err error
		var n2 int
		var n1 int
		var x int

		n2, err = strconv.Atoi(n[0])
		n1, err = strconv.Atoi(n[1])
		x, err = strconv.Atoi(n[2])

		if err != nil {
			result = "bad number"
		} else if x == 0 {
			result = fmt.Sprintf("%d", n1+n2)
		} else {
			result = fmt.Sprintf("%d %d %d", n1, n1+n2, x-1)
			keepFib = true
		}
	} else {
		result = "just give me one number"
	}

	if keepFib {
		return &cloudevents.Event{
			Context: cloudevents.EventContextV02{
				Type:       events.Bot.Type("command", "fib"),
				Source:     *types.ParseURLRef("//botless/command/fib"),
				Extensions: ec.Extensions,
			}.AsV02(),
			Data: events.Command{
				Channel: cmd.Channel,
				Cmd:     cmd.Cmd,
				Author:  cmd.Author,
				Args:    result,
			},
		}
	}
	return &cloudevents.Event{
		Context: cloudevents.EventContextV02{
			Type:       events.Bot.Type("response"),
			Source:     *types.ParseURLRef("//botless/command/fib"),
			Extensions: ec.Extensions,
		}.AsV02(),
		Data: events.Message{
			Channel: cmd.Channel,
			Text:    result,
		},
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
