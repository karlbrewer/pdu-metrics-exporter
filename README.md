# pdu-metrics-exporter
Prometheus exporter for TRENDnet PDUs


Example Victoria Metrics config

```
scrape_configs:
  - job_name: 'pdu_metrics_exporter'
    metrics_path: /metrics # Endpoint for metrics about the exporter itself
    static_configs:
      - targets:
        - 192.168.1.100:8080  # Exporter IP
  - job_name: 'pdu_metrics'
    metrics_path: /probe # Endpoint for metrics abou the PDUs
    static_configs:
      - targets:
        - 192.168.1.50  # PDU 1 IP
        - 192.168.1.51  # PDU 2 IP
    relabel_configs:
      - source_labels: [__address__]
        target_label: __param_target
      - source_labels: [__param_target]
        target_label: instance
      - target_label: __address__
        replacement: 192.168.1.100:8080  # Exporter IP
```
