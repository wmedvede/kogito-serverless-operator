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

package dev

import (
	"context"
	"fmt"
	"strconv"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/incubator-kie-kogito-serverless-operator/api"
	operatorapi "github.com/apache/incubator-kie-kogito-serverless-operator/api/v1alpha08"
	"github.com/apache/incubator-kie-kogito-serverless-operator/controllers/platform"
	"github.com/apache/incubator-kie-kogito-serverless-operator/controllers/profiles/common"
	"github.com/apache/incubator-kie-kogito-serverless-operator/controllers/profiles/common/constants"
	"github.com/apache/incubator-kie-kogito-serverless-operator/controllers/workflowdef"
	"github.com/apache/incubator-kie-kogito-serverless-operator/log"
	kubeutil "github.com/apache/incubator-kie-kogito-serverless-operator/utils/kubernetes"
)

const (
	configMapResourcesVolumeName               = "resources"
	configMapExternalResourcesVolumeNamePrefix = "res-"
	// quarkusDevConfigMountPath mount path for application properties file in the Workflow Quarkus Application
	// See: https://quarkus.io/guides/config-reference#application-properties-file
	quarkusDevConfigMountPath = "/home/kogito/serverless-workflow-project/src/main/resources"
)

type ensureRunningWorkflowState struct {
	*common.StateSupport
	ensurers *objectEnsurers
}

func (e *ensureRunningWorkflowState) CanReconcile(workflow *operatorapi.SonataFlow) bool {
	return workflow.Status.IsReady() || workflow.Status.GetTopLevelCondition().IsUnknown() || workflow.Status.IsChildObjectsProblem()
}

func (e *ensureRunningWorkflowState) Do(ctx context.Context, workflow *operatorapi.SonataFlow) (ctrl.Result, []client.Object, error) {
	var objs []client.Object

	fmt.Println("XXXX states_dev.go *ensureRunningWorkflowState.Do 1")

	fmt.Println("XXXX states_dev.go *ensureRunningWorkflowState.Do before ensurers.definitionConfigMap.Ensure")

	flowDefCM, _, err := e.ensurers.definitionConfigMap.Ensure(ctx, workflow, ensureWorkflowDefConfigMapMutator(workflow))
	if err != nil {
		fmt.Println("XXXX states_dev.go *ensureRunningWorkflowState.Do ensurers.definitionConfigMap.Ensure failed: " + err.Error())

		return ctrl.Result{Requeue: false}, objs, err
	}
	fmt.Println("XXXX states_dev.go *ensureRunningWorkflowState.Do after ensurers.definitionConfigMap.Ensure")

	objs = append(objs, flowDefCM)

	fmt.Println("XXXX states_dev.go *ensureRunningWorkflowState.Do 2")

	devBaseContainerImage := workflowdef.GetDefaultWorkflowDevModeImageTag()
	// check if the Platform available
	pl, err := platform.GetActivePlatform(ctx, e.C, workflow.Namespace)
	if err != nil {
		fmt.Println("XXXX states_dev.go *ensureRunningWorkflowState.Do NO active plaform, me la pela. err: " + err.Error())
	}
	if err == nil && len(pl.Spec.DevMode.BaseImage) > 0 {
		devBaseContainerImage = pl.Spec.DevMode.BaseImage
	}
	fmt.Println("XXXX states_dev.go *ensureRunningWorkflowState.Do 3")

	fmt.Println("XXXX states_dev.go *ensureRunningWorkflowState.Do before ensurers.userPropsConfigMap.Ensure")
	userPropsCM, _, err := e.ensurers.userPropsConfigMap.Ensure(ctx, workflow)
	if err != nil {
		fmt.Println("XXXX states_dev.go *ensureRunningWorkflowState.Do ensurers.userPropsConfigMap.Ensure failed: " + err.Error())
		return ctrl.Result{Requeue: false}, objs, err
	}

	managedPropsCM, _, err := e.ensurers.managedPropsConfigMap.Ensure(ctx, workflow, pl, common.ManagedPropertiesMutateVisitor(ctx, e.StateSupport.Catalog, workflow, pl, userPropsCM.(*corev1.ConfigMap)))
	if err != nil {
		fmt.Println("XXXX states_dev.go *ensureRunningWorkflowState.Do ensurers.managedPropsConfigMap.Ensure failed: " + err.Error())

		return ctrl.Result{Requeue: false}, objs, err
	}
	objs = append(objs, managedPropsCM)

	fmt.Println("XXXX states_dev.go *ensureRunningWorkflowState.Do 4")
	externalCM, err := workflowdef.FetchExternalResourcesConfigMapsRef(e.C, workflow)
	if err != nil {
		fmt.Println("XXXX states_dev.go *ensureRunningWorkflowState.Do workflowdef.FetchExternalResourcesConfigMapsRef failed: " + err.Error())

		workflow.Status.Manager().MarkFalse(api.RunningConditionType, api.ExternalResourcesNotFoundReason, "External Resources ConfigMap not found: %s", err.Error())
		if _, err = e.PerformStatusUpdate(ctx, workflow); err != nil {

			fmt.Println("XXXX states_dev.go *ensureRunningWorkflowState.Do  workflowdef.FetchExternalResourcesConfigMapsRef PerformStatusUpdate failed: " + err.Error())

			return ctrl.Result{RequeueAfter: constants.RequeueAfterFailure}, objs, err
		}
		fmt.Println("XXXX states_dev.go *ensureRunningWorkflowState.Do workflowdef.FetchExternalResourcesConfigMapsRef PerformStatusUpdate Ok!")

		return ctrl.Result{RequeueAfter: constants.RequeueAfterFailure}, objs, nil
	}

	fmt.Println("XXXX states_dev.go *ensureRunningWorkflowState.Do 5")

	fmt.Println("XXXX states_dev.go *ensureRunningWorkflowState.Do  before deployment ensurers")

	deployment, _, err := e.ensurers.deployment.Ensure(ctx, workflow,
		deploymentMutateVisitor(workflow),
		common.ImageDeploymentMutateVisitor(workflow, devBaseContainerImage),
		mountDevConfigMapsMutateVisitor(workflow, flowDefCM.(*corev1.ConfigMap), userPropsCM.(*corev1.ConfigMap), managedPropsCM.(*corev1.ConfigMap), externalCM))
	if err != nil {
		fmt.Println("XXXX states_dev.go *ensureRunningWorkflowState.Do deployment ensurers failed: " + err.Error())
		return ctrl.Result{RequeueAfter: constants.RequeueAfterFailure}, objs, err
	}
	objs = append(objs, deployment)

	fmt.Println("XXXX states_dev.go *ensureRunningWorkflowState.Do   after deployment ensurers")

	fmt.Println("XXXX states_dev.go *ensureRunningWorkflowState.Do   before service ensurers")

	service, _, err := e.ensurers.service.Ensure(ctx, workflow, common.ServiceMutateVisitor(workflow))
	if err != nil {
		fmt.Println("XXXX states_dev.go *ensureRunningWorkflowState.Do service ensurers failed: " + err.Error())
		return ctrl.Result{RequeueAfter: constants.RequeueAfterFailure}, objs, err
	}
	objs = append(objs, service)

	fmt.Println("XXXX states_dev.go *ensureRunningWorkflowState.Do   after service ensurers")

	fmt.Println("XXXX states_dev.go *ensureRunningWorkflowState.Do   before network ensurers")

	route, _, err := e.ensurers.network.Ensure(ctx, workflow)
	if err != nil {
		fmt.Println("XXXX states_dev.go *ensureRunningWorkflowState.Do   network ensurers failed: " + err.Error())
		return ctrl.Result{RequeueAfter: constants.RequeueAfterFailure}, objs, err
	}
	objs = append(objs, route)

	fmt.Println("XXXX states_dev.go *ensureRunningWorkflowState.Do   after network ensurers")

	// First time reconciling this object, mark as wait for deployment
	if workflow.Status.GetTopLevelCondition().IsUnknown() {

		fmt.Println("XXXX states_dev.go *ensureRunningWorkflowState.Do   first time reconcilling this object")

		klog.V(log.I).InfoS("Workflow is in WaitingForDeployment Condition")
		workflow.Status.Manager().MarkFalse(api.RunningConditionType, api.WaitingForDeploymentReason, "")
		if _, err = e.PerformStatusUpdate(ctx, workflow); err != nil {
			fmt.Println("XXXX states_dev.go *ensureRunningWorkflowState.Do   first time reconcilling this object, perform status update failed: " + err.Error())

			return ctrl.Result{RequeueAfter: constants.RequeueAfterFailure}, objs, err
		}
		fmt.Println("XXXX states_dev.go *ensureRunningWorkflowState.Do   first time reconcilling this object ok! and returning")

		return ctrl.Result{RequeueAfter: constants.RequeueAfterIsRunning}, objs, nil
	}

	// Is the deployment still available?

	fmt.Println("XXXX states_dev.go *ensureRunningWorkflowState.Do   Is deployment still available?")
	convertedDeployment := deployment.(*appsv1.Deployment)
	if !kubeutil.IsDeploymentAvailable(convertedDeployment) {
		fmt.Println("XXXX states_dev.go *ensureRunningWorkflowState.Do   Is deployment NOT available, attempt to recover")

		klog.V(log.I).InfoS("Workflow is not running due to a problem in the Deployment. Attempt to recover.")
		workflow.Status.Manager().MarkFalse(api.RunningConditionType,
			api.DeploymentUnavailableReason,
			common.GetDeploymentUnavailabilityMessage(convertedDeployment))
		if _, err = e.PerformStatusUpdate(ctx, workflow); err != nil {
			fmt.Println("XXXX states_dev.go *ensureRunningWorkflowState.Do   Is deployment NOT available, attempt to recover failed: " + err.Error())

			return ctrl.Result{RequeueAfter: constants.RequeueAfterFailure}, objs, err
		}
	}

	fmt.Println("XXXX states_dev.go *ensureRunningWorkflowState.Do   RequeueAfterIsRunning case!")
	return ctrl.Result{RequeueAfter: constants.RequeueAfterIsRunning}, objs, nil
}

func (e *ensureRunningWorkflowState) PostReconcile(ctx context.Context, workflow *operatorapi.SonataFlow) error {
	//By default, we don't want to perform anything after the reconciliation, and so we will simply return no error
	return nil
}

type followWorkflowDeploymentState struct {
	*common.StateSupport
	enrichers *statusEnrichers
}

func (f *followWorkflowDeploymentState) CanReconcile(workflow *operatorapi.SonataFlow) bool {
	return workflow.Status.IsWaitingForDeployment()
}

func (f *followWorkflowDeploymentState) Do(ctx context.Context, workflow *operatorapi.SonataFlow) (ctrl.Result, []client.Object, error) {

	fmt.Println("XXXX states_dev.go *followWorkflowDeploymentState.Do 1")

	fmt.Println("XXXX states_dev.go *followWorkflowDeploymentState.Do before SyncDeploymentStatus")
	result, err := common.DeploymentManager(f.C).SyncDeploymentStatus(ctx, workflow)
	if err != nil {
		fmt.Println("XXXX states_dev.go *followWorkflowDeploymentState.Do SyncDeploymentStatus failed: " + err.Error())
		return ctrl.Result{RequeueAfter: constants.RequeueAfterFailure}, nil, err
	}
	fmt.Println("XXXX states_dev.go *followWorkflowDeploymentState.Do after SyncDeploymentStatus")

	fmt.Println("XXXX states_dev.go *followWorkflowDeploymentState.Do before PerformStatusUpdate")
	if _, err := f.PerformStatusUpdate(ctx, workflow); err != nil {
		fmt.Println("XXXX states_dev.go *followWorkflowDeploymentState.Do PerformStatusUpdate failed: " + err.Error())

		return ctrl.Result{RequeueAfter: constants.RequeueAfterFailure}, nil, err
	}

	fmt.Println("XXXX states_dev.go *followWorkflowDeploymentState.Do after PerformStatusUpdate")

	return result, nil, nil
}

func (f *followWorkflowDeploymentState) PostReconcile(ctx context.Context, workflow *operatorapi.SonataFlow) error {

	fmt.Println("XXXX states_dev.go *followWorkflowDeploymentState.PostReconcile 1")

	deployment := &appsv1.Deployment{}
	if err := f.C.Get(ctx, client.ObjectKeyFromObject(workflow), deployment); err != nil {
		fmt.Println("XXXX states_dev.go *followWorkflowDeploymentState.PostReconcile.ObjectKeyFromObject failed: " + err.Error())
		return err
	}
	available := false
	if deployment != nil && kubeutil.IsDeploymentAvailable(deployment) {
		available = true
		// Enriching Workflow CR status with needed network info
		fmt.Println("XXXX states_dev.go *followWorkflowDeploymentState.PostReconcile.IsDeploymentAvailable true!")

		fmt.Println("XXXX states_dev.go *followWorkflowDeploymentState.PostReconcile before networkInfo.Enrich")
		if _, err := f.enrichers.networkInfo.Enrich(ctx, workflow); err != nil {
			fmt.Println("XXXX states_dev.go *followWorkflowDeploymentState.PostReconcile  networkInfo.Enrich failed: " + err.Error())
			return err
		}
		fmt.Println("XXXX states_dev.go *followWorkflowDeploymentState.PostReconcile after networkInfo.Enrich")

		fmt.Println("XXXX states_dev.go *followWorkflowDeploymentState.PostReconcile before PerformStatusUpdate")

		if _, err := f.PerformStatusUpdate(ctx, workflow); err != nil {
			fmt.Println("XXXX states_dev.go *followWorkflowDeploymentState.PostReconcile PerformStatusUpdate failed: " + err.Error())
			return err
		}
		fmt.Println("XXXX states_dev.go *followWorkflowDeploymentState.PostReconcile after PerformStatusUpdate")
	}

	fmt.Println("XXXX states_dev.go *followWorkflowDeploymentState.PostReconcile exit with no errors, deployment available: " + strconv.FormatBool(available))

	return nil
}

type recoverFromFailureState struct {
	*common.StateSupport
}

func (r *recoverFromFailureState) CanReconcile(workflow *operatorapi.SonataFlow) bool {
	return workflow.Status.GetCondition(api.RunningConditionType).IsFalse()
}

func (r *recoverFromFailureState) Do(ctx context.Context, workflow *operatorapi.SonataFlow) (ctrl.Result, []client.Object, error) {

	fmt.Println("XXXX states_dev.go *recoverFromFailureState.Do 1")

	// for now, a very basic attempt to recover by rolling out the deployment
	deployment := &appsv1.Deployment{}
	if err := r.C.Get(ctx, client.ObjectKeyFromObject(workflow), deployment); err != nil {
		// if the deployment is not there, let's try to reset the status condition and make the reconciliation fix the objects
		fmt.Println("XXXX states_dev.go *recoverFromFailureState read deployment failure: " + err.Error())

		if errors.IsNotFound(err) {
			fmt.Println("XXXX states_dev.go *recoverFromFailureState deployment was not found, try to update workflow status")

			klog.V(log.I).InfoS("Tried to recover from failed state, no deployment found, trying to reset the workflow conditions")
			workflow.Status.RecoverFailureAttempts = 0
			workflow.Status.Manager().MarkUnknown(api.RunningConditionType, "", "")
			if _, updateErr := r.PerformStatusUpdate(ctx, workflow); updateErr != nil {
				fmt.Println("XXXX states_dev.go *recoverFromFailureState deployment was not found, updated workflow status failed: " + updateErr.Error())
				return ctrl.Result{Requeue: false}, nil, updateErr
			}
			fmt.Println("XXXX states_dev.go *recoverFromFailureState deployment was not found, requeue after failure.")

			return ctrl.Result{RequeueAfter: constants.RequeueAfterFailure}, nil, nil
		}
		fmt.Println("XXXX states_dev.go *recoverFromFailureState deployment was not found, requeue with false")

		return ctrl.Result{Requeue: false}, nil, err
	}

	fmt.Println("XXXX states_dev.go *recoverFromFailureState.Do, before check if deployment is available")

	// if the deployment is progressing we might have good news
	if kubeutil.IsDeploymentAvailable(deployment) {
		fmt.Println("XXXX states_dev.go *recoverFromFailureState.Do, deployment is available!")

		fmt.Println("XXXX states_dev.go *recoverFromFailureState.Do, try to update workflow status to api.RunningConditionType = true")
		workflow.Status.RecoverFailureAttempts = 0
		workflow.Status.Manager().MarkTrue(api.RunningConditionType)
		if _, updateErr := r.PerformStatusUpdate(ctx, workflow); updateErr != nil {
			fmt.Println("XXXX states_dev.go *recoverFromFailureState.Do, try to update workflow status to api.RunningConditionType = true failed: " + updateErr.Error())
			return ctrl.Result{Requeue: false}, nil, updateErr
		}
		fmt.Println("XXXX states_dev.go *recoverFromFailureState.Do, try to update workflow status to api.RunningConditionType = true updated ok!, requeue")

		return ctrl.Result{RequeueAfter: constants.RequeueAfterFailure}, nil, nil
	}

	fmt.Println("XXXX states_dev.go *recoverFromFailureState.Do, before check  workflow.Status.RecoverFailureAttempts")

	if workflow.Status.RecoverFailureAttempts >= constants.RecoverDeploymentErrorRetries {
		workflow.Status.Manager().MarkFalse(api.RunningConditionType, api.RedeploymentExhaustedReason,
			"Can't recover workflow from failure after maximum attempts: %d", workflow.Status.RecoverFailureAttempts)
		if _, updateErr := r.PerformStatusUpdate(ctx, workflow); updateErr != nil {
			fmt.Println("XXXX states_dev.go *recoverFromFailureState.Do, check  workflow.Status.RecoverFailureAttempts failed: " + updateErr.Error())

			return ctrl.Result{}, nil, updateErr
		}
		return ctrl.Result{RequeueAfter: constants.RequeueRecoverDeploymentErrorInterval}, nil, nil
	}

	fmt.Println("XXXX states_dev.go *recoverFromFailureState.Do, after  workflow.Status.RecoverFailureAttempts")

	// TODO: we can improve deployment failures https://issues.redhat.com/browse/KOGITO-8812

	fmt.Println("XXXX states_dev.go *recoverFromFailureState.Do, Guard to avoid consecutive reconciliations to mess with the recover interval")

	// Guard to avoid consecutive reconciliations to mess with the recover interval
	if !workflow.Status.LastTimeRecoverAttempt.IsZero() &&
		metav1.Now().Sub(workflow.Status.LastTimeRecoverAttempt.Time).Minutes() > 10 {
		fmt.Println("XXXX states_dev.go *recoverFromFailureState.Do, Guard to avoid consecutive reconciliations to mess with the recover interval, requeue in 10 mins")
		return ctrl.Result{RequeueAfter: time.Minute * constants.RecoverDeploymentErrorInterval}, nil, nil
	}

	// let's try rolling out the deployment
	fmt.Println("XXXX states_dev.go *recoverFromFailureState.Do, before mark deployment to rollout")
	if err := kubeutil.MarkDeploymentToRollout(deployment); err != nil {
		fmt.Println("XXXX states_dev.go *recoverFromFailureState.Do, mark deployment to rollout failed: " + err.Error())
		return ctrl.Result{}, nil, err
	}
	fmt.Println("XXXX states_dev.go *recoverFromFailureState.Do, after mark deployment to rollout")

	fmt.Println("XXXX states_dev.go *recoverFromFailureState.Do, before retry.RetryOnConflict")
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		updateErr := r.C.Update(ctx, deployment)
		fmt.Println("XXXX states_dev.go *recoverFromFailureState.Do, retry.RetryOnConflict updateErr: " + updateErr.Error())
		return updateErr
	})

	if retryErr != nil {
		fmt.Println("XXXX states_dev.go *recoverFromFailureState.Do, before retry.RetryOnConflict failed, retryErr: " + retryErr.Error())

		klog.V(log.E).ErrorS(retryErr, "Error during Deployment rollout")
		return ctrl.Result{RequeueAfter: constants.RequeueRecoverDeploymentErrorInterval}, nil, nil
	}

	fmt.Println("XXXX states_dev.go *recoverFromFailureState.Do, before workflow.Status.RecoverFailureAttempts increase")
	workflow.Status.RecoverFailureAttempts += 1
	workflow.Status.LastTimeRecoverAttempt = metav1.Now()
	if _, err := r.PerformStatusUpdate(ctx, workflow); err != nil {
		fmt.Println("XXXX states_dev.go *recoverFromFailureState.Do, workflow.Status.RecoverFailureAttempts increase failed: " + err.Error())
		return ctrl.Result{Requeue: false}, nil, err
	}
	fmt.Println("XXXX states_dev.go *recoverFromFailureState.Do, exit with constants.RequeueRecoverDeploymentErrorInterval")
	return ctrl.Result{RequeueAfter: constants.RequeueRecoverDeploymentErrorInterval}, nil, nil
}

func (r *recoverFromFailureState) PostReconcile(ctx context.Context, workflow *operatorapi.SonataFlow) error {
	//By default, we don't want to perform anything after the reconciliation, and so we will simply return no error
	return nil
}
