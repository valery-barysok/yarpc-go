// Code generated by thriftrw v1.7.0. DO NOT EDIT.
// @generated

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

package internal

import "go.uber.org/thriftrw/thriftreflect"

// ThriftModule represents the IDL file used to generate this package.
var ThriftModule = &thriftreflect.ThriftModule{
	Name:     "internal",
	Package:  "go.uber.org/yarpc/serialize/internal",
	FilePath: "internal.thrift",
	SHA1:     "4b38ca0173a5216c6fef21a4acfe27b0b3d02a92",
	Raw:      rawIDL,
}

const rawIDL = "struct RPC {\n\t1: required binary spanContext\n\n\t2: required string callerName\n\t3: required string serviceName\n\t4: required string encoding\n\t5: required string procedure\n\n\t6: optional map<string,string> headers\n\t7: optional string shardKey\n\t8: optional string routingKey\n\t9: optional string routingDelegate\n\t10: optional binary body\n  11: optional Features features\n}\n\nstruct Features {\n  1: optional bool supportsBothResponseAndError\n}\n"
