package openshift

import (
	appsv1 "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	routev1 "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	"k8s.io/client-go/rest"
)

var routeClient routev1.RouteV1Interface
var appsClient appsv1.AppsV1Interface

func GetRouteClient(cfg *rest.Config) (routev1.RouteV1Interface, error) {
	if routeClient == nil {
		if osRouteClient, err := NewOpenShiftRouteClient(cfg); err != nil {
			return nil, err
		} else {
			routeClient = osRouteClient
		}
	}
	return routeClient, nil
}

func GetAppsClient(cfg *rest.Config) (appsv1.AppsV1Interface, error) {
	if appsClient == nil {
		if osAppsClient, err := NewOpenShiftAppsClientClient(cfg); err != nil {
			return nil, err
		} else {
			appsClient = osAppsClient
		}
	}
	return appsClient, nil
}

func NewOpenShiftRouteClient(cfg *rest.Config) (*routev1.RouteV1Client, error) {
	return routev1.NewForConfig(cfg)
}

func NewOpenShiftAppsClientClient(cfg *rest.Config) (*appsv1.AppsV1Client, error) {
	return appsv1.NewForConfig(cfg)
}
