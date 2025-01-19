package controller

import (
    "context"
    "log"
    "strconv"

    v1 "k8s.io/api/apps/v1"
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/apimachinery/pkg/runtime/schema"
    "k8s.io/apimachinery/pkg/watch"
    "k8s.io/client-go/dynamic"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"
    "k8s.io/client-go/tools/cache"
    "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type HealthCheckSpec struct {
    Endpoint        string `json:"endpoint"`
    IntervalSeconds int    `json:"intervalSeconds"`
    ExpectedStatus  int    `json:"expectedStatus"`
    Auth            Auth   `json:"auth,omitempty"`
}

type Auth struct {
    MTLS  MTLS  `json:"mtls,omitempty"`
    OAuth OAuth `json:"oauth,omitempty"`
}

type MTLS struct {
    SecretName string `json:"secretName"`
}

type OAuth struct {
    ClientID     string `json:"clientId"`
    ClientSecret string `json:"clientSecret"`
    TokenURL     string `json:"tokenUrl"`
}

func StartController() error {
    config, err := rest.InClusterConfig()
    if err != nil {
        return err
    }

    clientset, err := kubernetes.NewForConfig(config)
    if err != nil {
        return err
    }

    dynClient, err := dynamic.NewForConfig(config)
    if err != nil {
        return err
    }

    resource := schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "healthchecks"}

    informer := cache.NewSharedIndexInformer(
        &cache.ListWatch{
            ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
                return dynClient.Resource(resource).Namespace(corev1.NamespaceAll).List(context.Background(), options)
            },
            WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
                return dynClient.Resource(resource).Namespace(corev1.NamespaceAll).Watch(context.Background(), options)
            },
        },
        &unstructured.Unstructured{},
        0, // Skip resyncing
        cache.Indexers{},
    )

    informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
        AddFunc: func(obj interface{}) {
            u := obj.(*unstructured.Unstructured)
            spec := parseSpec(u.Object)
            createDeployment(clientset, spec, u.GetNamespace(), u.GetName())
        },
    })

    stop := make(chan struct{})
    defer close(stop)
    go informer.Run(stop)
    <-stop

    return nil
}

func parseSpec(obj map[string]interface{}) HealthCheckSpec {
    // Parse the spec from the unstructured object
    spec := HealthCheckSpec{}
    spec.Endpoint = obj["endpoint"].(string)
    spec.IntervalSeconds = int(obj["intervalSeconds"].(int64))
    spec.ExpectedStatus = int(obj["expectedStatus"].(int64))

    if auth, ok := obj["auth"].(map[string]interface{}); ok {
        if mtls, ok := auth["mtls"].(map[string]interface{}); ok {
            spec.Auth.MTLS.SecretName = mtls["secretName"].(string)
        }

        if oauth, ok := auth["oauth"].(map[string]interface{}); ok {
            spec.Auth.OAuth.ClientID = oauth["clientId"].(string)
            spec.Auth.OAuth.ClientSecret = oauth["clientSecret"].(string)
            spec.Auth.OAuth.TokenURL = oauth["tokenUrl"].(string)
        }
    }

    return spec
}

func createDeployment(clientset *kubernetes.Clientset, spec HealthCheckSpec, namespace, name string) {
    deployment := &v1.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      name,
            Namespace: namespace,
        },
        Spec: v1.DeploymentSpec{
            Replicas: int32Ptr(1),
            Selector: &metav1.LabelSelector{
                MatchLabels: map[string]string{
                    "app": name,
                },
            },
            Template: corev1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: map[string]string{
                        "app": name,
                    },
                },
                Spec: corev1.PodSpec{
                    Containers: []corev1.Container{
                        {
                            Name:  "healthcheck",
                            Image: "placeholder-for-your-deployment-image",
                            Env: []corev1.EnvVar{
                                {
                                    Name:  "HEALTH_ENDPOINT",
                                    Value: spec.Endpoint,
                                },
                                {
                                    Name:  "INTERVAL_SECONDS",
                                    Value: strconv.Itoa(spec.IntervalSeconds),
                                },
                                {
                                    Name:  "EXPECTED_STATUS",
                                    Value: strconv.Itoa(spec.ExpectedStatus),
                                },
                            },
                        },
                    },
                },
            },
        },
    }

    if spec.Auth.MTLS.SecretName != "" {
        deployment.Spec.Template.Spec.Volumes = []corev1.Volume{
            {
                Name: "mtls-certs",
                VolumeSource: corev1.VolumeSource{
                    Secret: &corev1.SecretVolumeSource{
                        SecretName: spec.Auth.MTLS.SecretName,
                    },
                },
            },
        }

        deployment.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
            {
                Name:      "mtls-certs",
                MountPath: "/etc/mtls",
                ReadOnly:  true,
            },
        }
        deployment.Spec.Template.Spec.Containers[0].Env = append(deployment.Spec.Template.Spec.Containers[0].Env,
            corev1.EnvVar{
                Name:  "MTLS_CERTS_PATH",
                Value: "/etc/mtls",
            },
        )
    }

    if spec.Auth.OAuth.ClientID != "" && spec.Auth.OAuth.ClientSecret != "" && spec.Auth.OAuth.TokenURL != "" {
        deployment.Spec.Template.Spec.Containers[0].Env = append(deployment.Spec.Template.Spec.Containers[0].Env,
            corev1.EnvVar{
                Name:  "OAUTH_CLIENT_ID",
                Value: spec.Auth.OAuth.ClientID,
            },
            corev1.EnvVar{
                Name:  "OAUTH_CLIENT_SECRET",
                Value: spec.Auth.OAuth.ClientSecret,
            },
            corev1.EnvVar{
                Name:  "OAUTH_TOKEN_URL",
                Value: spec.Auth.OAuth.TokenURL,
            },
        )
    }

    _, err := clientset.AppsV1().Deployments(namespace).Create(context.Background(), deployment, metav1.CreateOptions{})
    if err != nil {
        log.Printf("Error creating deployment: %v", err)
    }
}

func int32Ptr(i int32) *int32 { return &i }