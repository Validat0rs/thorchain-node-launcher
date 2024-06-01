package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	snapv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	snapclientset "github.com/kubernetes-csi/external-snapshotter/client/v4/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

////////////////////////////////////////////////////////////////////////////////////////
// Config
////////////////////////////////////////////////////////////////////////////////////////

type Config struct {
	StateSyncPodName     string `mapstructure:"state_sync_pod_name"`
	ExportGenesisPodName string `mapstructure:"export_genesis_pod_name"`
	CreateArchivePodName string `mapstructure:"create_archive_pod_name"`
	UploadPodName        string `mapstructure:"upload_pod_name"`
	PVCName              string `mapstructure:"pvc_name"`
	VolumeSnapshotName   string `mapstructure:"volume_snapshot_name"`

	ThornodeImage      string `mapstructure:"thornode_image"`
	ThornodeChainID    string `mapstructure:"thornode_chain_id"`
	ThornodeRPCServers string `mapstructure:"thornode_rpc_servers"`

	MinioImage string `mapstructure:"minio_image"`

	PVCSize              string `mapstructure:"pvc_size"`
	StateSyncCPU         string `mapstructure:"state_sync_cpu"`
	StateSyncMemory      string `mapstructure:"state_sync_memory"`
	StateSyncTolerations string `mapstructure:"state_sync_tolerations"`

	Namespace string `mapstructure:"namespace"`

	DiscordWebhookMainnetInfo string `mapstructure:"discord_webhook_mainnet_info"`

	// Some setups have a service mesh sidecar proxy that needs to initialize before
	// start, or manually quit upon completion.
	ReadyEndpoint string `mapstructure:"ready_endpoint"`
	QuitEndpoint  string `mapstructure:"quit_endpoint"`
}

func (c Config) PodQuitCommand() string {
	if c.QuitEndpoint == "" {
		return ""
	}
	return fmt.Sprintf("curl -sX POST %s > /dev/null", c.QuitEndpoint)
}

func (c Config) Validate() {
	// assert any required config without defaults is set
	if c.ThornodeRPCServers == "" {
		log.Fatal().Msg("missing required THORNODE_RPC_SERVERS")
	}
	if c.PVCSize == "" {
		log.Fatal().Msg("missing required PVC_SIZE")
	}
	if c.StateSyncCPU == "" {
		log.Fatal().Msg("missing required STATE_SYNC_CPU")
	}
	if c.StateSyncMemory == "" {
		log.Fatal().Msg("missing required STATE_SYNC_MEMORY")
	}
	if c.MinioImage == "" {
		log.Fatal().Msg("missing required MINIO_IMAGE")
	}
}

var config Config

func init() {
	defaults := Config{
		StateSyncPodName:     "thornode-statesync",
		ExportGenesisPodName: "thornode-export-genesis",
		CreateArchivePodName: "thornode-create-archive",
		UploadPodName:        "thornode-upload",
		PVCName:              "thornode-snapshot-pvc",
		VolumeSnapshotName:   "thornode-latest-statesync",
		ThornodeImage:        "registry.gitlab.com/thorchain/thornode:mainnet",
		ThornodeChainID:      "thorchain-mainnet-v1",
	}

	// parse all fields and set defaults so they may be overridden by env vars
	rt := reflect.TypeOf(defaults)
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if field.Tag.Get("mapstructure") == "" {
			log.Fatal().Msgf("missing mapstructure tag for field %s", field.Name)
		}
		fieldValue := reflect.ValueOf(&defaults).Elem().FieldByName(field.Name)
		viper.SetDefault(field.Tag.Get("mapstructure"), fieldValue.Interface())
	}

	// read overrides from env vars
	viper.AutomaticEnv()

	// read the namespace from file
	ns, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get namespace")
	}
	viper.Set("namespace", string(ns))

	// load config
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatal().Err(err).Msg("failed to unmarshal config")
	}

	// validate config
	config.Validate()
}

////////////////////////////////////////////////////////////////////////////////////////
// RunPod
////////////////////////////////////////////////////////////////////////////////////////

type RunPod struct {
	PodName     string
	Image       string
	Command     []string
	CPU         string
	Memory      string
	ExtraEnv    []corev1.EnvVar
	Tolerations string
}

func (r RunPod) Run(ctx context.Context, cs *kubernetes.Clientset) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.PodName,
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:            r.PodName,
					Image:           r.Image,
					ImagePullPolicy: corev1.PullAlways,
					Command:         r.Command,
					Env: []corev1.EnvVar{
						{
							Name:  "CHAIN_ID",
							Value: config.ThornodeChainID,
						},
						{
							Name:  "NET",
							Value: "mainnet",
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      config.PVCName,
							MountPath: "/root",
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: config.PVCName,
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: config.PVCName,
						},
					},
				},
			},
		},
	}

	// set cpu/memory requests
	requests := corev1.ResourceList{}
	if r.CPU != "" {
		requests[corev1.ResourceCPU] = resource.MustParse(r.CPU)
	}
	if r.Memory != "" {
		requests[corev1.ResourceMemory] = resource.MustParse(r.Memory)
	}
	pod.Spec.Containers[0].Resources = corev1.ResourceRequirements{
		Requests: requests,
	}

	// add extra env vars
	pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, r.ExtraEnv...)

	// add tolerations
	tolerations := strings.Split(r.Tolerations, ",")
	for _, t := range tolerations {
		pod.Spec.Tolerations = append(pod.Spec.Tolerations, corev1.Toleration{
			Key:      t,
			Operator: corev1.TolerationOpExists,
			Effect:   corev1.TaintEffectNoSchedule,
		})
	}

	_, err := cs.CoreV1().Pods(config.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create pod")
	}
}

func (r RunPod) WaitUntilComplete(ctx context.Context, cs *kubernetes.Clientset) {
	tick := time.NewTicker(10 * time.Second)
	defer tick.Stop()
	for range tick.C {
		// get export genesis pod
		pod, err := cs.CoreV1().Pods(config.Namespace).Get(ctx, r.PodName, metav1.GetOptions{})
		if err != nil {
			log.Error().Err(err).Msg("failed to get pod")
		}

		// wait for success or failure
		switch pod.Status.Phase {
		case corev1.PodPending, corev1.PodRunning:
			continue
		case corev1.PodSucceeded:
			log.Info().Str("pod", r.PodName).Msg("pod succeeded")
			return
		case corev1.PodFailed:
			log.Fatal().Str("pod", r.PodName).Msg("pod failed")
		default:
			log.Error().Str("pod", r.PodName).Msg("pod in unknown state")
		}
	}
}

////////////////////////////////////////////////////////////////////////////////////////
// Stages
////////////////////////////////////////////////////////////////////////////////////////

func stopAndReset(ctx context.Context, cs *kubernetes.Clientset) {
	log.Info().Msg("stopping and resetting")

	// delete any current statesync, genesis, and snapshot pods
	pods := []string{
		config.StateSyncPodName,
		config.ExportGenesisPodName,
		config.CreateArchivePodName,
		config.UploadPodName,
	}
	for _, pod := range pods {
		err := cs.CoreV1().Pods(config.Namespace).Delete(ctx, pod, metav1.DeleteOptions{})
		if err != nil {
			log.Error().Err(err).Msg("failed to delete pod")
		}
	}

	// delete existing pvc
	pvcs := cs.CoreV1().PersistentVolumeClaims(config.Namespace)
	err := pvcs.Delete(ctx, config.PVCName, metav1.DeleteOptions{})
	if err != nil {
		log.Error().Err(err).Msg("failed to delete pvc")
	}

	// wait for pvc to be deleted
	tick := time.NewTicker(10 * time.Second)
	defer tick.Stop()
	for range tick.C {
		_, err := pvcs.Get(ctx, config.PVCName, metav1.GetOptions{})
		if err != nil {
			log.Info().Err(err).Msg("pvc deleted")
			break
		}
	}

	_, err = pvcs.Create(ctx, &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: config.PVCName,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(config.PVCSize),
				},
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create pvc")
	}
}

func statesyncRecover(ctx context.Context, cs *kubernetes.Clientset) {
	log.Info().Msg("recovering statesync")

	// create statesync pod
	rp := &RunPod{
		PodName: config.StateSyncPodName,
		Image:   config.ThornodeImage,
		Command: []string{
			"/scripts/fullnode.sh",
		},
		CPU:    config.StateSyncCPU,
		Memory: config.StateSyncMemory,
		ExtraEnv: []corev1.EnvVar{
			{
				Name:  "THOR_AUTO_STATE_SYNC_ENABLED",
				Value: "true",
			},
			{
				Name:  "THOR_TENDERMINT_STATE_SYNC_RPC_SERVERS",
				Value: config.ThornodeRPCServers,
			},
		},
		Tolerations: config.StateSyncTolerations,
	}
	rp.Run(ctx, cs)

	// wait for statesync to complete
	tick := time.NewTicker(10 * time.Second)
	defer tick.Stop()
	for range tick.C {
		// get pod
		pod, err := cs.CoreV1().Pods(config.Namespace).Get(ctx, config.StateSyncPodName, metav1.GetOptions{})
		if err != nil {
			log.Error().Err(err).Msg("failed to get statesync pod")
			continue
		}

		switch pod.Status.Phase {
		case corev1.PodSucceeded, corev1.PodFailed, corev1.PodUnknown:
			log.Fatal().Str("phase", string(pod.Status.Phase)).Msg("statesync pod completed unexpectedly")
		case corev1.PodPending:
			log.Info().Str("phase", string(pod.Status.Phase)).Msg("statesync pod not running")
			continue
		case corev1.PodRunning:
			// check if statesync is complete
		}

		// get statesync pod status
		res, err := http.Get(fmt.Sprintf("http://%s:27147/status", pod.Status.PodIP))
		if err != nil {
			log.Error().Err(err).Msg("failed to get statesync status")
			continue
		}
		defer res.Body.Close()

		// decode status response
		var status struct {
			Result struct {
				SyncInfo struct {
					LatestBlockHeight string `json:"latest_block_height"`
				} `json:"sync_info"`
			} `json:"result"`
		}
		err = json.NewDecoder(res.Body).Decode(&status)
		if err != nil {
			log.Error().Err(err).Msg("failed to decode statesync status")
			continue
		}

		// if we have a height statesync recover is complete
		if status.Result.SyncInfo.LatestBlockHeight != "0" {
			log.Info().Str("height", status.Result.SyncInfo.LatestBlockHeight).Msg("statesync recover complete")
			break
		}
	}

	// delete statesync pod
	err := cs.CoreV1().Pods(config.Namespace).Delete(ctx, config.StateSyncPodName, metav1.DeleteOptions{})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to delete statesync pod")
	}
}

func volumeSnapshot(ctx context.Context, cs *snapclientset.Clientset) {
	log.Info().Msg("creating volume snapshot")

	// delete volume snapshot
	err := cs.SnapshotV1().VolumeSnapshots(config.Namespace).Delete(ctx, config.VolumeSnapshotName, metav1.DeleteOptions{})
	if err != nil {
		log.Error().Err(err).Msg("failed to delete volume snapshot")
	}

	// wait for volume snapshot to be deleted
	tick := time.NewTicker(10 * time.Second)
	defer tick.Stop()
	for range tick.C {
		_, err := cs.SnapshotV1().VolumeSnapshots(config.Namespace).Get(ctx, config.VolumeSnapshotName, metav1.GetOptions{})
		if err != nil {
			break
		}
	}

	// create volume snapshot
	name := config.PVCName
	_, err = cs.SnapshotV1().VolumeSnapshots(config.Namespace).Create(ctx, &snapv1.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name: config.VolumeSnapshotName,
		},
		Spec: snapv1.VolumeSnapshotSpec{
			Source: snapv1.VolumeSnapshotSource{
				PersistentVolumeClaimName: &name,
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create volume snapshot")
	}
}

func exportGenesis(ctx context.Context, cs *kubernetes.Clientset) (height int64) {
	log.Info().Msg("exporting genesis")

	// create export genesis pod
	rp := &RunPod{
		PodName: config.ExportGenesisPodName,
		Image:   config.ThornodeImage,
		Command: []string{
			"sh",
			"-c",
			`
			thornode export > /root/genesis.json;
			jq '.initial_height|tonumber' /root/genesis.json;
			` + config.PodQuitCommand(),
		},
		CPU:    "2",
		Memory: "16Gi",
	}
	rp.Run(ctx, cs)
	rp.WaitUntilComplete(ctx, cs)

	// get height (last line of output)
	lines := int64(1)
	out, err := cs.CoreV1().Pods(config.Namespace).GetLogs(config.ExportGenesisPodName, &corev1.PodLogOptions{
		Container: config.ExportGenesisPodName,
		TailLines: &lines,
	}).Do(ctx).Raw()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get export genesis pod logs")
	}

	// parse height
	height, err = strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse genesis height")
	}
	log.Info().Int64("height", height).Msg("exported genesis")
	return height
}

func createArchive(ctx context.Context, cs *kubernetes.Clientset, height int64) {
	log.Info().Msg("creating archive")

	// create archive pod
	rp := &RunPod{
		PodName: config.CreateArchivePodName,
		Image:   config.ThornodeImage,
		Command: []string{
			"sh",
			"-c",
			fmt.Sprintf(`
			tar -czvf /root/%d.tar.gz -C /root/.thornode data;
			`+config.PodQuitCommand(), height),
		},
		CPU:    "2",
		Memory: "16Gi",
	}
	rp.Run(ctx, cs)
	rp.WaitUntilComplete(ctx, cs)
}

func upload(ctx context.Context, cs *kubernetes.Clientset, height int64) {
	log.Info().Msg("uploading archive")

	// create upload pod
	rp := &RunPod{
		PodName: config.UploadPodName,
		Image:   config.MinioImage,
		Command: []string{
			"sh",
			"-c",
			fmt.Sprintf(`
			mc config host add minio http://minio:9000 minio minio123;
			mc mb minio/snapshots;
			mc anonymous set download minio/snapshots;
			mc cp /root/%d.tar.gz minio/snapshots/thornode/;
			mc cp /root/genesis.json minio/snapshots/genesis/%d.json;
			mc rm -r --force --older-than 60d minio/snapshots/thornode;
			`+config.PodQuitCommand(), height, height),
		},
		CPU:    "2",
		Memory: "16Gi",
	}
	rp.Run(ctx, cs)
	rp.WaitUntilComplete(ctx, cs)
}

func discordAlert(height int64) {
	if config.DiscordWebhookMainnetInfo == "" {
		return
	}
	log.Info().Msg("sending discord alert")

	// create message
	type DiscordMessage struct {
		Content string `json:"content"`
	}
	msg := DiscordMessage{
		Content: fmt.Sprintf("`[%d]` >> New Pruned Snapshot", height),
	}
	body, err := json.Marshal(msg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to marshal discord message")
	}

	// send message
	resp, err := http.Post(
		strings.TrimSpace(config.DiscordWebhookMainnetInfo),
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil || resp.StatusCode != http.StatusNoContent {
		log.Error().Err(err).Int("status", resp.StatusCode).Msg("failed to send discord message")
	}
}

////////////////////////////////////////////////////////////////////////////////////////
// Main
////////////////////////////////////////////////////////////////////////////////////////

func main() {
	// wait for any sidecars to initialize
	if config.ReadyEndpoint != "" {
		for {
			resp, err := http.Get(config.ReadyEndpoint)
			if err == nil && resp.StatusCode == 200 {
				break
			}
			time.Sleep(1 * time.Second)
		}
		log.Info().Msg("waiting for sidecars to initialize")
	}

	// creates the in-cluster config
	kubeConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create config")
	}

	// creates the clientsets
	client, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create clientset")
	}
	snapshotClient, err := snapclientset.NewForConfig(kubeConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create snapshot clientset")
	}

	ctx := context.Background()
	stopAndReset(ctx, client)
	statesyncRecover(ctx, client)
	volumeSnapshot(ctx, snapshotClient)
	height := exportGenesis(ctx, client)
	createArchive(ctx, client, height)
	upload(ctx, client, height)
	discordAlert(height)

	// kill any sidecars
	if config.QuitEndpoint != "" {
		resp, err := http.Post(config.QuitEndpoint, "", nil)
		if err != nil || resp.StatusCode != 200 {
			log.Err(err).Msg("failed to quit sidecars")
		}
	}
}
