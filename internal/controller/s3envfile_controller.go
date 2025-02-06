/*
Copyright 2025 Nikolas Sepos.

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

package controller

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	configv1alpha1 "github.com/greationtech/s3-configmap-controller/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// S3EnvFileReconciler reconciles a S3EnvFile object
type S3EnvFileReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=config.greatlion.tech,resources=s3envfiles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=config.greatlion.tech,resources=s3envfiles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=config.greatlion.tech,resources=s3envfiles/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the S3EnvFile object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *S3EnvFileReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	s3EnvFile := &configv1alpha1.S3EnvFile{}
	if err := r.Get(ctx, req.NamespacedName, s3EnvFile); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("S3EnvFile resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get S3EnvFile")
		return ctrl.Result{}, err
	}

	target := types.NamespacedName{
		Name:      s3EnvFile.Spec.ConfigMapName,
		Namespace: s3EnvFile.Namespace,
	}
	configMap := &corev1.ConfigMap{}
	if err := r.Get(ctx, target, configMap); err != nil {
		if apierrors.IsNotFound(err) {
			cm, err := r.generateConfigMap(ctx, s3EnvFile)
			if err != nil {
				log.Error(err, "Failed to generate ConfigMap")
				return ctrl.Result{}, err
			}
			if err := r.Create(ctx, cm); err != nil {
				log.Error(err, "Failed to create ConfigMap")
				return ctrl.Result{}, err
			}
			return ctrl.Result{RequeueAfter: time.Minute}, nil
		}
	}

	return ctrl.Result{}, nil
}

func (r *S3EnvFileReconciler) generateConfigMap(ctx context.Context, s3EnvFile *configv1alpha1.S3EnvFile) (*corev1.ConfigMap, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	cfg.Region = s3EnvFile.Spec.Region

	client := s3.NewFromConfig(cfg)

	res, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s3EnvFile.Spec.Bucket),
		Key:    aws.String(s3EnvFile.Spec.Key),
	})
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	env, err := envKeyValFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s3EnvFile.Spec.ConfigMapName,
			Namespace: s3EnvFile.Namespace,
		},
		Data: make(map[string]string),
	}
	for k, v := range env {
		configMap.Data[k] = v
	}

	if err := controllerutil.SetControllerReference(s3EnvFile, configMap, r.Scheme); err != nil {
		return nil, err
	}

	return configMap, nil
}

func envKeyValFromReader(r io.Reader) (map[string]string, error) {
	scanner := bufio.NewScanner(r)
	env := make(map[string]string)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}

		parts := bytes.SplitN([]byte(line), []byte("="), 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid line: %s", line)
		}
		env[string(parts[0])] = string(parts[1])
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return env, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *S3EnvFileReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&configv1alpha1.S3EnvFile{}).
		Named("s3envfile").
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
