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

package knative

import (
	"k8s.io/client-go/rest"
	clienteventingv1 "knative.dev/eventing/pkg/client/clientset/versioned/typed/eventing/v1"
	clientservingv1 "knative.dev/serving/pkg/client/clientset/versioned/typed/serving/v1"
)

func NewKnativeServingClient(cfg *rest.Config) (*clientservingv1.ServingV1Client, error) {
	return clientservingv1.NewForConfig(cfg)
}

func NewKnativeEventingClient(cfg *rest.Config) (*clienteventingv1.EventingV1Client, error) {
	return clienteventingv1.NewForConfig(cfg)
}
