package discovery

import (
	"context"
	"fmt"
	routev1 "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

const (
	openShiftRoutes = "routes"
)

type openShiftServiceCatalog struct {
	dc *OpenShiftDiscoveryClient
}

type OpenShiftDiscoveryClient struct {
	RouteClient routev1.RouteV1Interface
}

func newOpenShiftServiceCatalog(discoveryClient *OpenShiftDiscoveryClient) openShiftServiceCatalog {
	return openShiftServiceCatalog{
		dc: discoveryClient,
	}
}
func newOpenShiftServiceCatalogForConfig(cfg *rest.Config) openShiftServiceCatalog {
	return openShiftServiceCatalog{
		dc: newOpenShiftDiscoveryClientForConfig(cfg),
	}
}

func newOpenShiftDiscoveryClientForConfig(cfg *rest.Config) *OpenShiftDiscoveryClient {
	if client, err := routev1.NewForConfig(cfg); err != nil {
		//TODO revisar el tratamiento de errores
		return nil
	} else {
		return newOpenShiftDiscoveryClient(client)
	}
}

func newOpenShiftDiscoveryClient(routeClient routev1.RouteV1Interface) *OpenShiftDiscoveryClient {
	return &OpenShiftDiscoveryClient{
		RouteClient: routeClient,
	}
}

func (c openShiftServiceCatalog) Query(ctx context.Context, uri ResourceUri, outputFormat string) (string, error) {
	if c.dc == nil {
		return "", fmt.Errorf("OpenShiftDiscoveryClient was not provided, maybe current operator is not running in OpenShift")
	}
	switch uri.GVK.Kind {
	case openShiftRoutes:
		return c.resolveOpenShiftRouteQuery(ctx, uri)
	default:
		return "", fmt.Errorf("resolution of openshift kind: %s is not implemented", uri.GVK.Kind)
	}
}

func (c openShiftServiceCatalog) resolveOpenShiftRouteQuery(ctx context.Context, uri ResourceUri) (string, error) {
	if route, err := c.dc.RouteClient.Routes(uri.Namespace).Get(ctx, uri.Name, metav1.GetOptions{}); err != nil {
		return "", err
	} else {
		scheme := httpProtocol
		port := defaultHttpPort
		if route.Spec.TLS != nil {
			scheme = httpsProtocol
			port = defaultHttpsPort
		}
		return buildURI(scheme, route.Spec.Host, port), nil
	}
}
