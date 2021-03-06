package http

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/cloudevents/sdk-go/pkg/cloudevents"
	cecontext "github.com/cloudevents/sdk-go/pkg/cloudevents/context"
	"github.com/cloudevents/sdk-go/pkg/cloudevents/transport"
)

type EncodingSelector func(e cloudevents.Event) Encoding

// type check that this transport message impl matches the contract
var _ transport.Transport = (*Transport)(nil)

// Transport acts as both a http client and a http handler.
type Transport struct {
	Encoding                   Encoding
	DefaultEncodingSelectionFn EncodingSelector

	// Sending
	Client *http.Client
	Req    *http.Request

	// Receiving
	Receiver transport.Receiver
	Port     *int   // if nil, default 8080
	Path     string // if "", default "/"
	Handler  *http.ServeMux

	realPort          int
	server            *http.Server
	handlerRegistered bool
	codec             transport.Codec
}

// TransportContext allows a Receiver to understand the context of a request.
type TransportContext struct {
	URI    string
	Host   string
	Method string
}

func New(opts ...Option) (*Transport, error) {
	t := &Transport{
		Req: &http.Request{
			Method: http.MethodPost,
		},
	}
	if err := t.applyOptions(opts...); err != nil {
		return nil, err
	}
	return t, nil
}

func (t *Transport) applyOptions(opts ...Option) error {
	for _, fn := range opts {
		if err := fn(t); err != nil {
			return err
		}
	}
	return nil
}

func (t *Transport) loadCodec() bool {
	if t.codec == nil {
		if t.DefaultEncodingSelectionFn != nil && t.Encoding != Default {
			log.Printf("[warn] Transport has a DefaultEncodingSelectionFn set but Encoding is not Default. DefaultEncodingSelectionFn will be ignored.")
		}
		t.codec = &Codec{
			Encoding:                   t.Encoding,
			DefaultEncodingSelectionFn: t.DefaultEncodingSelectionFn,
		}
	}
	return true
}

func (t *Transport) Send(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, error) {
	if t.Client == nil {
		t.Client = &http.Client{}
	}

	var req http.Request
	if t.Req != nil {
		req.Method = t.Req.Method
		req.URL = t.Req.URL
	}

	// Override the default request with target from context.
	if target := cecontext.TargetFrom(ctx); target != nil {
		req.URL = target
	}

	if ok := t.loadCodec(); !ok {
		return nil, fmt.Errorf("unknown encoding set on transport: %d", t.Encoding)
	}

	msg, err := t.codec.Encode(event)
	if err != nil {
		return nil, err
	}

	// TODO: merge the incoming request with msg, for now just replace.
	if m, ok := msg.(*Message); ok {
		req.Header = m.Header
		req.Body = ioutil.NopCloser(bytes.NewBuffer(m.Body))
		req.ContentLength = int64(len(m.Body))
		return httpDo(ctx, &req, func(resp *http.Response, err error) (*cloudevents.Event, error) {
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()

			body, _ := ioutil.ReadAll(resp.Body)
			msg := &Message{
				Header: resp.Header,
				Body:   body,
			}

			var respEvent *cloudevents.Event
			if msg.CloudEventsVersion() != "" {
				if ok := t.loadCodec(); !ok {
					err := fmt.Errorf("unknown encoding set on transport: %d", t.Encoding)
					log.Printf("failed to load codec: %s", err)
				}
				if respEvent, err = t.codec.Decode(msg); err != nil {
					log.Printf("failed to decode message: %s %v", err, resp)
				}
			}

			if accepted(resp) {
				return respEvent, nil
			}
			return respEvent, fmt.Errorf("error sending cloudevent: %s", status(resp))
		})
	}
	return nil, fmt.Errorf("failed to encode Event into a Message")
}

func (t *Transport) SetReceiver(r transport.Receiver) {
	t.Receiver = r
}

func (t *Transport) StartReceiver(ctx context.Context) error {
	if t.server == nil {
		if t.Handler == nil {
			t.Handler = http.NewServeMux()
		}
		if !t.handlerRegistered {
			// handler.Handle might panic if the user tries to use the same path as the sdk.
			t.Handler.Handle(t.GetPath(), t)
			t.handlerRegistered = true
		}

		addr := fmt.Sprintf(":%d", t.GetPort())
		t.server = &http.Server{
			Addr:    addr,
			Handler: t.Handler,
		}

		listener, err := net.Listen("tcp", addr)
		if err != nil {
			return err
		}
		t.realPort = listener.Addr().(*net.TCPAddr).Port

		go func() {
			log.Print(t.server.Serve(listener))
			t.server = nil
			t.realPort = 0
		}()
		return nil
	}
	return fmt.Errorf("http server already started")
}

// This blocks until all active connections are closed.
func (t *Transport) StopReceiver(ctx context.Context) error {
	if t.server != nil {
		err := t.server.Close()
		t.server = nil
		if err != nil {
			return fmt.Errorf("failed to close http server, %s", err.Error())
		}
	}
	return nil
}

type eventError struct {
	event *cloudevents.Event
	err   error
}

func httpDo(ctx context.Context, req *http.Request, fn func(*http.Response, error) (*cloudevents.Event, error)) (*cloudevents.Event, error) {
	// Run the HTTP request in a goroutine and pass the response to fn.
	c := make(chan eventError, 1)
	req = req.WithContext(ctx)
	go func() {
		event, err := fn(http.DefaultClient.Do(req))
		c <- eventError{event: event, err: err}
	}()
	select {
	case <-ctx.Done():
		<-c // Wait for f to return.
		return nil, ctx.Err()
	case ee := <-c:
		return ee.event, ee.err
	}
}

// accepted is a helper method to understand if the response from the target
// accepted the CloudEvent.
func accepted(resp *http.Response) bool {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true
	}
	return false
}

// status is a helper method to read the response of the target.
func status(resp *http.Response) string {
	status := resp.Status
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("Status[%s] error reading response body: %v", status, err)
	}
	return fmt.Sprintf("Status[%s] %s", status, body)
}

func (t *Transport) invokeReceiver(ctx context.Context, event cloudevents.Event) (*Response, error) {
	if t.Receiver != nil {

		// Note: http does not use eventResp.Reason
		eventResp := cloudevents.EventResponse{}
		resp := Response{}

		err := t.Receiver.Receive(ctx, event, &eventResp)
		if err != nil {
			log.Printf("got an error from receiver fn: %s", err.Error())
			resp.StatusCode = http.StatusInternalServerError
			return &resp, err
		}

		if eventResp.Event != nil {
			if t.loadCodec() {
				if m, err := t.codec.Encode(*eventResp.Event); err != nil {
					log.Printf("failed to encode response from receiver fn: %s", err.Error())
				} else if msg, ok := m.(*Message); ok {
					resp.Header = msg.Header
					resp.Body = msg.Body
				}
			} else {
				log.Printf("failed to load codec")
				resp.StatusCode = http.StatusInternalServerError
				return &resp, err
			}
		}

		if eventResp.Status != 0 {
			resp.StatusCode = eventResp.Status
		} else {
			resp.StatusCode = http.StatusAccepted // default is 202 - Accepted
		}
		return &resp, err
	}
	return nil, nil
}

// ServeHTTP implements http.Handler
func (t *Transport) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Printf("failed to handle request: %s %v", err, req)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Invalid request"}`))
		return
	}
	msg := &Message{
		Header: req.Header,
		Body:   body,
	}

	if ok := t.loadCodec(); !ok {
		err := fmt.Errorf("unknown encoding set on transport: %d", t.Encoding)
		log.Printf("failed to load codec: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`{"error":%q}`, err.Error())))
		return
	}
	event, err := t.codec.Decode(msg)
	if err != nil {
		log.Printf("failed to decode message: %s %v", err, req)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`{"error":%q}`, err.Error())))
		return
	}

	ctx := req.Context()

	if req != nil {
		ctx = cecontext.WithTransportContext(ctx, TransportContext{
			URI:    req.RequestURI,
			Host:   req.Host,
			Method: req.Method,
		})
	}

	resp, err := t.invokeReceiver(ctx, *event)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`{"error":%q}`, err.Error())))
		return
	}
	if resp != nil {
		if len(resp.Header) > 0 {
			for k, vs := range resp.Header {
				for _, v := range vs {
					w.Header().Set(k, v)
				}
			}
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 600 {
			w.WriteHeader(resp.StatusCode)
		}
		if len(resp.Body) > 0 {
			w.Write(resp.Body)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (t *Transport) GetPort() int {
	if t.Port != nil && *t.Port == 0 && t.realPort != 0 {
		return t.realPort
	}

	if t.Port != nil && *t.Port >= 0 { // 0 means next open port
		return *t.Port
	}
	return 8080 // default
}

func (t *Transport) GetPath() string {
	path := strings.TrimSpace(t.Path)
	if len(path) > 0 {
		return path
	}
	return "/" // default
}
