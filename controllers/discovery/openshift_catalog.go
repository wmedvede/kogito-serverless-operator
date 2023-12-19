package discovery

import (
	"context"
	"fmt"

	"github.com/apache/incubator-kie-kogito-serverless-operator/controllers/openshift"
	"github.com/apache/incubator-kie-kogito-serverless-operator/log"
	"github.com/apache/incubator-kie-kogito-serverless-operator/utils"
	appsv1 "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	routev1 "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

const (
	openShiftRoutes            = "routes"
	openShiftDeploymentConfigs = "deploymentconfigs"
)

type openShiftServiceCatalog struct {
	dc *OpenShiftDiscoveryClient
}

type OpenShiftDiscoveryClient struct {
	RouteClient routev1.RouteV1Interface
	AppsClient  appsv1.AppsV1Interface
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
	var routeClient routev1.RouteV1Interface
	var appsClient appsv1.AppsV1Interface
	var err error
	if utils.IsOpenShift() {
		if routeClient, err = openshift.GetRouteClient(cfg); err != nil {
			klog.V(log.E).ErrorS(err, "Unable to get the openshift route client")
			return nil
		}
		if appsClient, err = openshift.GetAppsClient(cfg); err != nil {
			klog.V(log.E).ErrorS(err, "Unable to get the openshift apps client")
			return nil
		}
		return newOpenShiftDiscoveryClient(routeClient, appsClient)
	}
	return nil
}

func newOpenShiftDiscoveryClient(routeClient routev1.RouteV1Interface, appsClient appsv1.AppsV1Interface) *OpenShiftDiscoveryClient {
	return &OpenShiftDiscoveryClient{
		RouteClient: routeClient,
		AppsClient:  appsClient,
	}
}

func (c openShiftServiceCatalog) Query(ctx context.Context, uri ResourceUri, outputFormat string) (string, error) {
	if c.dc == nil {
		return "", fmt.Errorf("OpenShiftDiscoveryClient was not provided, maybe current operator is not running in OpenShift")
	}
	switch uri.GVK.Kind {
	case openShiftRoutes:
		return c.resolveOpenShiftRouteQuery(ctx, uri)
	case openshiftDeploymentConfigs:
		return c.resolveOpenShiftDeploymentConfigQuery(ctx, uri)
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

func (c openShiftServiceCatalog) resolveOpenShiftDeploymentConfigQuery(ctx context.Context, uri ResourceUri) (string, error) {
	return "", nil
}
