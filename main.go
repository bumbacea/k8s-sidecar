package main

import (
	"context"
	"fmt"
	"github.com/bumbacea/k8s-sidecar/handlers"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/strings/slices"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Label               string   `envconfig:"LABEL" required:"true"`
	LabelValue          string   `envconfig:"LABEL_VALUE"`
	Folder              string   `envconfig:"FOLDER" required:"true"`
	FolderAnnotation    string   `envconfig:"FOLDER_ANNOTATION" default:"k8s-sidecar-target-directory"`
	Namespace           []string `envconfig:"NAMESPACE" default:""`
	Resource            string   `envconfig:"RESOURCE" default:"configmap"`
	Req                 ReqConfig
	UniqueFilenames     bool   `envconfig:"UNIQUE_FILENAMES" default:"false"`
	DefaultFileMode     string `envconfig:"DEFAULT_FILE_MODE" default:"0755"`
	Kubeconfig          string `envconfig:"KUBECONFIG"`
	Enable5Xx           string `envconfig:"ENABLE_5XX"`
	WatchServerTimeout  int64  `envconfig:"WATCH_SERVER_TIMEOUT" default:"60"`
	WatchClientTimeout  int64  `envconfig:"WATCH_CLIENT_TIMEOUT" default:"66"`
	MetricsServerPort   uint   `envconfig:"METRICS_SERVER_PORT" default:"8089"`
	EnableMetricsServer bool   `envconfig:"METRICS_SERVER_ENABLED" default:"false"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	sigch := make(chan os.Signal)
	signal.Notify(sigch, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGUSR2, syscall.SIGHUP, os.Interrupt)
	go func() {
		select {
		case sig := <-sigch:
			log.Printf("received signal: %v", sig)
			cancel()
		}
	}()
	config := Config{}
	err := envconfig.Process("", &config)
	if err != nil {
		panic(fmt.Sprintf("failed to prepare config: %s", err))
	}
	log.Printf("config: %+v", config)

	err = os.MkdirAll(config.Folder, os.FileMode(0755))
	if err != nil {
		panic(fmt.Sprintf("unable to create folder: %s", err))
	}
	var clientCfg *rest.Config
	if config.Kubeconfig != "" {
		clientCfg, err = clientcmd.BuildConfigFromFlags("", config.Kubeconfig)
		if err != nil {
			panic(fmt.Sprintf("unable to retrieve kubernetes version: %s", err))
		}
	} else {
		clientCfg, err = rest.InClusterConfig()
	}
	clientCfg.Timeout = time.Second * time.Duration(config.WatchClientTimeout)

	// Get things set up for watching - we need a valid k8s client
	kubeClient, _ := kubernetes.NewForConfig(clientCfg)
	_, err = kubeClient.ServerVersion()
	if err != nil {
		panic(fmt.Sprintf("unable to retrieve kubernetes version: %s", err))
	}

	if config.EnableMetricsServer {
		server, err := startMetricsServer(config.MetricsServerPort)
		if err != nil {
			panic(fmt.Sprintf("failed to start metrics server: %s", err))
		}
		server.Close()
	}

	if slices.Equal(config.Namespace, []string{"ALL"}) {
		config.Namespace = []string{}
	}
	sharedInformerOpts := make([]informers.SharedInformerOption, 0, len(config.Namespace)+1)
	for _, ns := range config.Namespace {
		sharedInformerOpts = append(sharedInformerOpts, informers.WithNamespace(ns))
	}
	sharedInformerOpts = append(sharedInformerOpts, informers.WithTweakListOptions(func(options *v1.ListOptions) {
		if config.Label != "" {
			options.LabelSelector = config.Label
		} else {
			panic("no LABEL env defined")
		}
		if config.LabelValue != "" {
			options.LabelSelector = labels.SelectorFromSet(map[string]string{config.Label: config.LabelValue}).String()
		}
		log.Printf("Watching for labels: %s", options.LabelSelector)
	}))

	globalHandler, err := handlers.NewGenericHandlerImpl(
		config.Folder, func() {
			runCallback(config.Req)
		},
		config.DefaultFileMode,
		config.FolderAnnotation,
		config.UniqueFilenames,
	)
	if err != nil {
		panic(fmt.Sprintf("failed to init handler: %s", err))
	}
	informerFactory := informers.NewSharedInformerFactoryWithOptions(
		kubeClient,
		time.Second*time.Duration(config.WatchServerTimeout),
		sharedInformerOpts...,
	)
	configMapInformer := informerFactory.Core().V1().ConfigMaps().Informer()
	if config.Resource == "configmap" || config.Resource == "both" {
		_, err = configMapInformer.AddEventHandler(handlers.NewCmHandler(globalHandler))
		if err != nil {
			panic(fmt.Sprintf("unable to add event handler: %s", err))
		}
	}
	if config.Resource == "secrets" || config.Resource == "both" {
		secretsInformer := informerFactory.Core().V1().Secrets().Informer()
		_, err = secretsInformer.AddEventHandler(handlers.NewSecretsHandler(globalHandler))
		if err != nil {
			panic(fmt.Sprintf("unable to add event handler: %s", err))
		}
	}
	informerFactory.Start(ctx.Done())
	informerFactory.WaitForCacheSync(ctx.Done())

	select {
	case <-ctx.Done():
	}
	informerFactory.Shutdown()
	log.Printf("waited to all threads to end")
}
