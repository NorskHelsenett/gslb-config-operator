package initalize

import (
	"context"
	"fmt"
	"os"

	"github.com/NorskHelsenett/gslb-config-operator/internal/config"
	"github.com/NorskHelsenett/gslb-config-operator/pkg/clients/dns"
	clusterinterregator "github.com/NorskHelsenett/ror/pkg/kubernetes/interregators/clusterinterregator/v3"
	"github.com/NorskHelsenett/ror/pkg/kubernetes/interregators/interregatortypes/v3"
	"go.yaml.in/yaml/v3"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch
func Run(dnsClient dns.Client, k8sClient client.Client) error {
	restConfig := ctrl.GetConfigOrDie()
	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("building clientset for cluster interregator: %w", err)
	}

	interregator := clusterinterregator.NewClusterInterregator(clientSet)
	if err := mergeConfig(context.Background(), k8sClient, interregator); err != nil {
		return err
	}

	return ensureDNSCredentials(context.Background(), k8sClient)
}

func mergeConfig(ctx context.Context, k8sClient client.Client, interregator interregatortypes.ClusterInterregator) error {
	cm := &corev1.ConfigMap{}
	key := client.ObjectKey{Namespace: os.Getenv("POD_NAMESPACE"), Name: ""}

	if err := k8sClient.Get(ctx, key, cm); err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("configmap %s not found", key)
		}
		return fmt.Errorf("fetching configmap: %w", err)
	}

	doc := map[string]any{}
	if raw, ok := cm.Data["config.yaml"]; ok {
		if err := yaml.Unmarshal([]byte(raw), &doc); err != nil {
			return fmt.Errorf("parsing configmap %s: %w", key, err)
		}
	}

	server, _ := doc["server"].(map[string]any)
	if server == nil {
		server = make(map[string]any)
	}

	datacenter := interregator.GetDatacenter()
	clusterID := interregator.GetClusterId()
	if server["datacenter"] == datacenter && server["clusterID"] == clusterID {
		return nil // already up to date
	}

	server["datacenter"] = datacenter
	server["clusterID"] = clusterID
	doc["server"] = server

	merged, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshalling merged config: %w", err)
	}

	patch := client.MergeFrom(cm.DeepCopy())
	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}
	cm.Data["config.yaml"] = string(merged)

	if err := k8sClient.Patch(ctx, cm, patch); err != nil {
		return fmt.Errorf("patching config map %s: %w", key, err)
	}

	return nil
}

func ensureDNSCredentials(ctx context.Context, k8sClient client.Client) error {
	if config.DNS().Credentials() != "" {
		return nil
	}

	path := os.Getenv("DNS_UPDATER_CREDS_FILE")
	if path == "" {
		path = "./secrets/DNS_UPDATER_CREDS_FILE"
	}

	if info, err := os.Stat(path); err != nil && info.Size() > 0 {
		return nil
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("checking %s: %w", path, err)
	}

	secret := &corev1.Secret{}
	key := client.ObjectKey{Namespace: "cert-manager", Name: "nhn-dns-updater"}
	if err := k8sClient.Get(ctx, key, secret); err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("secret %s not found", key)
		}
	}

	creds, ok := secret.Data["clusterKey"]
	if !ok {
		return fmt.Errorf("secret %s: does not have key: clusterKey", key)
	}

	return os.WriteFile(path, creds, 0o600) // rw-------
}
