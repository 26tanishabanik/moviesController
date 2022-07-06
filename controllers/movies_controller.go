/*
Copyright 2022.

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

package controllers

import (
	"context"
	"fmt"
	ratingv1 "github.com/26tanishabanik/imdb-controller/api/v1"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
	"time"
)

// MoviesReconciler reconciles a Movies object
type MoviesReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=rating.tanisha.banik,resources=movies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rating.tanisha.banik,resources=movies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=rating.tanisha.banik,resources=movies/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Movies object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *MoviesReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log = log.WithValues("movies", req.NamespacedName)
	var movie ratingv1.Movies
	if err := r.Get(ctx, req.NamespacedName, &movie); err != nil {
		log.Error(err, "could not get the Movies object")
		return ctrl.Result{}, err
	}
	if movie.Status.Rating == "" {
		log.Info(fmt.Sprintf("Reconciling for %s", req.NamespacedName))
		log.Info(fmt.Sprintf("Movie Name is: %s", movie.Spec.MovieName))
		// TODO(user): your logic here
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("job-%s", req.Name),
				Namespace: "default",
			},
			Spec: corev1.PodSpec{
				RestartPolicy: "Never",
				Containers: []corev1.Container{
					{
						Name:  "imdb-fetcher",
						Image: "26tanishabanik/imdb:v1",
						Args:  []string{movie.Spec.MovieName},
					},
				},
			},
		}
		if err := r.Create(ctx, &pod, &client.CreateOptions{}); err != nil {
			log.Error(err, "could not create the container")
			return ctrl.Result{}, err
		}
		log.Info("Created the container")
		time.Sleep(10 * time.Second)
		answer, err := readPodLogs(pod, ctx)
		if err != nil {
			log.Error(err, "could not read logs")
			return ctrl.Result{}, err
		}

		// log.Info(fmt.Sprintf("Answer is %s", answer))
		movie.Status.Rating = "rating"
		log.Info(fmt.Sprintf("Answer is %s", answer))
		if err := r.Update(ctx, &movie, &client.UpdateOptions{}); err != nil {
			log.Error(err, "could not update pod")
			return ctrl.Result{}, err
		}
		log.Info(fmt.Sprintf("Updated Pod status is  %s", movie.Status.Rating))
	}
	return ctrl.Result{}, nil
}
func readPodLogs(pod corev1.Pod, ctx context.Context) (string, error) {
	config := ctrl.GetConfigOrDie()
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", err
	}

	req := clientSet.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{})

	reader, err := req.Stream(ctx)
	if err != nil {
		return "", err
	}

	defer reader.Close()

	answer, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", err
	}
	res1 := strings.Split(string(answer), "-")

	return res1[len(res1)-2], nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MoviesReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ratingv1.Movies{}).
		Complete(r)
}
