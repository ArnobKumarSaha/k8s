package v1alpha1

import (
	"fmt"

	apps "k8s.io/api/apps/v1"
	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func (dst *MyPod) Duckify(srcRaw runtime.Object) error {
	gvk := srcRaw.GetObjectKind().GroupVersionKind()

	switch src := srcRaw.(type) {
	case *core.ReplicationController:
		dst.TypeMeta = metav1.TypeMeta{
			Kind:       "ReplicationController",
			APIVersion: core.SchemeGroupVersion.String(),
		}
		dst.ObjectMeta = src.ObjectMeta
		dst.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: src.Spec.Selector,
		}
		return nil
	case *apps.Deployment:
		dst.TypeMeta = metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: apps.SchemeGroupVersion.String(),
		}
		dst.ObjectMeta = src.ObjectMeta
		dst.Spec.Selector = src.Spec.Selector
		return nil
	case *apps.StatefulSet:
		dst.TypeMeta = metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: apps.SchemeGroupVersion.String(),
		}
		dst.ObjectMeta = src.ObjectMeta
		dst.Spec.Selector = src.Spec.Selector
		return nil
	case *apps.DaemonSet:
		dst.TypeMeta = metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: apps.SchemeGroupVersion.String(),
		}
		dst.ObjectMeta = src.ObjectMeta
		dst.Spec.Selector = src.Spec.Selector
		return nil
	case *batch.Job:
		dst.TypeMeta = metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: batch.SchemeGroupVersion.String(),
		}
		dst.ObjectMeta = src.ObjectMeta
		dst.Spec.Selector = src.Spec.Selector
		return nil
	case *batch.CronJob:
		dst.TypeMeta = metav1.TypeMeta{
			Kind:       "CronJob",
			APIVersion: batch.SchemeGroupVersion.String(),
		}
		dst.ObjectMeta = src.ObjectMeta
		dst.Spec.Selector = src.Spec.JobTemplate.Spec.Selector
		return nil
	case *unstructured.Unstructured:
		switch gvk {
		case apps.SchemeGroupVersion.WithKind("Deployment"),
			apps.SchemeGroupVersion.WithKind("StatefulSet"),
			apps.SchemeGroupVersion.WithKind("DaemonSet"),
			batch.SchemeGroupVersion.WithKind("Job"):
			return runtime.DefaultUnstructuredConverter.FromUnstructured(src.UnstructuredContent(), dst)
		case core.SchemeGroupVersion.WithKind("ReplicationController"):
			var obj core.ReplicationController
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(src.UnstructuredContent(), &obj); err != nil {
				return err
			}
			dst.TypeMeta = metav1.TypeMeta{
				Kind:       "ReplicationController",
				APIVersion: core.SchemeGroupVersion.String(),
			}
			dst.ObjectMeta = obj.ObjectMeta
			dst.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: obj.Spec.Selector,
			}
			return nil
		case batch.SchemeGroupVersion.WithKind("CronJob"):
			var obj batch.CronJob
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(src.UnstructuredContent(), &obj); err != nil {
				return err
			}
			dst.TypeMeta = metav1.TypeMeta{
				Kind:       "CronJob",
				APIVersion: batch.SchemeGroupVersion.String(),
			}
			dst.ObjectMeta = obj.ObjectMeta
			dst.Spec.Selector = obj.Spec.JobTemplate.Spec.Selector
			return nil
		}
	}
	return fmt.Errorf("unknown src type %T", srcRaw)
}
