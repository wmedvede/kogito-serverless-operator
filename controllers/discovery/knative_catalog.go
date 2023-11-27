/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package discovery

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clienteventingv1 "knative.dev/eventing/pkg/client/clientset/versioned/typed/eventing/v1"
	clientservingv1 "knative.dev/serving/pkg/client/clientset/versioned/typed/serving/v1"
)

const (
	knServiceKind = "services"
	knBrokerKind  = "brokers"
)

type knServiceCatalog struct {
	ServingClient  clientservingv1.ServingV1Interface
	EventingClient clienteventingv1.EventingV1Interface
}

func newKnServiceCatalog(servingClient clientservingv1.ServingV1Interface, eventingClient clienteventingv1.EventingV1Interface) knServiceCatalog {
	return knServiceCatalog{
		ServingClient:  servingClient,
		EventingClient: eventingClient,
	}
}

func (c knServiceCatalog) Query(ctx context.Context, uri ResourceUri, outputFormat string) (string, error) {
	switch uri.GVK.Kind {
	case knServiceKind:
		return c.resolveKnServiceQuery(ctx, uri)
	case knBrokerKind:
		return c.resolveKnBrokerQuery(ctx, uri)
	default:
		return "", fmt.Errorf("resolution of knative kind: %s is not implemented", uri.GVK.Kind)
	}
}

func (c knServiceCatalog) resolveKnServiceQuery(ctx context.Context, uri ResourceUri) (string, error) {
	if service, err := c.ServingClient.Services(uri.Namespace).Get(ctx, uri.Name, metav1.GetOptions{}); err != nil {
		return "", err
	} else {
		// knative objects discovery should rely on the addressable interface
		return service.Status.Address.URL.String(), nil
	}
}

func (c knServiceCatalog) resolveKnBrokerQuery(ctx context.Context, uri ResourceUri) (string, error) {
	if broker, err := c.EventingClient.Brokers(uri.Namespace).Get(ctx, uri.Name, metav1.GetOptions{}); err != nil {
		return "", err
	} else {
		// knative objects discovery should rely on the addressable interface
		return broker.Status.Address.URL.String(), nil
	}
}
