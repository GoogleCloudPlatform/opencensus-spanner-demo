# OpenTelemetry Workshop

## Introduction

This code lab demonstrates the
[OpenTelemetry collector](https://github.com/open-telemetry/opentelemetry-collector)
and agent. It includes a test application which sends metrics and trace data to
Prometheus and Zipkin backends via the OpenTelemetry agent and collector. All
software is deployed to the same cluster on Google Kubernetes Engine. The
codelab will step through the deployment, explaining configuration, and viewing
of data.

The advantage of using the OpenTelemetry agent rather than sending trace and
metrics direct to Prometheus, Zipkin, or other backend is that you can change
trace and metrics configuration without changing or redeploying your app.
The advantage of using the OpenTelemetry collector is that it prodives
additional flexibility in configuration options and can scale out to support a
large processing pipeline. A schematic diagram is shown
[here](screenshots/schematic.png).

The code lab is an adaptation of the
[opentelemetry-collector](https://github.com/open-telemetry/opentelemetry-collector/tree/master/examples/demo)
demo.

## Test App
The 
[test app](https://github.com/open-telemetry/opentelemetry-collector/blob/master/examples/main.go)
generates metrics and trace data, which it sends to the
OpenTelemtry agent. The agent then forwards the data to the OpenCensus
collector.

The test application first creates and register metrics and trace exports to the
agent with the code:

```go
	ocAgentAddr, ok := os.LookupEnv("OTEL_AGENT_ENDPOINT")
	if !ok {
		ocAgentAddr = ocagent.DefaultAgentHost + ":" + string(ocagent.DefaultAgentPort)
	}
	oce, err := ocagent.NewExporter(
		ocagent.WithAddress(ocAgentAddr),
		ocagent.WithInsecure(),
		ocagent.WithServiceName(fmt.Sprintf("example-go-%d", os.Getpid())))
	if err != nil {
		log.Fatalf("Failed to create ocagent-exporter: %v", err)
	}
	trace.RegisterExporter(oce)
	view.RegisterExporter(oce)
```

Views for the metrics are defined with the code

```go
	views := []*view.View{
		{
			Name:        "opdemo/latency",
			Description: "The various latencies of the methods",
			Measure:     mLatencyMs,
			Aggregation: view.Distribution(0, 10, 50, 100, 200, 400, 800, 1000, 1400, 2000, 5000, 10000, 15000),
			TagKeys:     []tag.Key{keyClient, keyMethod},
		}
	}
```

Latency and other metrics data is recorded with the code

```go
stats.Record(ctx, mLatencyMs.M(latencyMs))
```

Spans are created for each iteration with the code:

```go
_, span := trace.StartSpan(context.Background(), "Foo")
```

## Project Setup

Enable the following APIs
```shell
gcloud services enable stackdriver.googleapis.com \
 cloudtrace.googleapis.com \
 logging.googleapis.com \
 container.googleapis.com
 ```

## Create a cluser

Create a cluster with the command:

```shell
ZONE=us-central1-c
gcloud container clusters create ot-demo-cluster \
   --num-nodes 2 \
   --machine-type=n1-standard-2 \
   --enable-basic-auth \
   --issue-client-certificate \
   --zone $ZONE
```

See 
[Quickstart: Deploying a language-specific app](https://cloud.google.com/kubernetes-engine/docs/quickstarts/deploying-a-language-specific-app)
for more details on creating clusters.

## Prometheus

Create a ConfigMap volume for the Prometheus configuration file:

```shell
kubectl create configmap prometheus-config-volume --from-file=prometheus.yaml
```

Add the Prometheus workload to the cluster

```shell
kubectl apply -f promdeployment.yaml
```

Check the status

```shell
kubectl get deployments
```

Expose prometheus as a service

```shell
kubectl apply -f promservice.yaml
```

Check the services are running ok

```shell
kubectl get services
NAME         TYPE           CLUSTER-IP     EXTERNAL-IP      PORT(S)          AGE
kubernetes   ClusterIP      10.19.240.1    <none>           443/TCP          91m
prometheus   LoadBalancer   10.19.249.82   35.222.254.152   9090:31355/TCP   57s
```
Note the IP of the LB for Prometheus. It might take a few minutes to create.
Go to http://35.222.254.152:9090 to see the Prometheus console.

## Zipkin

Add the all-in-one Zipkin workload to the cluster

```shell
kubectl apply -f zipkindeployment.yaml
```

Expose Zipkin as a service

```shell
kubectl apply -f zipkinservice.yaml
```

Check the status

```shell
kubectl get svc
NAME         TYPE           CLUSTER-IP      EXTERNAL-IP      PORT(S)          AGE
kubernetes   ClusterIP      10.19.240.1     <none>           443/TCP          141m
kzipkin      LoadBalancer   10.19.243.199   35.238.211.233   9411:31685/TCP   108s
prometheus   LoadBalancer   10.19.249.82    35.222.254.152   9090:31355/TCP   51m
```

Go to the Zipkin console at http://35.238.211.233:9411

## Deploy the OT collector

Create a ConfigMap volume for the OT collector with a configuration file:

```shell
kubectl create configmap otel-collector-config-vol \
  --from-file=otel-collector-config.yaml
```

Deploy the OT collector and a K8s service for it

```shell
kubectl apply -f otel-collector-k8s.yaml
```

## Deploy the OT agent

Create a ConfigMap volume for the OT agent with a configuration file:

```shell
kubectl create configmap otel-agent-config-vol \
  --from-file=otel-agent-config.yaml
```

Deploy the agent as a daemon set:

```shell
kubectl apply -f otel-agent-k8s.yaml
```

## Build and deploy the test app

The example file in the
[opentelemetry-collector/examples/main.go](https://github.com/open-telemetry/opentelemetry-collector/blob/master/examples/main.go)
generates trace and metrics data.

A Docker image needs to be built and pushed to Google Container Registry:

```shell
cd metrics-load-generator
docker build -t metrics-load-generator . 
docker tag metrics-load-generator gcr.io/$PROJECT_ID/metrics-load-generator:v0.0.2
docker push gcr.io/$PROJECT_ID/metrics-load-generator:v0.0.2
```

Edit the file deployment-k8s.yaml, replacing PROJECT_ID with your own
project id. Deploy the metrics generator:

```shell
kubectl apply -f deployment-k8s.yaml
```

## Viewing the data

Find the external IPs of Prometheus and Zipkin from the command

```shell
kubectl get svc
```

Navigate to the Prometheus user interface at http://external_ip:9090 
Click on the metrics dropdown and select a metric in the promexample namespace,
such as promexample_opdemo_latency_count. Enter a
[Prometheus query](https://prometheus.io/docs/prometheus/latest/querying/basics/)
for the timeseries in the textfield. For example,

```
rate(promexample_opdemo_latency_count[1m])
```

You should see a chart similar to [this screenshot](screenshots/prometheus.png).

You can also view percentile values for latency using Prometheus 
[histogram functions](https://prometheus.io/docs/practices/histograms/).
To try this enter the query

```
histogram_quantile(0.9, sum(rate(promexample_opdemo_latency_bucket[2m])) by (le))
```

You should see something like
[this screenshot](screenshots/prometheus_percentile.png).

Explore the other metrics in the Prometheus metric dropdown.

Go to the Zipkin console at http://external_ip:9411
Click on Find Traces. You should see something like 
[this screenshot](screenshots/zipkin.png).

Select one of the traces to view the details.

## Troubleshooting
### Kubernetes Workloads

If you have trouble deploying workloads or service then use the 
[kubectl command](https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands)

```shell
kubectl get pods
```

This will show the name and status of each pod. To view the detailed status and
deployment errors for a pod use the command

```shell
kubectl describe pod [POD NAME]
```

You can also view the workloads in the Google Cloud Platform console, such as
in [this screenshot](screenshots/gcp_workloads.png).

### Prometheus targets

Prometheus collects metrics by scraping targets over HTTP. Two common problems
are (1) the targets are not configured or (2_ the targets are not reachable.
To check the configured targets go to http://external-ip:9090/targets
You should see something like 
[this screenshot](screenshots/prometheus_targets.png) listing the OpenTelemetry
collector as a target.

Check that the target is available in the Kubernetes service for the
OpenTelemetry collector with the command

```shell
kubectl describe service otel-collector
Port:              prom-metrics  8888/TCP
TargetPort:        8888/TCP
Endpoints:         10.16.1.50:8888
Port:              prom-exporter  8889/TCP
TargetPort:        8889/TCP
```

In this configuration, port 8889 provides the app metrics whereas port 8888
provides the metrics for the collector.

### Logs

To view the logs for a pod use the command

```shell
kubectl logs [POD NAME]
```

The container logs can also be viewed in the GCP console. You
can easily find them by first navigating to the workload that you are interested
in and following the link to container logs, such as in
[this screenshot](screenshots/gcp_container_logs.png).

### Volume mounts

Volume mounts are a common problem in deploying Kubernetes workloads. The
Prometheus, OpenTelemetry collector, and agent all require
[volume](https://kubernetes.io/docs/concepts/storage/volumes/) mounts for
configuration files. Use this command to list the
[configmaps](https://kubernetes.io/docs/concepts/storage/volumes/#configmap):

```shell
kubectl get configmap
```

To see the details of the configmap use the command

```shell
kubectl describe configmap [name]
```

The volume mounts themselves are specified in the Kubernetes
[deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/)
files.

### zPages
[zPages](https://opencensus.io/zpages/) may help with Troubleshooting of network
paths within the cluster. To view these, enable Kubernetes port forwarding to
either the agent or collector pod with the command

```shell
kubectl port-forward [POD NAME]  55679:55679
```

The navigate to the pages:
http://localhost:55679/debug/rpcz
and
http://localhost:55679/debug/tracez