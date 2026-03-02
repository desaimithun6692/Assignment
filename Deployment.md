# Infrastructure Setup Guide

### A. Database (Cloud SQL)
*   **Provision Instance:** Create a [PostgreSQL 15+](https://cloud.google.com) instance for production stability.
*   **Connectivity:** Enable the [Cloud SQL Auth Proxy](https://cloud.google.com) on your Cloud Run service to ensure secure, private connectivity without public IP exposure.

### B. Messaging (Pub/Sub)
*   **Topic:** Create `events-topic` as the central message bus.
*   **Push Subscription:** Configure the [Push endpoint](https://cloud.google.com) to point to your Ingestion Function URL. Use [OIDC tokens](https://cloud.google.com) for identity-based secure delivery.

### C. Artifact Registry
*   **Create Repository:** Initialize a [Docker repository](https://cloud.google.com) in Artifact Registry to store service images.
*   **Build & Push:** Use `gcloud builds submit` to containerize your code and push it to the registry for deployment.

### D. Ingestion Function
*   **Deploy:** Launch a [2nd Gen Cloud Function](https://cloud.google.com) with an HTTP trigger.
*   **Integration:** Map the [Pub/Sub Push subscription](https://cloud.google.com) to this function, ensuring the service account has the **Cloud Run Invoker** role.

### E. Secrets Management
*   **Create Secret:** Store your database URL in [Secret Manager](https://cloud.google.com) using `gcloud 
secrets create db-url`.
*   **Access Control:** Grant your function’s service account the [Secret Accessor](https://cloud.google.com) role to pull the credentials at runtime.


sample cloud build yaml

steps:
  # --- 1. BUILD & PUSH EVENT SERVICE (Cloud Run) ---
  - name: 'gcr.io/cloud-builders/docker'
    args: ['build', '-t', '$_AR_HOSTNAME/$PROJECT_ID/$_AR_REPO/event-api:$SHORT_SHA', './Event_service']

  - name: 'gcr.io/cloud-builders/docker'
    args: ['push', '$_AR_HOSTNAME/$PROJECT_ID/$_AR_REPO/event-api:$SHORT_SHA']

  # --- 2. DEPLOY EVENT SERVICE ---
  - name: 'gcr.io/://google.com'
    entrypoint: gcloud
    args:
      - 'run'
      - 'deploy'
      - 'event-api'
      - '--image=$_AR_HOSTNAME/$PROJECT_ID/$_AR_REPO/event-api:$SHORT_SHA'
      - '--region=$_REGION'
      - '--service-account=$_API_SA'
      - '--set-secrets=DATABASE_URL=DB_URL:latest'

  # --- 3. DEPLOY INGESTION FUNCTION ---
  - name: 'gcr.io/://google.com'
    entrypoint: gcloud
    args:
      - 'functions'
      - 'deploy'
      - 'event-ingestor'
      - '--gen2'
      - '--runtime=go122'
      - '--region=$_REGION'
      - '--source=./ingestion_function'
      - '--entry-point=ProcessMessage'
      - '--trigger-topic=events-topic'
      - '--service-account=$_INGEST_SA'
      - '--set-secrets=DATABASE_URL=DB_URL:latest'

substitutions:
  _REGION: us-central1
  _AR_HOSTNAME: us-central1-docker.pkg.dev
  _AR_REPO: Assignment
  _API_SA: event-app-sa@${PROJECT_ID}.iam.gserviceaccount.com
  _INGEST_SA: ingestion-func-sa@${PROJECT_ID}.iam.gserviceaccount.com
  _DB_CONN: ${PROJECT_ID}:us-central1:my-instance-id