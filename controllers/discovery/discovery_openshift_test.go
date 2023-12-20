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
	v1 "github.com/openshift/api/route/v1"
	"github.com/openshift/client-go/route/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"testing"
)

func Test_QueryOpenShiftRoute(t *testing.T) {
	doTestQueryOpenShiftRoute(t, false, "http://openshiftroutehost1:80")
}

func Test_QueryOpenShiftRouteWithTLS(t *testing.T) {
	doTestQueryOpenShiftRoute(t, true, "https://openshiftroutehost1:443")
}

func doTestQueryOpenShiftRoute(t *testing.T, tls bool, expectedUri string) {
	route := &v1.Route{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace1,
			Name:      openShiftRouteName1,
		},
		Spec: v1.RouteSpec{
			Host: openShiftRouteHost1,
		},
		Status: v1.RouteStatus{},
	}
	if tls {
		route.Spec.TLS = &v1.TLSConfig{}
	}
	fakeRoutesClient := fake.NewSimpleClientset(route)
	ctg := NewServiceCatalog(nil, nil, newOpenShiftDiscoveryClient(nil, fakeRoutesClient.RouteV1(), nil))
	doTestQuery(t, ctg, *NewResourceUriBuilder(OpenshiftScheme).
		Kind("routes").
		Group("route.openshift.io").
		Version("v1").
		Namespace(namespace1).
		Name(openShiftRouteName1).Build(), "", expectedUri)
}
