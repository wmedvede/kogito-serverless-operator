package discovery

import (
	"context"
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"

	duckv1 "knative.dev/pkg/apis/duck/v1"

	eventingv1 "knative.dev/eventing/pkg/apis/eventing/v1"

	fakeeventingclient "knative.dev/eventing/pkg/client/injection/client/fake"

	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	fakeservingclient "knative.dev/serving/pkg/client/injection/client/fake"
)

func Test_QueryKnativeService(t *testing.T) {
	doTestQueryKnativeService(t, "", "http://hello-world-knative-quarkus-service.usecase1.10.102.164.0.sslip.io")
}

func Test_QueryKnativeServiceNotFound(t *testing.T) {
	_, client := fakeservingclient.With(context.TODO())
	ctg := NewServiceCatalog(nil, client.ServingV1(), nil)
	doTestQueryWithError(t, ctg, *NewResourceUriBuilder(KnativeScheme).
		Kind("services").
		Group("serving.knative.dev").
		Version("v1").
		Namespace(namespace1).
		Name(knServiceName1).Build(), "", fmt.Sprintf("services.serving.knative.dev %q not found", knServiceName1))
}

func doTestQueryKnativeService(t *testing.T, outputFormat string, expectedUri string) {
	service := &servingv1.Service{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace1,
			Name:      knServiceName1,
		},
		Spec: servingv1.ServiceSpec{},
		Status: servingv1.ServiceStatus{
			RouteStatusFields: servingv1.RouteStatusFields{
				URL: &apis.URL{
					Scheme: "http",
					Host:   "hello-world-knative-quarkus-service.usecase1.10.102.164.0.sslip.io",
				},
			},
		},
	}
	_, client := fakeservingclient.With(context.TODO(), service)
	ctg := NewServiceCatalog(nil, client.ServingV1(), nil)
	doTestQuery(t, ctg, *NewResourceUriBuilder(KnativeScheme).
		Kind("services").
		Group("serving.knative.dev").
		Version("v1").
		Namespace(namespace1).
		Name(knServiceName1).Build(), "", expectedUri)
}

func Test_QueryKnativeBroker(t *testing.T) {
	doTestQueryKnativeBroker(t, "http://broker-ingress.knative-eventing.svc.cluster.local/usecase1/hello-world-broker1")
}

func Test_QueryKnativeBrokerNotFound(t *testing.T) {
	_, client := fakeeventingclient.With(context.TODO())
	ctg := NewServiceCatalog(nil, nil, client.EventingV1())
	doTestQueryWithError(t, ctg, *NewResourceUriBuilder(KnativeScheme).
		Kind("brokers").
		Group("eventing.knative.dev").
		Version("v1").
		Namespace(namespace1).
		Name(knBrokerName1).Build(), "", fmt.Sprintf("brokers.eventing.knative.dev %q not found", knBrokerName1))
}

func doTestQueryKnativeBroker(t *testing.T, expectedUri string) {
	broker := &eventingv1.Broker{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace1,
			Name:      knBrokerName1,
		},
		Spec: eventingv1.BrokerSpec{},
		Status: eventingv1.BrokerStatus{
			Address: duckv1.Addressable{
				URL: &apis.URL{
					Scheme: "http",
					Host:   "broker-ingress.knative-eventing.svc.cluster.local",
					Path:   "/usecase1/hello-world-broker1",
				},
			},
		},
	}
	_, client := fakeeventingclient.With(context.TODO(), broker)
	ctg := NewServiceCatalog(nil, nil, client.EventingV1())
	doTestQuery(t, ctg, *NewResourceUriBuilder(KnativeScheme).
		Kind("brokers").
		Group("eventing.knative.dev").
		Version("v1").
		Namespace(namespace1).
		Name(knBrokerName1).Build(), "", expectedUri)
}
