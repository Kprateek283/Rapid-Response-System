# Rapid Crisis Response — Cloud Run Deployment
# ═══════════════════════════════════════════════

# 1. Build the Docker image
#   docker build -t rapid-api .

# 2. Tag for GCR
#   docker tag rapid-api gcr.io/$GCS_PROJECT_ID/rapid-api

# 3. Push to GCR
#   docker push gcr.io/$GCS_PROJECT_ID/rapid-api

# 4. Deploy to Cloud Run
#   gcloud run deploy rapid-api \
#     --image gcr.io/$GCS_PROJECT_ID/rapid-api \
#     --platform managed \
#     --region us-central1 \
#     --allow-unauthenticated \
#     --add-cloudsql-instances=$CLOUD_SQL_CONNECTION_NAME \
#     --set-env-vars="DATABASE_URL=$DATABASE_URL" \
#     --set-env-vars="REDIS_URL=$REDIS_URL" \
#     --set-env-vars="RABBITMQ_URL=$RABBITMQ_URL" \
#     --set-env-vars="JWT_SECRET=$JWT_SECRET" \
#     --set-env-vars="GCS_BUCKET_NAME=$GCS_BUCKET_NAME" \
#     --set-env-vars="GCS_CREDENTIALS_JSON=$GCS_CREDENTIALS_JSON" \
#     --set-env-vars="FIREBASE_CREDENTIALS_JSON=$FIREBASE_CREDENTIALS_JSON" \
#     --set-env-vars="GEMINI_API_KEY=$GEMINI_API_KEY" \
#     --min-instances=1 \
#     --max-instances=10 \
#     --memory=512Mi \
#     --cpu=1 \
#     --timeout=300 \
#     --concurrency=80

# Note: WebSocket connections require Cloud Run to allow HTTP/2 and
# the --timeout flag sets the maximum request duration (5 min for WS).
