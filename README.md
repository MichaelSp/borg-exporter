# borg-exporter

A Prometheus exporter for [Borg](https://github.com/borgbackup/borg) backups. Based on python version
of [borg-exporter](https://github.com/nupplaphil/borg-exporter) now in Golang, optimized to run as sidecar to borgmatic
in k8s.

It provides the following metrics:

| Name                                | Description                                       | Type  |
| ----------------------------------- | ------------------------------------------------- | ----- |
| borg_backups_total                  | Total number of Borg backups                      | Gauge |
| borg_last_backup_timestamp          | Timestamp of the last backup                      | Gauge |
| borg_unique_size                    | Uncompressed unique size of the Borg Repo (bytes) | Gauge |
| borg_number_of_files                | Number of files in the Borg Repo                  | Gauge |
| borg_error_fetching_repository_info | Error fetching repository info                    | Gauge |
| borg_deduplicated_size              | Deduplicated size of the Borg Repo (bytes)        | Gauge |
| borg_compressed_size                | Compressed size of the Borg Repo (bytes)          | Gauge |
| borg_cache_size                     | Size of the Borg cache (bytes)                    | Gauge |

## Configuration

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backup-sender
  labels:
    app: backup-sender
spec:
  selector:
    matchLabels:
      app: backup-sender
  template:
    metadata:
      labels:
        app: backup-sender
    spec:
      containers:
        - name: backup-sender
          image: ghcr.io/borgmatic-collective/borgmatic:1.9.4
          env:
            - name: BORGMATIC_CONFIG
              value: "/etc/borgmatic.d"
            - name: BORG_REPO
              value: "ssh://backup:2234/dst"
            - name: BORG_PASSCOMMAND
              value: "cat /root/.config/borg/passphrase"
            - name: BORG_SECURITY_DIR
              value: "/var/borg/security"
            - name: RUN_ON_STARTUP
              value: "false"
            - name: K8S_NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
          volumeMounts:
            - name: borgmatic-config
              mountPath: /etc/borgmatic/
            - name: init
              mountPath: /etc/borgmatic.d
            - name: ssh-secrets
              mountPath: /root/.ssh
            - name: borg-passphrase
              mountPath: /root/.config/borg
            - name: borg-security
              mountPath: /var/borg/security
              subPath: security
            - name: borg-security
              mountPath: /root/.cache/borg
              subPath: cache
        - name: borg-exporter
          image: ghcr.io/michaelsp/borg-exporter:1.0.0
          imagePullPolicy: Always
          env:
            - name: BORGMATIC_CONFIG
              value: "/etc/borgmatic.d"
            - name: BORG_REPO
              value: "ssh://backup:2234/dst"
            - name: BORG_PASSCOMMAND
              value: "cat /root/.config/borg/passphrase"
            - name: BORG_SECURITY_DIR
              value: "/var/borg/security"
          volumeMounts:
            - name: borgmatic-config
              mountPath: /etc/borgmatic/
            - name: init
              mountPath: /etc/borgmatic.d
            - name: ssh-secrets
              mountPath: /root/.ssh
            - name: borg-passphrase
              mountPath: /root/.config/borg
            - name: borg-security
              mountPath: /var/borg/security
              subPath: security
            - name: borg-security
              mountPath: /root/.cache/borg
              subPath: cache
          ports:
            - containerPort: 9996
      volumes:
        - name: init
          emptyDir: {}
        - name: borgmatic-config
          configMap:
            name: borgmatic-config
        - name: hostpath
          hostPath:
            path: /var/lib/rancher/k3s/storage/
        - name: ssh-config
          configMap:
            name: ssh-config
        - name: ssh-secrets
          secret:
            secretName: backup-ssh-keys-source
            defaultMode: 0600
        - name: borg-passphrase
          secret:
            secretName: borg-repo-passphrase-secrets
            defaultMode: 0600
        - name: borg-security
          persistentVolumeClaim:
            claimName: borg-security
---
apiVersion: v1
kind: Service
metadata:
  name: backup-sender
spec:
  selector:
    app: backup-sender
  ports:
    - name: metrics
      port: 9996
      targetPort: 9996
      protocol: TCP
---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: backup-sender
spec:
  selector:
    matchLabels:
      app: backup-sender
  endpoints:
    - port: metrics
      interval: 5m
      scrapeTimeout: 90s
      path: /metrics
```

## Alerting rules

Alerting rules can be found [here](./borg-mixin/prometheus-alerts.yaml). By
default, Prometheus sends an alert if a backup hasn't been issued in 24h5m.

## Grafana Dashboard

You can find the generated Grafana dashboard [here](./borg-mixin/dashboards_out/dashboard.json) and it can be imported
directly into the Grafana UI.

It's also available in [Grafana's Dashboard Library](https://grafana.com/grafana/dashboards/14489).
