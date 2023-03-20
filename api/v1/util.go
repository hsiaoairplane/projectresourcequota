/*
Copyright 2023 JenTing.

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

package v1

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

func AddFinalizer(name string, obj runtime.Object) error {
	metadata, err := meta.Accessor(obj)
	if err != nil {
		return err
	}

	exists := false
	finalizers := metadata.GetFinalizers()
	for _, f := range finalizers {
		if f == name {
			exists = true
			break
		}
	}
	if !exists {
		metadata.SetFinalizers(append(finalizers, name))
	}

	return nil
}

func RemoveFinalizer(name string, obj runtime.Object) error {
	metadata, err := meta.Accessor(obj)
	if err != nil {
		return err
	}

	var finalizers []string
	for _, finalizer := range metadata.GetFinalizers() {
		if finalizer == name {
			continue
		}
		finalizers = append(finalizers, finalizer)
	}
	metadata.SetFinalizers(finalizers)

	return nil
}

func IsAnnotationExists(obj runtime.Object, key string) bool {
	metadata, err := meta.Accessor(obj)
	if err != nil {
		return false
	}

	_, found := metadata.GetAnnotations()[key]
	return found
}

func AddAnnotation(obj runtime.Object, key, val string) error {
	objMeta, err := meta.Accessor(obj)
	if err != nil {
		return fmt.Errorf("cannot add annotation for invalid object %v: %v", obj, err)
	}

	annos := objMeta.GetAnnotations()
	if annos == nil {
		annos = map[string]string{}
	}
	annos[key] = val
	objMeta.SetAnnotations(annos)

	return nil
}

func RemoveAnnotation(obj runtime.Object, key string) error {
	metadata, err := meta.Accessor(obj)
	if err != nil {
		return fmt.Errorf("cannot remove annotation for invalid object %v: %v", obj, err)
	}

	annos := metadata.GetAnnotations()
	delete(annos, key)
	metadata.SetAnnotations(annos)

	return nil
}
