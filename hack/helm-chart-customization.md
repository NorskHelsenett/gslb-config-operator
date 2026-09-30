# Helm chart customizations to reapply after `kubebuilder edit --plugins=helm/v2-alpha --force`

`--force` regenerates the entire `charts/chart/templates/` tree from the kustomize
base, silently dropping anything below that has no kustomize equivalent. Reapply
these after every force-regen.

## 1. `charts/chart/values.yaml`

Add under the `manager:` key:

```yaml
  ## One-shot init container that seeds cluster/datacenter info into the GSLB
  ## config map and DNS updater credentials before the manager starts
  ##
  initJob:
    enabled: true
    # Name of the ConfigMap containing config.yaml (required when enabled)
    configMapName: ""
```
## 2. `charts/chart/manager/manager.yaml`
a) `initContainers` inserted immediately before the `containers:` key
```yaml
  {{- if (.Values.manager.initJob.enabled | default true) }}
  initContainers:
  - name: init
    command:
    - /manager
    args:
    - --init
    image: "{{ .Values.manager.image.repository }}{{- if not (contains "@" .Values.manager.image.repository) }}:{{ .Values.manager.image.tag | default .Chart.AppVersion }}{{- end }}"
    {{- with .Values.manager.image.pullPolicy }}
    imagePullPolicy: {{ . }}
    {{- end }}
    env:
    - name: POD_NAMESPACE
      valueFrom:
        fieldRef:
          fieldPath: metadata.namespace
    - name: GSLB_CONFIGMAP_NAME
      value: {{ required "manager.initJob.configMapName is required" .Values.manager.initJob.configMapName | quote }}
    resources:
      {{- if .Values.manager.resources }}
      {{- toYaml .Values.manager.resources | nindent 10 }}
      {{- else }}
      {}
      {{- end }}
    securityContext:
      {{- if .Values.manager.securityContext }}
      {{- toYaml .Values.manager.securityContext | nindent 10 }}
      {{- else }}
      {}
      {{- end }}
    volumeMounts:
    - mountPath: /secrets
      name: dns-updater-creds
  {{- end }}
  containers:
  ```

b) extra mount on the manager container, inside its existing volumeMounts: block

```yaml
    volumeMounts:
      {{- if .Values.manager.extraVolumeMounts }}
      {{- toYaml .Values.manager.extraVolumeMounts | nindent 10 }}
      {{- end }}
      - mountPath: /secrets
        name: dns-updater-creds
    {{- if .Values.certManager.enable }}
    - mountPath: /tmp/k8s-webhook-server/serving-certs
      name: webhook-certs
      readOnly: true
    {{- end }}
```

c) shared volume, inside the pod-level volumes: block

```yaml
    volumes:
      {{- if .Values.manager.extraVolumes }}
      {{- toYaml .Values.manager.extraVolumes | nindent 8 }}
      {{- end }}
    - name: dns-updater-creds
      emptyDir: {}
    {{- if .Values.certManager.enable }}
    - name: webhook-certs
      secret:
        secretName: webhook-server-cert
    {{- end }}
```