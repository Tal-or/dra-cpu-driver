/*
 * Copyright 2023 The Kubernetes Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package driver

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreclientset "k8s.io/client-go/kubernetes"
	"k8s.io/dynamic-resource-allocation/kubeletplugin"
	"k8s.io/klog/v2"
	drapbv1 "k8s.io/kubelet/pkg/apis/dra/v1beta1"

	"github.com/Tal-or/dra-cpu-driver/pkg/config"
	"github.com/Tal-or/dra-cpu-driver/pkg/state"
)

var _ drapbv1.DRAPluginServer = &Driver{}

type Driver struct {
	Client coreclientset.Interface
	Plugin kubeletplugin.DRAPlugin
	State  *state.DeviceState
}

func New(ctx context.Context, cfg *config.Config) (*Driver, error) {
	drv := &Driver{
		Client: cfg.Coreclient,
	}

	deviceState, err := state.NewDeviceState(cfg)
	if err != nil {
		return nil, err
	}
	drv.State = deviceState

	plugin, err := kubeletplugin.Start(
		ctx,
		[]any{drv},
		kubeletplugin.KubeClient(cfg.Coreclient),
		kubeletplugin.NodeName(cfg.ProgArgs.NodeName),
		kubeletplugin.DriverName(config.DriverName),
		kubeletplugin.RegistrarSocketPath(config.DriverPluginRegistrationPath),
		kubeletplugin.PluginSocketPath(config.DriverPluginSocketPath),
		kubeletplugin.KubeletPluginSocketPath(config.DriverPluginSocketPath))
	if err != nil {
		return nil, err
	}
	drv.Plugin = plugin

	var resources kubeletplugin.Resources
	for _, device := range deviceState.Allocatable {
		resources.Devices = append(resources.Devices, device)
	}
	klog.InfoS("publishing resources", "resources", resources)
	if err := plugin.PublishResources(ctx, resources); err != nil {
		return nil, err
	}

	return drv, nil
}

func (d *Driver) Shutdown(ctx context.Context) error {
	d.Plugin.Stop()
	return nil
}

func (d *Driver) NodePrepareResources(ctx context.Context, req *drapbv1.NodePrepareResourcesRequest) (*drapbv1.NodePrepareResourcesResponse, error) {
	klog.Infof("NodePrepareResource is called: number of claims: %d", len(req.Claims))
	preparedResources := &drapbv1.NodePrepareResourcesResponse{Claims: map[string]*drapbv1.NodePrepareResourceResponse{}}

	for _, claim := range req.Claims {
		preparedResources.Claims[claim.UID] = d.nodePrepareResource(ctx, claim)
	}

	return preparedResources, nil
}

func (d *Driver) nodePrepareResource(ctx context.Context, claim *drapbv1.Claim) *drapbv1.NodePrepareResourceResponse {
	resourceClaim, err := d.Client.ResourceV1beta1().ResourceClaims(claim.Namespace).Get(
		ctx,
		claim.Name,
		metav1.GetOptions{})
	if err != nil {
		return &drapbv1.NodePrepareResourceResponse{
			Error: fmt.Sprintf("failed to fetch ResourceClaim %s in namespace %s", claim.Name, claim.Namespace),
		}
	}

	prepared, err := d.State.Prepare(resourceClaim)
	if err != nil {
		return &drapbv1.NodePrepareResourceResponse{
			Error: fmt.Sprintf("error preparing devices for claim %v: %v", claim.UID, err),
		}
	}

	klog.Infof("Returning newly prepared devices for claim '%v': %v", claim.UID, prepared)
	return &drapbv1.NodePrepareResourceResponse{Devices: prepared}
}

func (d *Driver) NodeUnprepareResources(ctx context.Context, req *drapbv1.NodeUnprepareResourcesRequest) (*drapbv1.NodeUnprepareResourcesResponse, error) {
	klog.Infof("NodeUnPrepareResource is called: number of claims: %d", len(req.Claims))
	unpreparedResources := &drapbv1.NodeUnprepareResourcesResponse{Claims: map[string]*drapbv1.NodeUnprepareResourceResponse{}}

	for _, claim := range req.Claims {
		unpreparedResources.Claims[claim.UID] = d.nodeUnprepareResource(ctx, claim)
	}

	return unpreparedResources, nil
}

func (d *Driver) nodeUnprepareResource(ctx context.Context, claim *drapbv1.Claim) *drapbv1.NodeUnprepareResourceResponse {
	if err := d.State.Unprepare(claim.UID); err != nil {
		return &drapbv1.NodeUnprepareResourceResponse{
			Error: fmt.Sprintf("error unpreparing devices for claim %v: %v", claim.UID, err),
		}
	}

	return &drapbv1.NodeUnprepareResourceResponse{}
}
