/*
Copyright 2017, 2019 the Velero contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	corev1api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	envAZOverride = "VELERO_AWS_AZ_OVERRIDE"
	labelZoneBeta = "failure-domain.beta.kubernetes.io/zone"
	labelZone     = "topology.kubernetes.io/zone"
)

// RestorePlugin is a restore item action plugin for Velero
type RestorePlugin struct {
	log logrus.FieldLogger
}

func newRestorePlugin(logger logrus.FieldLogger) *RestorePlugin {
	return &RestorePlugin{log: logger}
}

// AppliesTo returns information about which resources this action should be invoked for.
// A RestoreItemAction's Execute function will only be invoked on items that match the returned
// selector. A zero-valued ResourceSelector matches all resources.g
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"persistentvolumes"},
	}, nil
}

// Execute allows the RestorePlugin to perform arbitrary logic with the item being restored,
// in this case, overwriting AWS availability zone settings within Persistent Volumes.
func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	p.log.Info("AWS RestorePlugin")

	overrideAZ := os.Getenv(envAZOverride)
	if overrideAZ == "" {
		p.log.Infof("%s not found, using existing AZ for PV", envAZOverride)
		return velero.NewRestoreItemActionExecuteOutput(input.Item), nil
	}
	p.log.Infof("variable %s found, overriding to: %s", envAZOverride, overrideAZ)

	metadata, err := meta.Accessor(input.Item)
	if err != nil {
		return &velero.RestoreItemActionExecuteOutput{}, err
	}

	labels := metadata.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	if _, ok := labels[labelZoneBeta]; ok {
		labels[labelZoneBeta] = overrideAZ
	}
	if _, ok := labels[labelZone]; ok {
		labels[labelZone] = overrideAZ
	}

	metadata.SetLabels(labels)

	var pv corev1api.PersistentVolume
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(input.Item.UnstructuredContent(), &pv); err != nil {
		return nil, errors.WithStack(err)
	}

	if ebs := pv.Spec.AWSElasticBlockStore; ebs != nil {
		tmp := strings.Split(ebs.VolumeID, "/")
		if len(tmp) > 2 {
			tmp[2] = overrideAZ
			ebs.VolumeID = strings.Join(tmp, "/")
		} else {
			p.log.Infof("no AZ component found in EBS volume ID %s, so leaving it as is", ebs.VolumeID)
		}
	}

	if na := pv.Spec.NodeAffinity; na != nil {
		if r := na.Required; r != nil {
			if nst := r.NodeSelectorTerms; nst != nil {
				for i := range nst {
					if nst[i].MatchExpressions != nil {
						for j := range nst[i].MatchExpressions {
							if (nst[i].MatchExpressions[j].Key == labelZoneBeta || nst[i].MatchExpressions[j].Key == labelZone) && nst[i].MatchExpressions[j].Operator == "In" {
								p.log.Infof("Current node affinity set to %s, overriding to %s", nst[i].MatchExpressions[j].Values[0], overrideAZ)
								nst[i].MatchExpressions[j].Values = []string{overrideAZ}
							}
						}
					}
				}
			}
		}
	}

	inputMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&pv)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return velero.NewRestoreItemActionExecuteOutput(&unstructured.Unstructured{Object: inputMap}), nil
}
