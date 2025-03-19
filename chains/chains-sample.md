# Chains Sample

Cluster with Tekton resources must be running. See [Deploy Cluster](../README.md#deploy-cluster).

Try Supply Chain Security. Install Tekton Chains.

```sh
# install
kubectl apply --filename \
https://storage.googleapis.com/tekton-releases/chains/latest/release.yaml

# monitor
kubectl get po -n tekton-chains -w
```

Configure Tekton Chains to store the provenance metadata locally.

```sh
kubectl patch configmap chains-config -n tekton-chains \
-p='{"data":{"artifacts.oci.storage": "", "artifacts.taskrun.format":"in-toto", "artifacts.taskrun.storage": "tekton"}}'
```

Generate a key pair to sign the artifact provenance. You are prompted to enter a password for the private key. For this guide, leave the password empty and press Enter twice.

```sh
cosign generate-key-pair k8s://tekton-chains/signing-secrets
```

*Optional: Restart the controller to ensure it picks up the changes.*

```sh
kubectl delete po -n tekton-chains -l app=tekton-chains-controller
```

Deploy a demo pipeline to your cluster.

```sh
kubectl apply -k chains/pipeline
```

Run and monitor the demo pipeline.

```sh
# run
kubectl create -f chains/runs/build-push-pipeline-run.yaml

# monitor
tkn pr logs build-push-run-... -f
```

Get the metadata.

```sh
export PR_UID=$(tkn pr describe --last -o  jsonpath='{.metadata.uid}')
tkn pr describe --last \
-o jsonpath="{.metadata.annotations.chains\.tekton\.dev/signature-pipelinerun-$PR_UID}" \
| base64 -d > metadata.json
```

View the provenance.

```sh
cat metadata.json | jq -r '.payload' | base64 -d | jq .
```

Verify that the metadata hasn’t been tampered with.

```sh
cosign verify-blob-attestation --insecure-ignore-tlog \
--key k8s://tekton-chains/signing-secrets --signature metadata.json \
--type slsaprovenance --check-claims=false /dev/null
```
