/*
Copyright 2019 The Knative Authors

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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1alpha1 "github.com/knative/serving-operator/pkg/apis/serving/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeKnativeServings implements KnativeServingInterface
type FakeKnativeServings struct {
	Fake *FakeServingV1alpha1
	ns   string
}

var knativeservingsResource = schema.GroupVersionResource{Group: "serving.knative.dev", Version: "v1alpha1", Resource: "knativeservings"}

var knativeservingsKind = schema.GroupVersionKind{Group: "serving.knative.dev", Version: "v1alpha1", Kind: "KnativeServing"}

// Get takes name of the knativeServing, and returns the corresponding knativeServing object, and an error if there is any.
func (c *FakeKnativeServings) Get(name string, options v1.GetOptions) (result *v1alpha1.KnativeServing, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(knativeservingsResource, c.ns, name), &v1alpha1.KnativeServing{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.KnativeServing), err
}

// List takes label and field selectors, and returns the list of KnativeServings that match those selectors.
func (c *FakeKnativeServings) List(opts v1.ListOptions) (result *v1alpha1.KnativeServingList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(knativeservingsResource, knativeservingsKind, c.ns, opts), &v1alpha1.KnativeServingList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.KnativeServingList{ListMeta: obj.(*v1alpha1.KnativeServingList).ListMeta}
	for _, item := range obj.(*v1alpha1.KnativeServingList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested knativeServings.
func (c *FakeKnativeServings) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(knativeservingsResource, c.ns, opts))

}

// Create takes the representation of a knativeServing and creates it.  Returns the server's representation of the knativeServing, and an error, if there is any.
func (c *FakeKnativeServings) Create(knativeServing *v1alpha1.KnativeServing) (result *v1alpha1.KnativeServing, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(knativeservingsResource, c.ns, knativeServing), &v1alpha1.KnativeServing{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.KnativeServing), err
}

// Update takes the representation of a knativeServing and updates it. Returns the server's representation of the knativeServing, and an error, if there is any.
func (c *FakeKnativeServings) Update(knativeServing *v1alpha1.KnativeServing) (result *v1alpha1.KnativeServing, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(knativeservingsResource, c.ns, knativeServing), &v1alpha1.KnativeServing{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.KnativeServing), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeKnativeServings) UpdateStatus(knativeServing *v1alpha1.KnativeServing) (*v1alpha1.KnativeServing, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(knativeservingsResource, "status", c.ns, knativeServing), &v1alpha1.KnativeServing{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.KnativeServing), err
}

// Delete takes name of the knativeServing and deletes it. Returns an error if one occurs.
func (c *FakeKnativeServings) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(knativeservingsResource, c.ns, name), &v1alpha1.KnativeServing{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeKnativeServings) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(knativeservingsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.KnativeServingList{})
	return err
}

// Patch applies the patch and returns the patched knativeServing.
func (c *FakeKnativeServings) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.KnativeServing, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(knativeservingsResource, c.ns, name, pt, data, subresources...), &v1alpha1.KnativeServing{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.KnativeServing), err
}
