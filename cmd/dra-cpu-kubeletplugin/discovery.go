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

package main

import (
	"fmt"

	resourceapi "k8s.io/api/resource/v1beta1"
	"k8s.io/utils/cpuset"
	"k8s.io/utils/ptr"
	"math/rand"

	"github.com/google/uuid"
)

func enumerateAllPossibleDevices(cpus map[string]*cpuset.CPUSet) (AllocatableDevices, error) {
	allDevices := make(AllocatableDevices)
	for class, list := range cpus {
		devices := enumerateDevicesForCPUClass(class, list)
		allDevices = MergeMaps(allDevices, devices)
	}
	return allDevices, nil
}

func enumerateDevicesForCPUClass(class string, set *cpuset.CPUSet) AllocatableDevices {
	devices := make(AllocatableDevices)
	uuids := generateUUIDs(class, set.Size())
	for i, cpuID := range set.List() {
		device := resourceapi.Device{
			Name: fmt.Sprintf("cpu-%d", cpuID),
			Basic: &resourceapi.BasicDevice{
				Attributes: map[resourceapi.QualifiedName]resourceapi.DeviceAttribute{
					"index": {
						IntValue: ptr.To(int64(cpuID)),
					},
					"uuid": {
						StringValue: ptr.To(uuids[i]),
					},
					"zone": {
						IntValue: ptr.To(int64(0)),
					},
				},
			},
		}
		fillInMissingAttributes(device.Basic, class)
		devices[device.Name] = device
	}
	return devices
}

func fillInMissingAttributes(basicDevice *resourceapi.BasicDevice, cpuClass string) {
	switch cpuClass {
	case "reserved":
		basicDevice.Attributes["reserved"] = resourceapi.DeviceAttribute{BoolValue: ptr.To(true)}
		basicDevice.Attributes["shared"] = resourceapi.DeviceAttribute{BoolValue: ptr.To(false)}
		basicDevice.Attributes["allocatable"] = resourceapi.DeviceAttribute{BoolValue: ptr.To(false)}
	case "allocatable":
		basicDevice.Attributes["reserved"] = resourceapi.DeviceAttribute{BoolValue: ptr.To(false)}
		basicDevice.Attributes["shared"] = resourceapi.DeviceAttribute{BoolValue: ptr.To(false)}
		basicDevice.Attributes["allocatable"] = resourceapi.DeviceAttribute{BoolValue: ptr.To(true)}
	case "shared":
		basicDevice.Attributes["reserved"] = resourceapi.DeviceAttribute{BoolValue: ptr.To(false)}
		basicDevice.Attributes["shared"] = resourceapi.DeviceAttribute{BoolValue: ptr.To(true)}
		basicDevice.Attributes["allocatable"] = resourceapi.DeviceAttribute{BoolValue: ptr.To(false)}
	}
	return
}

func generateUUIDs(seed string, count int) []string {
	rand := rand.New(rand.NewSource(hash(seed)))

	uuids := make([]string, count)
	for i := 0; i < count; i++ {
		charset := make([]byte, 16)
		rand.Read(charset)
		uuid, _ := uuid.FromBytes(charset)
		uuids[i] = "cpu-" + uuid.String()
	}

	return uuids
}

func hash(s string) int64 {
	h := int64(0)
	for _, c := range s {
		h = 31*h + int64(c)
	}
	return h
}

func MergeMaps[K comparable, V any](a, b map[K]V) map[K]V {
	merged := make(map[K]V)
	for k, v := range a {
		merged[k] = v
	}
	for k, v := range b {
		merged[k] = v
	}
	return merged
}
