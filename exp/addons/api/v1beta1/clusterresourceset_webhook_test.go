/*
Copyright 2021 The Kubernetes Authors.

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

package v1beta1

import (
	"testing"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	utildefaulting "sigs.k8s.io/cluster-api/util/defaulting"
)

func TestClusterResourcesetDefault(t *testing.T) {
	g := NewWithT(t)
	clusterResourceSet := &ClusterResourceSet{}
	defaultingValidationCRS := clusterResourceSet.DeepCopy()
	defaultingValidationCRS.Spec.ClusterSelector = metav1.LabelSelector{
		MatchLabels: map[string]string{"foo": "bar"},
	}
	t.Run("for ClusterResourceSet", utildefaulting.DefaultValidateTest(defaultingValidationCRS))
	clusterResourceSet.Default()

	g.Expect(clusterResourceSet.Spec.Strategy).To(Equal(string(ClusterResourceSetStrategyApplyOnce)))
}

func TestClusterResourcesetDefaultWithClusterResourceSetStrategyApplyAlways(t *testing.T) {
	g := NewWithT(t)
	clusterResourceSet := &ClusterResourceSet{
		Spec: ClusterResourceSetSpec{
			Strategy: string(ClusterResourceSetStrategyApplyAlways),
		},
	}
	defaultingValidationCRS := clusterResourceSet.DeepCopy()
	defaultingValidationCRS.Spec.ClusterSelector = metav1.LabelSelector{
		MatchLabels: map[string]string{"foo": "bar"},
	}
	t.Run("for ClusterResourceSet", utildefaulting.DefaultValidateTest(defaultingValidationCRS))
	clusterResourceSet.Default()

	g.Expect(clusterResourceSet.Spec.Strategy).To(Equal(string(ClusterResourceSetStrategyReconcile)))
}

func TestClusterResourceSetLabelSelectorAsSelectorValidation(t *testing.T) {
	tests := []struct {
		name      string
		selectors map[string]string
		expectErr bool
	}{
		{
			name:      "should not return error for valid selector",
			selectors: map[string]string{"foo": "bar"},
			expectErr: false,
		},
		{
			name:      "should return error for invalid selector",
			selectors: map[string]string{"-123-foo": "bar"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			clusterResourceSet := &ClusterResourceSet{
				Spec: ClusterResourceSetSpec{
					ClusterSelector: metav1.LabelSelector{
						MatchLabels: tt.selectors,
					},
				},
			}
			if tt.expectErr {
				warnings, err := clusterResourceSet.ValidateCreate()
				g.Expect(err).To(HaveOccurred())
				g.Expect(warnings).To(BeEmpty())
				warnings, err = clusterResourceSet.ValidateUpdate(clusterResourceSet)
				g.Expect(err).To(HaveOccurred())
				g.Expect(warnings).To(BeEmpty())
			} else {
				warnings, err := clusterResourceSet.ValidateCreate()
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(warnings).To(BeEmpty())
				warnings, err = clusterResourceSet.ValidateUpdate(clusterResourceSet)
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(warnings).To(BeEmpty())
			}
		})
	}
}

func TestClusterResourceSetStrategyImmutable(t *testing.T) {
	tests := []struct {
		name        string
		oldStrategy string
		newStrategy string
		expectErr   bool
	}{
		{
			name:        "when the Strategy has not changed",
			oldStrategy: string(ClusterResourceSetStrategyApplyOnce),
			newStrategy: string(ClusterResourceSetStrategyApplyOnce),
			expectErr:   false,
		},
		{
			name:        "when the Strategy has changed",
			oldStrategy: string(ClusterResourceSetStrategyApplyOnce),
			newStrategy: "",
			expectErr:   true,
		},
		{
			name:        "when the Strategy has changed, but the old value was ApplyAlways",
			oldStrategy: string(ClusterResourceSetStrategyApplyAlways),
			newStrategy: string(ClusterResourceSetStrategyReconcile),
			expectErr:   false,
		},
		{
			name:        "when the Strategy has changed, but the old value was ApplyAlways and the new value is ApplyOnce",
			oldStrategy: string(ClusterResourceSetStrategyApplyAlways),
			newStrategy: string(ClusterResourceSetStrategyApplyOnce),
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			newClusterResourceSet := &ClusterResourceSet{
				Spec: ClusterResourceSetSpec{
					ClusterSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test": "test",
						},
					},
					Strategy: tt.newStrategy,
				},
			}

			oldClusterResourceSet := &ClusterResourceSet{
				Spec: ClusterResourceSetSpec{
					ClusterSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test": "test",
						},
					},
					Strategy: tt.oldStrategy,
				},
			}

			warnings, err := newClusterResourceSet.ValidateUpdate(oldClusterResourceSet)
			if tt.expectErr {
				g.Expect(err).To(HaveOccurred())
				g.Expect(warnings).To(BeEmpty())
				return
			}
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(warnings).To(BeEmpty())
		})
	}
}

func TestClusterResourceSetClusterSelectorImmutable(t *testing.T) {
	tests := []struct {
		name               string
		oldClusterSelector map[string]string
		newClusterSelector map[string]string
		expectErr          bool
	}{
		{
			name:               "when the ClusterSelector has not changed",
			oldClusterSelector: map[string]string{"foo": "bar"},
			newClusterSelector: map[string]string{"foo": "bar"},
			expectErr:          false,
		},
		{
			name:               "when the ClusterSelector has changed",
			oldClusterSelector: map[string]string{"foo": "bar"},
			newClusterSelector: map[string]string{"foo": "different"},
			expectErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			newClusterResourceSet := &ClusterResourceSet{
				Spec: ClusterResourceSetSpec{
					ClusterSelector: metav1.LabelSelector{
						MatchLabels: tt.newClusterSelector,
					},
				},
			}

			oldClusterResourceSet := &ClusterResourceSet{
				Spec: ClusterResourceSetSpec{
					ClusterSelector: metav1.LabelSelector{
						MatchLabels: tt.oldClusterSelector,
					},
				},
			}

			warnings, err := newClusterResourceSet.ValidateUpdate(oldClusterResourceSet)
			if tt.expectErr {
				g.Expect(err).To(HaveOccurred())
				g.Expect(warnings).To(BeEmpty())
				return
			}
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(warnings).To(BeEmpty())
		})
	}
}

func TestClusterResourceSetSelectorNotEmptyValidation(t *testing.T) {
	g := NewWithT(t)
	clusterResourceSet := &ClusterResourceSet{}
	err := clusterResourceSet.validate(nil)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("selector must not be empty"))
}
