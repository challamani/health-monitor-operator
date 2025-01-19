package controller

import (
    "context"
    "fmt"
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
        return fmt.Errorf("failed to get in-cluster config: %v", err)
    }

    clientset, err := kubernetes.NewForConfig(config)
    if err != nil {
        return fmt.Errorf("failed to create kubernetes clientset: %v", err)
    }

    dynClient, err := dynamic.NewForConfig(config)
    if err != nil {
        return fmt.Errorf("failed to create dynamic client: %v", err)
    }

    resource := schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "healthchecks"}

    informer := cache.NewSharedIndexInformer(
        &cache.ListWatch{
            ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
                log.Println("Listing healthchecks")
                return dynClient.Resource(resource).Namespace(corev1.NamespaceAll).List(context.Background(), options)
            },
            WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
                log.Println("Watching healthchecks")
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
            log.Printf("Healthcheck added: %s/%s", u.GetNamespace(), u.GetName())

            // Log the entire object to understand its structure
            log.Printf("Unstructured object: %+v", u.Object)

            spec, err := parseSpec(u.Object)
            if err != nil {
                log.Printf("Failed to parse spec: %v", err)
                return
            }
            createDeployment(clientset, spec, u.GetNamespace(), u.GetName())
        },
        DeleteFunc: func(obj interface{}) {
            u := obj.(*unstructured.Unstructured)
            log.Printf("Healthcheck deleted: %s/%s", u.GetNamespace(), u.GetName())
            deleteDeployment(clientset, u.GetNamespace(), u.GetName())
        },
    })

    stop := make(chan struct{})
    defer close(stop)
    go informer.Run(stop)
    <-stop

    return nil
}

func parseSpec(obj map[string]interface{}) (HealthCheckSpec, error) {
    var spec HealthCheckSpec

    // Log the obj to debug the structure
    log.Printf("Parsing spec from object: %+v", obj)

    specMap, ok := obj["spec"].(map[string]interface{})
    if !ok {
        return spec, fmt.Errorf("spec field is missing or not a map")
    }

    if endpoint, ok := specMap["endpoint"].(string); ok {
        spec.Endpoint = endpoint
    } else {
        return spec, fmt.Errorf("endpoint is missing or not a string")
    }

    if interval, ok := specMap["intervalSeconds"].(int64); ok {
        spec.IntervalSeconds = int(interval)
    } else {
        return spec, fmt.Errorf("intervalSeconds is missing or not an int64")
    }

    if expectedStatus, ok := specMap["expectedStatus"].(int64); ok {
        spec.ExpectedStatus = int(expectedStatus)
    } else {
        return spec, fmt.Errorf("expectedStatus is missing or not an int64")
    }

    if auth, ok := specMap["auth"].(map[string]interface{}); ok {
        if mtls, ok := auth["mtls"].(map[string]interface{}); ok {
            if secretName, ok := mtls["secretName"].(string); ok {
                spec.Auth.MTLS.SecretName = secretName
            }
        }

        if oauth, ok := auth["oauth"].(map[string]interface{}); ok {
            if clientID, ok := oauth["clientId"].(string); ok {
                spec.Auth.OAuth.ClientID = clientID
            }
            if clientSecret, ok := oauth["clientSecret"].(string); ok {
                spec.Auth.OAuth.ClientSecret = clientSecret
            }
            if tokenURL, ok := oauth["tokenUrl"].(string); ok {
                spec.Auth.OAuth.TokenURL = tokenURL
            }
        }
    }

    return spec, nil
}

func createDeployment(clientset *kubernetes.Clientset, spec HealthCheckSpec, namespace, name string) {
    log.Printf("Creating deployment for healthcheck: %s/%s", namespace, name)
    
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
                            Image: "docker.io/library/healthcheck-monitor:latest",
                            ImagePullPolicy: "Never",
                            Env: []corev1.EnvVar{
                                {
                                    Name:  "HEALTH_ENDPOINT",
                                    Value: spec.Endpoint,
                                },
                                {
                                    Name:  "HEALTH_INTERVAL",
                                    Value: strconv.Itoa(spec.IntervalSeconds),
                                },
                                {
                                    Name:  "HEALTH_EXPECTEDSTATUS",
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
    } else {
        log.Printf("Deployment created successfully: %s/%s", namespace, name)
    }
}

func deleteDeployment(clientset *kubernetes.Clientset, namespace, name string) {
    log.Printf("Deleting deployment for healthcheck: %s/%s", namespace, name)
    err := clientset.AppsV1().Deployments(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
    if err != nil {
        log.Printf("Error deleting deployment: %v", err)
    } else {
        log.Printf("Deployment deleted successfully: %s/%s", namespace, name)
    }
}

func int32Ptr(i int32) *int32 { return &i }