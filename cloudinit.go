package main

const cloudInitTemplate = `#cloud-config

write_files:
- path: /etc/kubernetes/kubelet-config.yaml
  permissions: '0644'
  content: |
    apiVersion: kubelet.config.k8s.io/v1beta1
    kind: KubeletConfiguration
    authentication:
      webhook:
        enabled: false
      anonymous:
        enabled: true
    authorization:
      mode: AlwaysAllow
    staticPodPath: /etc/kubernetes/manifests
    readOnlyPort: 10255
    serverTLSBootstrap: false
    clusterDNS: []
    clusterDomain: ""
    cgroupDriver: systemd
    containerRuntimeEndpoint: unix:///run/containerd/containerd.sock
    imageGCHighThresholdPercent: 85
    imageGCLowThresholdPercent: 50
    imageMinimumGCAge: 2m
    healthzBindAddress: 0.0.0.0
    healthzPort: 10248

- path: /etc/systemd/system/k8sless-kubelet.service
  permissions: '0644'
  content: |
    [Unit]
    Description=k8sless kubelet
    After=network-online.target
    Wants=network-online.target
    
    [Service]
    Type=notify
    ExecStartPre=/bin/mkdir -p /etc/kubernetes/manifests
    ExecStartPre=/bin/bash -c 'curl -H "Metadata-Flavor: Google" "http://metadata.google.internal/computeMetadata/v1/instance/attributes/podspec" | /usr/bin/python3 -m json.tool > /etc/kubernetes/manifests/pod.yaml'
    ExecStart=/usr/bin/kubelet \
      --config=/etc/kubernetes/kubelet-config.yaml \
      --hostname-override=%H \
      --pod-infra-container-image=gcr.io/google-containers/pause:3.9 \
      --v=2
    Restart=on-failure
    RestartSec=10
    
    [Install]
    WantedBy=multi-user.target

runcmd:
- systemctl daemon-reload
- systemctl enable k8sless-kubelet.service
- systemctl start k8sless-kubelet.service
# Open firewall for kubelet read-only API
# NOTE: Alternative approaches could include SSH tunneling or a sidecar container
- iptables -A INPUT -p tcp --dport 10255 -j ACCEPT
- iptables -A INPUT -p tcp --dport 10248 -j ACCEPT
`
