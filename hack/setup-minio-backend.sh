#!/usr/bin/env bash
# Copyright 2020 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# setup-minio-backend.sh - Deploy a single-replica MinIO into the current
# kubectl context and emit a Chainsaw-compatible S3 credentials values file.
#
# Owned by the sample driver for the same reason as its peer
# setup-s3-backend.sh: this driver is the component that interfaces directly
# with the S3 backend, so the backend tooling lives here. The two scripts are
# interchangeable from the caller's point of view - both emit the same six-key
# values file that cosi's hack/setup-sample-driver.sh and its Chainsaw e2e
# suite consume.
#
# Scope: a local-development and laptop-friendly alternative to
# setup-s3-backend.sh, which needs raw disks for Rook/Ceph OSDs and host-level
# privileges (losetup, mount --make-rshared) that only the Prow CI node has.
# MinIO needs none of that: it is a single Deployment with emptyDir storage on
# a plain kind cluster. Nothing here is durable - the bucket data lives in the
# pod's emptyDir and is lost when the pod restarts, which is acceptable for
# e2e but makes this unsuitable for anything else.
#
# The sample driver exercises only the plain S3 API (CreateBucket, HeadBucket,
# DeleteBucket, and object Put/Get/Delete), so MinIO's root credentials are
# sufficient and no MinIO admin API or `mc` client is needed. As in
# setup-s3-backend.sh, the admin and access key pairs emitted below are the
# same credentials: the sample driver's DriverGrantBucketAccess does not mint
# per-bucket users, it hands back the credentials it is given.
#
# Output: a YAML values file with the S3 credentials, written to
# ${OUT_CREDS_FILE:-./s3-credentials.yaml}.

set -o errexit
set -o nounset
set -o pipefail

MINIO_NS="${MINIO_NS:-minio}"
MINIO_IMAGE="${MINIO_IMAGE:-quay.io/minio/minio:RELEASE.2025-04-22T22-12-26Z}"

# Root credentials. MinIO refuses to start if either is shorter than 3 (user)
# or 8 (password) characters. These are e2e-only throwaways.
MINIO_ROOT_USER="${MINIO_ROOT_USER:-cosiadmin}"
MINIO_ROOT_PASSWORD="${MINIO_ROOT_PASSWORD:-cosiadmin123}"

MINIO_PORT=9000

OUT_CREDS_FILE="${OUT_CREDS_FILE:-${PWD}/s3-credentials.yaml}"

echo "==> Deploying MinIO (${MINIO_IMAGE}) into namespace ${MINIO_NS}"

kubectl create namespace "${MINIO_NS}" --dry-run=client -o yaml | kubectl apply -f -

kubectl -n "${MINIO_NS}" create secret generic minio-root-credentials \
  --from-literal=MINIO_ROOT_USER="${MINIO_ROOT_USER}" \
  --from-literal=MINIO_ROOT_PASSWORD="${MINIO_ROOT_PASSWORD}" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: minio
  namespace: ${MINIO_NS}
  labels:
    app.kubernetes.io/name: minio
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: minio
  # emptyDir storage means a rolling update would serve from an empty volume
  # while the old pod still holds the data, so replace rather than overlap.
  strategy:
    type: Recreate
  template:
    metadata:
      labels:
        app.kubernetes.io/name: minio
    spec:
      containers:
      - name: minio
        image: ${MINIO_IMAGE}
        imagePullPolicy: IfNotPresent
        args:
        - server
        - /data
        - --console-address
        - ":9001"
        envFrom:
        - secretRef:
            name: minio-root-credentials
        ports:
        - name: s3
          containerPort: ${MINIO_PORT}
        - name: console
          containerPort: 9001
        readinessProbe:
          httpGet:
            path: /minio/health/ready
            port: s3
          initialDelaySeconds: 5
          periodSeconds: 5
        livenessProbe:
          httpGet:
            path: /minio/health/live
            port: s3
          initialDelaySeconds: 10
          periodSeconds: 10
        volumeMounts:
        - name: data
          mountPath: /data
      volumes:
      - name: data
        emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: minio
  namespace: ${MINIO_NS}
  labels:
    app.kubernetes.io/name: minio
spec:
  type: ClusterIP
  selector:
    app.kubernetes.io/name: minio
  ports:
  - name: s3
    port: ${MINIO_PORT}
    targetPort: s3
  - name: console
    port: 9001
    targetPort: console
EOF

echo "==> Waiting for MinIO to be Available (up to 5 min)"
kubectl -n "${MINIO_NS}" rollout status deployment/minio --timeout=300s

# ---------------------------------------------------------------------------
# Write the values file.
# ---------------------------------------------------------------------------
ENDPOINT="http://minio.${MINIO_NS}.svc.cluster.local:${MINIO_PORT}"

mkdir -p "$(dirname "${OUT_CREDS_FILE}")"
cat > "${OUT_CREDS_FILE}" <<EOF
# Generated by hack/setup-minio-backend.sh - DO NOT EDIT or commit.
# Re-run hack/setup-minio-backend.sh to refresh after cluster recreation.
s3Endpoint: "${ENDPOINT}"
# MinIO accepts any region; AWS SDK v2 rejects empty strings, so use a placeholder.
s3Region: "us-east-1"
# The sample driver's DriverGrantBucketAccess hands back the credentials it is
# given rather than minting per-bucket users, so admin and access are the same
# key pair - matching what setup-s3-backend.sh emits for Rook.
adminAccessKeyId: "${MINIO_ROOT_USER}"
adminSecretAccessKey: "${MINIO_ROOT_PASSWORD}"
accessKeyId: "${MINIO_ROOT_USER}"
accessSecretKey: "${MINIO_ROOT_PASSWORD}"
EOF

echo "==> MinIO setup complete."
echo "    Credentials written to ${OUT_CREDS_FILE}"
echo "    Endpoint : ${ENDPOINT}"
echo "    AccessKey: ${MINIO_ROOT_USER}"
