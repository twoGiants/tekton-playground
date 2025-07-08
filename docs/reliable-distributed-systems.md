# Reliable Distributed Systems

## Overview

1. Fault Tolerance  
2. Consistency  
3. Availability  
4. Partition Tolerance  
5. Scalability  
6. Durability  
7. Observability  
8. Self-Healing  
9. Loose Coupling  
10. Idempotency  

## 1. Fault Tolerance

**Summary:**  
A reliable system keeps functioning even when some parts fail. It automatically detects issues and recovers or reroutes as needed, minimizing downtime.

**Cloud/K8s Example:**  
Pods are rescheduled on healthy nodes if a node fails.

**Q:** What happens in Kubernetes if a node crashes?  
**A:** Pods on that node are automatically rescheduled onto healthy nodes.

## 2. Consistency

**Summary:**  
Consistency ensures all users or nodes see the same data at the same time (or eventually). Without it, operations might return stale or conflicting results.

**Cloud/K8s Example:**  
A database in a StatefulSet ensures that all replicas have the same state.

**Q:** How does a distributed database like etcd achieve consistency?  
**A:** It uses consensus protocols (e.g., Raft) to ensure all nodes agree on the state.

## 3. Availability

**Summary:**  
The system is always ready to respond to requests, even if some components are down. High availability means users rarely experience outages.

**Cloud/K8s Example:**  
Service objects route traffic only to healthy pods, keeping the app up.

**Q:** How can an app remain available during a pod update?  
**A:** Use rolling updates and readiness probes to keep traffic flowing to healthy pods.

## 4. Partition Tolerance

**Summary:**  
Even if network problems split the system into isolated parts, it keeps running. The system can handle and recover from communication breakdowns.

**Cloud/K8s Example:**  
Etcd cluster tolerates network splits and still serves reads/writes if quorum exists.

**Q:** What does Kubernetes do if part of the cluster can’t reach the API server?  
**A:** Nodes may be marked NotReady, and workloads are shifted to reachable nodes.

## 5. Scalability

**Summary:**  
A scalable system can handle more traffic or data just by adding resources. It grows easily without a big redesign or manual intervention.

**Cloud/K8s Example:**  
Horizontal Pod Autoscaler increases pod count based on CPU load.

**Q:** How do you scale a web app in Kubernetes?  
**A:** Increase replica count or use HPA to scale automatically.

## 6. Durability

**Summary:**  
Once data is saved, it stays safe—even during crashes or restarts. Durable systems protect data from being lost unexpectedly.

**Cloud/K8s Example:**  
Persistent Volumes (PVs) in Kubernetes ensure data outlives pod restarts.

**Q:** What ensures your data isn’t lost after a pod crash?  
**A:** Persistent Volumes keep data outside of pod lifecycle.

## 7. Observability

**Summary:**  
Reliable systems let you monitor, log, and trace what's happening. This visibility helps to quickly detect, debug, and fix issues.

**Cloud/K8s Example:**  
Metrics via Prometheus, logs via Fluentd, tracing via Jaeger.

**Q:** How do you monitor pod CPU/memory in Kubernetes?  
**A:** Use Prometheus and Grafana dashboards.

## 8. Self-Healing

**Summary:**  
The system automatically recovers from failures, restarting or replacing broken parts. This reduces manual intervention and improves uptime.

**Cloud/K8s Example:**  
Deployments automatically restart failed pods.

**Q:** If a pod fails due to a bug, what happens?  
**A:** Kubernetes restarts the pod automatically.

## 9. Loose Coupling

**Summary:**  
Components interact through clear interfaces, not direct dependencies. Changes in one part don’t break others, making the system more robust.

**Cloud/K8s Example:**  
Microservices communicate via REST/gRPC or message queues.

**Q:** Why use a message queue between services?  
**A:** It decouples sender and receiver, improving reliability and scalability.

## 10. Idempotency

**Summary:**  
Repeating an operation has the same effect as doing it once. This prevents errors or duplication if requests are retried.

**Cloud/K8s Example:**  
Applying the same `kubectl apply` manifest multiple times yields the same resource state.

**Q:** Why should a payment API be idempotent?  
**A:** So repeated requests won’t charge the customer multiple times.
