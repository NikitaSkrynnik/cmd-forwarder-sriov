// Copyright (c) 2020 Doc.ai and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package resetmechanism provides wrapper chain element to reset underlying server on mechanism change
package resetmechanism

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chainbreak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

type resetMechanismServer struct {
	wrappedServer  networkservice.NetworkServiceServer
	connMechanisms map[string]*networkservice.Mechanism
}

// NewServer returns a new reset mechanism server chain element
func NewServer(wrappedServer networkservice.NetworkServiceServer) networkservice.NetworkServiceServer {
	return &resetMechanismServer{
		wrappedServer:  wrappedServer,
		connMechanisms: map[string]*networkservice.Mechanism{},
	}
}

func (s *resetMechanismServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	connID := request.GetConnection().GetId()

	if storedMech, ok := s.connMechanisms[connID]; ok {
		mech := request.GetConnection().GetMechanism()
		if mech.GetType() == storedMech.GetType() {
			// mechanism is the same, there is no need to request the wrapped server
			return next.Server(ctx).Request(ctx, request)
		}

		// requested mechanism has been changed, we need to reset the connection for the wrapped server
		conn := request.GetConnection().Clone()
		conn.Mechanism = storedMech
		if _, err := chainbreak.NewNetworkServiceServer(s.wrappedServer).Close(ctx, conn); err != nil {
			return nil, err
		}
	}

	conn, err := s.wrappedServer.Request(ctx, request)
	if mech := conn.GetMechanism(); err == nil && mech != nil {
		s.connMechanisms[connID] = mech.Clone()
	}
	return conn, err
}

func (s *resetMechanismServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	delete(s.connMechanisms, conn.GetId())

	return s.wrappedServer.Close(ctx, conn)
}
