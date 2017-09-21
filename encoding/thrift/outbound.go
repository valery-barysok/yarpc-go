// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package thrift

import (
	"bytes"
	"context"
	"fmt"

	"go.uber.org/thriftrw/envelope"
	"go.uber.org/thriftrw/protocol"
	"go.uber.org/thriftrw/wire"
	"go.uber.org/yarpc"
	encodingapi "go.uber.org/yarpc/api/encoding"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/thrift/internal"
	"go.uber.org/yarpc/pkg/encoding"
	"go.uber.org/yarpc/pkg/errors"
	"go.uber.org/yarpc/pkg/procedure"
)

// Client is a generic Thrift client. It speaks in raw Thrift payloads.
//
// Users should use the client generated by the code generator rather than
// using this directly.
type Client interface {
	// Call the given Thrift method.
	Call(ctx context.Context, reqBody envelope.Enveloper, opts ...yarpc.CallOption) (wire.Value, error)
	CallOneway(ctx context.Context, reqBody envelope.Enveloper, opts ...yarpc.CallOption) (transport.Ack, error)
}

// Config contains the configuration for the Client.
type Config struct {
	// Name of the Thrift service. This is the name used in the Thrift file
	// with the 'service' keyword.
	Service string

	// ClientConfig through which requests will be sent. Required.
	ClientConfig transport.ClientConfig
}

// New creates a new Thrift client.
func New(c Config, opts ...ClientOption) Client {
	// Code generated for Thrift client instantiation will probably be something
	// like this:
	//
	// 	func New(cc transport.ClientConfig, opts ...ClientOption) *MyServiceClient {
	// 		c := thrift.New(thrift.Config{
	// 			Service: "MyService",
	// 			ClientConfig: cc,
	// 			Protocol: protocol.Binary,
	// 		}, opts...)
	// 		return &MyServiceClient{client: c}
	// 	}
	//
	// So Config is really the internal config as far as consumers of the
	// generated client are concerned.

	var cc clientConfig
	for _, opt := range opts {
		opt.applyClientOption(&cc)
	}

	p := protocol.Binary
	if cc.Protocol != nil {
		p = cc.Protocol
	}

	if cc.Multiplexed {
		p = multiplexedOutboundProtocol{
			Protocol: p,
			Service:  c.Service,
		}
	}

	return thriftClient{
		p:             p,
		cc:            c.ClientConfig,
		thriftService: c.Service,
		Enveloping:    cc.Enveloping,
	}
}

type thriftClient struct {
	cc transport.ClientConfig
	p  protocol.Protocol

	// name of the Thrift service
	thriftService string
	Enveloping    bool
}

func (c thriftClient) Call(ctx context.Context, reqBody envelope.Enveloper, opts ...yarpc.CallOption) (wire.Value, error) {
	// Code generated for Thrift client calls will probably be something like
	// this:
	//
	// 	func (c *MyServiceClient) someMethod(ctx context.Context, arg1 Arg1Type, arg2 arg2Type, opts ...yarpc.CallOption) (returnValue, error) {
	// 		args := myservice.SomeMethodHelper.Args(arg1, arg2)
	// 		resBody, err := c.client.Call(ctx, args, opts...)
	// 		var result myservice.SomeMethodResult
	// 		if err = result.FromWire(resBody); err != nil {
	// 			return nil, err
	// 		}
	// 		success, err := myservice.SomeMethodHelper.UnwrapResponse(&result)
	// 		return success, err
	// 	}

	out := c.cc.GetUnaryOutbound()

	treq, proto, err := c.buildTransportRequest(reqBody)
	if err != nil {
		return wire.Value{}, err
	}

	call := encodingapi.NewOutboundCall(encoding.FromOptions(opts)...)
	ctx, err = call.WriteToRequest(ctx, treq)
	if err != nil {
		return wire.Value{}, err
	}

	tres, err := out.Call(ctx, treq)
	if err != nil && (tres == nil || !tres.ApplicationError) {
		return wire.Value{}, err
	}
	defer tres.Body.Close()

	if _, err = call.ReadFromResponse(ctx, tres); err != nil {
		return wire.Value{}, err
	}

	buf := bytes.NewBuffer(make([]byte, 0, _defaultBufferSize))
	if _, err = buf.ReadFrom(tres.Body); err != nil {
		return wire.Value{}, err
	}

	envelope, err := proto.DecodeEnveloped(bytes.NewReader(buf.Bytes()))
	if err != nil {
		return wire.Value{}, errors.ResponseBodyDecodeError(treq, err)
	}

	switch envelope.Type {
	case wire.Reply:
		return envelope.Value, nil
	case wire.Exception:
		var exc internal.TApplicationException
		if err := exc.FromWire(envelope.Value); err != nil {
			return wire.Value{}, errors.ResponseBodyDecodeError(treq, err)
		}
		return wire.Value{}, thriftException{
			Service:   treq.Service,
			Procedure: treq.Procedure,
			Reason:    &exc,
		}
	default:
		return wire.Value{}, errors.ResponseBodyDecodeError(
			treq, errUnexpectedEnvelopeType(envelope.Type))
	}
}

func (c thriftClient) CallOneway(ctx context.Context, reqBody envelope.Enveloper, opts ...yarpc.CallOption) (transport.Ack, error) {
	out := c.cc.GetOnewayOutbound()

	treq, _, err := c.buildTransportRequest(reqBody)
	if err != nil {
		return nil, err
	}

	call := encodingapi.NewOutboundCall(encoding.FromOptions(opts)...)
	ctx, err = call.WriteToRequest(ctx, treq)
	if err != nil {
		return nil, err
	}

	return out.CallOneway(ctx, treq)
}

func (c thriftClient) buildTransportRequest(reqBody envelope.Enveloper) (*transport.Request, protocol.Protocol, error) {
	proto := c.p
	if !c.Enveloping {
		proto = disableEnvelopingProtocol{
			Protocol: proto,
			Type:     wire.Reply, // we only decode replies with this instance
		}
	}

	treq := transport.Request{
		Caller:    c.cc.Caller(),
		Service:   c.cc.Service(),
		Encoding:  Encoding,
		Procedure: procedure.ToName(c.thriftService, reqBody.MethodName()),
	}

	value, err := reqBody.ToWire()
	if err != nil {
		// ToWire validates the request. If it failed, we should return the error
		// as-is because it's not an encoding error.
		return nil, nil, err
	}

	reqEnvelopeType := reqBody.EnvelopeType()
	if reqEnvelopeType != wire.Call && reqEnvelopeType != wire.OneWay {
		return nil, nil, errors.RequestBodyEncodeError(
			&treq, errUnexpectedEnvelopeType(reqEnvelopeType),
		)
	}

	var buffer bytes.Buffer
	err = proto.EncodeEnveloped(wire.Envelope{
		Name:  reqBody.MethodName(),
		Type:  reqEnvelopeType,
		SeqID: 1, // don't care
		Value: value,
	}, &buffer)
	if err != nil {
		return nil, nil, errors.RequestBodyEncodeError(&treq, err)
	}

	treq.Body = &buffer
	return &treq, proto, nil
}

type thriftException struct {
	Service   string
	Procedure string
	Reason    *internal.TApplicationException
}

func (e thriftException) Error() string {
	return fmt.Sprintf(
		"thrift request to procedure %q of service %q encountered an internal failure: %v",
		e.Procedure, e.Service, e.Reason)
}
