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

package utils

import (
	"context"

	clienteventingv1 "knative.dev/eventing/pkg/client/clientset/versioned/typed/eventing/v1"

	"github.com/RHsyseng/operator-utils/pkg/utils/openshift"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clientservingv1 "knative.dev/serving/pkg/client/clientset/versioned/typed/serving/v1"
)

const (
	knativeEventingNamespace = "knative-eventing"
	knativeServingNamespace  = "knative-serving"
)

var isOpenShift = false
var isKnativeEventing = false
var isKnativeServing = false

// IsOpenShift is a global flag that can be safely called across reconciliation cycles, defined at the controller manager start.
func IsOpenShift() bool {
	return isOpenShift
}

// SetIsOpenShift sets the global flag isOpenShift by the controller manager.
// We don't need to keep fetching the API every reconciliation cycle that we need to know about the platform.
func SetIsOpenShift(cfg *rest.Config) {
	var err error
	isOpenShift, err = openshift.IsOpenShift(cfg)
	if err != nil {
		panic("Impossible to verify if the cluster is OpenShift or not: " + err.Error())
	}
}

// IsKnativeServing is a global flag that can be safely called across reconciliation cycles, defined at the controller manager start.
func IsKnativeServing() bool {
	return isKnativeServing
}

// IsKnativeEventing is a global flag that can be safely called across reconciliation cycles, defined at the controller manager start.
func IsKnativeEventing() bool {
	return isKnativeEventing
}

// SetIsKnative sets the global flags isKnativeServing and isKnativeEventing by the controller manager.
func SetIsKnative(ctx context.Context, cli client.Client) {
	namespaceList := v1.NamespaceList{}
	if err := cli.List(ctx, &namespaceList); err != nil {
		panic("Impossible to verify if the knative system is installed in the cluster: " + err.Error())
	}
	for _, namespace := range namespaceList.Items {
		if namespace.Name == knativeServingNamespace {
			isKnativeServing = true
		}
		if namespace.Name == knativeEventingNamespace {
			isKnativeEventing = true
		}
	}
}

func NewKnativeServingClient(cfg *rest.Config) (*clientservingv1.ServingV1Client, error) {
	if cli, err := clientservingv1.NewForConfig(cfg); err != nil {
		return nil, err
	} else {
		return cli, nil
	}
}

func NewKnativeEventingClient(cfg *rest.Config) (*clienteventingv1.EventingV1Client, error) {
	if cli, err := clienteventingv1.NewForConfig(cfg); err != nil {
		return nil, err
	} else {
		return cli, nil
	}
}
