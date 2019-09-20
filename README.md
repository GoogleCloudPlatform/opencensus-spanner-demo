# Spanner Latency Troubleshooting
This application combines a variety of read and write workloads on Spanner to
simulate the problems that can be encountered in a real-world application at
scale and then demonstrates how to debug them. Running the application for a
few hundred thousand iterations in 'simulation' mode will be sufficient to
surface latency problems. The causes for the latency problems include large
payloads, complex transactions, and queries with full table scans.
Instrumentation for metrics and trace collection using OpenCensus and export to
Stackdriver. The application randomly executes a read and write transaction
using one of a variety of query and transaction strategies on each iteration.
The related paper
[Troubleshooting app latency with Cloud Spanner and OpenCensus](https://cloud.google.com/solutions/troubleshooting-app-latency-with-cloud-spanner-and-opencensus) provides
detailed steps and description of interpretation of results.

After running the application for a period, view the data collected
in the Stackdriver Monitoring and Stackdriver Trace user interfaces to check
the latency of the requests to find the reasons for the differences 
in performance.

The example application assumes that you are familiar with Go programming,
Google Cloud Platform, [Spanner](https://cloud.google.com/spanner/) basics,
[OpenCensus](https://opencensus.io/), and
[Stackdriver](https://cloud.google.com/stackdriver/).

## Prerequisites
The steps described here can be run on a Linux or Mac OS command line or the
GCP Cloud Shell. 

- Select or create a GCP project. Go to the 
  [Project Selector page](https://console.cloud.google.com/projectselector2/home/dashboard)
- Make sure that billing is enabled for your Google Cloud Platform project.
  [Learn how to enable billing](https://cloud.google.com/billing/docs/how-to/modify-project).
- Download and install the
  [Google Cloud SDK](https://cloud.google.com/sdk/docs/).

### Project Setup
In the Cloud Shell, clone the GitHub project

```shell
git clone https://github.com/GoogleCloudPlatform/opencensus-spanner-demo.git
cd opencensus-spanner-demo
```

Edit the variables in setup.env and import them into your development
environment:

```shell
source ./setup.env
```

Enable the Stackdriver and Spanner APIs:

```shell
gcloud services enable stackdriver.googleapis.com \
  cloudtrace.googleapis.com \
  spanner.googleapis.com \
  logging.googleapis.com \
  compute.googleapis.com
```

### Setup Spanner
Create a Spanner instance

```shell
gcloud spanner instances create $SPANNER_INSTANCE \
  --config=regional-us-central1 \
  --description="Test Instance" \
  --nodes=1
```

Create a database

```shell
gcloud spanner databases create $DATABASE --instance=$SPANNER_INSTANCE
```

Create some tables for the test application with the same schema as
[Getting started with Cloud Spanner in Go](https://cloud.google.com/spanner/docs/getting-started/go/).
Following the
[Data Manipulation Language syntax](https://cloud.google.com/spanner/docs/dml-syntax),
in the Cloud Console, navigate to the
[Spanner database](https://console.cloud.google.com/spanner/instances/test-instance/databases/test/createtable).
Check Edit as text, enter the following text into the text area

```sql
CREATE TABLE Singers (
  SingerId   INT64 NOT NULL,
  FirstName  STRING(1024),
  LastName   STRING(1024),
  BirthDate  DATE,
  LastUpdated TIMESTAMP,
) PRIMARY KEY(SingerId);
```

and click the Create button to create the table Singers.

Click on the Create index link and check Edit as text. Enter the following text
into the text area.

```sql
CREATE INDEX SingersByLastName ON Singers(LastName)
```

and click the Create button to add an index for last name.

Go back to Database details and click the Create table link. Check Edit as text
and enter the following text into the text area.

```sql
CREATE TABLE Albums (
  SingerId        INT64 NOT NULL,
  AlbumId         INT64 NOT NULL,
  AlbumTitle      STRING(MAX),
  MarketingBudget INT64,
) PRIMARY KEY(SingerId, AlbumId),
  INTERLEAVE IN PARENT Singers ON DELETE CASCADE;
```

Click Create to create the table Albums.

### Setup an GCE Instance
Back in the Cloud Shell, create a GCE instance to run the test application from

```shell
gcloud compute instances create $CLIENT_INSTANCE \
  --zone=$ZONE \
  --scopes=https://www.googleapis.com/auth/cloud-platform \
  --boot-disk-size=200GB
```

Grant the GCE instance service account the predefined role
[roles/spanner.databaseUser](https://cloud.google.com/spanner/docs/iam#roles)
following these steps. First, find the name of the service
account associated with the instance:

```shell
gcloud compute instances describe $CLIENT_INSTANCE \
 --zone=$ZONE \
 --format="value(serviceAccounts.email)"
```

Make a note of the service account ID to grant role roles/spanner.databaseUser
to the instance service account

```shell
SA_ACCOUNT=[service account id from command above]
gcloud projects add-iam-policy-binding $GOOGLE_CLOUD_PROJECT \
  --member serviceAccount:$SA_ACCOUNT \
  --role roles/spanner.databaseUser
```

SSH to the instance

```shell
gcloud compute ssh --zone $ZONE $CLIENT_INSTANCE
```

Install git

```shell
sudo apt-get update
sudo apt-get install -y git
```

Install [Go](https://golang.org/doc/install) and get the dependent libraries, as
above.

## Run the test app
Clone the code from the git repo.

```shell
git clone https://github.com/GoogleCloudPlatform/opencensus-spanner-demo
cd opencensus-spanner-demo
```

Edit the file setup.env and initialize the environment

```shell
source ./setup.env
```

Build the code

```shell
go build
```

If you have trouble building the application make sure that you have Go modules
enabled by setting the GO111MODULE environment variable:

```shell
export GO111MODULE=on
```

Set the project

```shell
export GOOGLE_CLOUD_PROJECT=[your project]
```

Run the test application:

```shell
nohup ./oc-spannerlab --project=$GOOGLE_CLOUD_PROJECT \
  --instance=$SPANNER_INSTANCE \
  --database=$DATABASE \
  --command=simulation \
  --iterations=100000 &
```
This runs 100,000 iterations of the test application in simulation mode, which
will execute a random combination of queries and updates. It will take several
minutes to run. Check that there are no errors in the command output:

```shell
tail -f nohup.log
```

## View the data
You can view these in the Google Cloud Logging
[Log Viewer](https://console.cloud.google.com/logs/viewer?expandAll=false&resource=gce_instance)
under GCE VM instances. 

Go to the [trace list](https://console.cloud.google.com/traces/traces) to see
the trace data. Notice the payload size in the Trace timeline and how higher
latency tends to be correlated with larger payload size.  To view the payload
size, click on a trace in the Trace list and in the Trace timeline click 
Show events. Notice the bytes received in the timeline. 

To view log-trace correlation, click on a trace in the Trace list and in the
Trace timeline click on Show logs. Notice the log entry in the trace timeline
and in the trace detail.

Also, check the aggregate metrics in
[Stackdriver Monitoring](https://console.cloud.google.com/monitoring).
In the Resource menu click Metrics Explorer. In the Metric textfield type in
the prefix 'spanner-oc-test' and select from the metrics displayed. The
metric 'completed_rpcs' is a good metric to view the overall status of the
test. From the Metrics Explorer click Save chart to save the chart into a
new dashboard.
