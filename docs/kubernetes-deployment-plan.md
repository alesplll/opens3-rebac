# Kubernetes deployment — план развёртывания в облаке

Дата: 2026-04-06
Связанные документы: [Storage Service — план реализации](storage-service-implementation-plan.md) | [Распределённое хранение](distributed-storage-architecture.md)

---

## Контекст

Docker Compose годится для локальной разработки, но не даёт настоящего distributed-опыта: все контейнеры на одной машине, нет реальных network partition, нет node failure. Kubernetes на 5 нодах в облаке даёт всё это.

---

## Целевая топология: 5 нод

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Kubernetes Cluster                           │
│                                                                     │
│  Node-1 (app)              Node-2 (app)            Node-3 (app)    │
│  t3.xlarge                 t3.xlarge                t3.xlarge       │
│  4 vCPU / 16 GB            4 vCPU / 16 GB           4 vCPU / 16 GB │
│  ┌──────────────┐          ┌──────────────┐         ┌─────────────┐│
│  │ Gateway ×1   │          │ Gateway ×1   │         │ Gateway ×1  ││
│  │ Placement ×1 │          │ AuthZ ×1     │         │ AuthZ ×1    ││
│  │ Auth ×1      │          │ Metadata ×1  │         │ Metadata ×1 ││
│  │ Users ×1     │          │ Auth ×1      │         │ Users ×1    ││
│  │ Storage ×1   │          │ Storage ×1   │         │ Storage ×1  ││
│  │ PG-users(P)  │          │ PG-users(R)  │         │ PG-meta(P)  ││
│  │ Redis-master │          │ Redis-replica│         │ PG-meta(R)  ││
│  └──────────────┘          └──────────────┘         └─────────────┘│
│                                                                     │
│  Node-4 (infra)            Node-5 (infra)                          │
│  t3.xlarge                 t3.large                                │
│  4 vCPU / 16 GB            2 vCPU / 8 GB                           │
│  ┌──────────────┐          ┌──────────────┐                        │
│  │ Kafka ×1     │          │ Kafka ×1     │                        │
│  │ Neo4j ×1     │          │ Neo4j ×2     │                        │
│  │ Kafka ×1     │          │ Prometheus   │                        │
│  │ Placement ×1 │          │ Grafana      │                        │
│  │ Redis-sent.  │          │ Jaeger       │                        │
│  └──────────────┘          └──────────────┘                        │
│                                                                     │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │ Ingress Controller (nginx) → Service: gateway (ClusterIP)    │  │
│  │ Managed LB (cloud) → Ingress → Gateway pods                 │  │
│  └──────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Kubernetes-примитивы для каждого компонента

| Компонент | K8s ресурс | Replicas | Почему именно этот ресурс |
|---|---|---|---|
| Gateway | **Deployment** | 3 | Stateless, любой pod взаимозаменяем |
| Placement | **Deployment** | 2 | Stateless (ring в ConfigMap) |
| AuthZ | **Deployment** | 2 | Stateless (состояние в Neo4j/Redis) |
| Metadata | **Deployment** | 2 | Stateless (состояние в PostgreSQL) |
| Auth | **Deployment** | 2 | Stateless |
| Users | **Deployment** | 2 | Stateless |
| Storage | **StatefulSet** | 3 | **Stateful**: каждый pod имеет свой PersistentVolume. Нельзя заменять произвольно — данные привязаны к конкретному pod |
| PostgreSQL | **StatefulSet** | 2 (per DB) | Stateful + stable network identity для репликации |
| Neo4j | **StatefulSet** | 3 | Stateful + кластерный протокол требует stable DNS |
| Redis | **StatefulSet** | 3 | Sentinel-режим требует stable identity |
| Kafka | **StatefulSet** | 3 | Брокер = stable identity + persistent log |
| Prometheus | **StatefulSet** | 1 | Persistent storage для метрик |
| Grafana | **Deployment** | 1 | Stateless (дашборды из ConfigMap/git) |
| Jaeger | **Deployment** | 1 | Можно терять трейсы без последствий |

**Итого: ~36 pod, 20 наших + 16 инфраструктурных**

---

## Storage как StatefulSet — ключевой момент

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: storage
spec:
  serviceName: storage  # → storage-0.storage, storage-1.storage, storage-2.storage
  replicas: 3
  template:
    spec:
      containers:
        - name: storage
          image: opens3/storage:latest
          env:
            - name: NODE_ID
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name  # "storage-0", "storage-1", "storage-2"
            - name: GRPC_PORT
              value: "50053"
          ports:
            - containerPort: 50053
          volumeMounts:
            - name: blob-data
              mountPath: /data
          readinessProbe:
            grpc:
              port: 50053
            periodSeconds: 5
          livenessProbe:
            grpc:
              port: 50053
            periodSeconds: 10
  volumeClaimTemplates:
    - metadata:
        name: blob-data
      spec:
        accessModes: ["ReadWriteOnce"]
        storageClassName: gp3  # AWS EBS gp3
        resources:
          requests:
            storage: 100Gi
```

Каждый pod получает:
- **Stable DNS**: `storage-0.storage.default.svc.cluster.local`
- **Свой PersistentVolume**: даже при перезапуске pod — данные сохраняются
- **Ordered startup/shutdown**: storage-0 → storage-1 → storage-2

Placement знает о нодах через Headless Service:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: storage
spec:
  clusterIP: None  # Headless → DNS резолвит в IP всех pod
  selector:
    app: storage
  ports:
    - port: 50053
```

---

## Networking: Service mesh vs простые Service

### Вариант A: Kubernetes Services (проще)

```yaml
apiVersion: v1
kind: Service
metadata:
  name: placement
spec:
  selector:
    app: placement
  ports:
    - port: 50060
```

| | |
|---|---|
| **Плюсы** | Ничего лишнего, всё из коробки |
| **Минусы** | Нет per-request load balancing для gRPC (DNS-based = per-connection) |

### Вариант B: Headless Service + gRPC client-side LB

```yaml
apiVersion: v1
kind: Service
metadata:
  name: placement-headless
spec:
  clusterIP: None
  selector:
    app: placement

# В Gateway:
# grpc.Dial("dns:///placement-headless:50060",
#   grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`))
```

gRPC сам резолвит DNS → получает все IP pod → round-robin на каждый запрос.

| | |
|---|---|
| **Плюсы** | Per-request балансировка. Нет доп. инфраструктуры |
| **Минусы** | DNS TTL (kube-dns кэширует 30s). Новый pod подхватывается не мгновенно |

### Вариант C: Istio service mesh (продакшен-уровень)

```yaml
apiVersion: networking.istio.io/v1
kind: DestinationRule
metadata:
  name: placement
spec:
  host: placement
  trafficPolicy:
    loadBalancer:
      simple: ROUND_ROBIN
    connectionPool:
      http:
        h2UpgradePolicy: UPGRADE  # HTTP/2 для gRPC
```

Envoy sidecar в каждом pod → прозрачный per-request LB, mTLS, retry, circuit breaker, distributed tracing.

| | |
|---|---|
| **Плюсы** | Всё из коробки: LB, mTLS, observability, traffic shaping, canary deploys |
| **Минусы** | +50-100 MB RAM на каждый pod (sidecar). Сложность отладки. Кривая обучения |

**Рекомендация:** начать с **Вариант B** (headless + client-side LB), перейти на Istio когда захотите mTLS и canary.

---

## CI/CD pipeline

```
┌─────────────┐     ┌──────────────┐     ┌──────────────┐     ┌───────────┐
│ git push    │────►│ GitHub       │────►│ Container    │────►│ Kubernetes│
│ (main/tag)  │     │ Actions      │     │ Registry     │     │ Cluster   │
└─────────────┘     │              │     │ (ECR/GHCR)   │     │           │
                    │ • go test    │     │              │     │ ArgoCD    │
                    │ • go build   │     │ image:tag    │     │ или       │
                    │ • docker     │     │              │     │ kubectl   │
                    │   build+push │     │              │     │ apply     │
                    └──────────────┘     └──────────────┘     └───────────┘
```

```yaml
# .github/workflows/deploy.yml
name: Build & Deploy

on:
  push:
    tags: ['v*']

jobs:
  build:
    strategy:
      matrix:
        service: [gateway, placement, storage, auth, users, authz, metadata]
    steps:
      - uses: actions/checkout@v4
      - uses: docker/build-push-action@v5
        with:
          context: .
          file: services/${{ matrix.service }}/Dockerfile
          push: true
          tags: ghcr.io/alesplll/opens3-${{ matrix.service }}:${{ github.ref_name }}

  deploy:
    needs: build
    steps:
      - uses: azure/k8s-set-context@v4  # или aws-actions/configure-aws-credentials
      - run: |
          helm upgrade --install opens3 ./k8s/helm/opens3 \
            --set image.tag=${{ github.ref_name }}
```

---

## Observability

```yaml
# ServiceMonitor для Prometheus (если установлен kube-prometheus-stack)
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: opens3-services
spec:
  selector:
    matchLabels:
      monitoring: opens3
  endpoints:
    - port: metrics
      interval: 15s
```

Все Go-сервисы уже экспортируют OTEL-метрики → Prometheus scrapes → Grafana dashboards.

Jaeger собирает трейсы → можно увидеть полный путь запроса:
```
Client → Gateway (12ms)
  → Placement.Allocate (0.3ms)
  → Storage-0.StoreObject (145ms)
  → Storage-2.StoreObject (152ms)
  → Metadata.CreateObjectVersion (3ms)
```

---

## Chaos-тестирование

Главная ценность Kubernetes — можно **ломать вещи** и смотреть что происходит:

```bash
# Убить pod storage-1 → реплики на storage-0 и storage-2 доступны
kubectl delete pod storage-1

# Убить целую ноду
kubectl drain node-2 --delete-emptydir-data --force

# Network partition: изолировать storage-2
kubectl exec storage-2 -- iptables -A INPUT -s gateway -j DROP

# Или через chaos-mesh (если установлен)
kubectl apply -f - <<EOF
apiVersion: chaos-mesh.org/v1alpha1
kind: PodChaos
metadata:
  name: kill-storage
spec:
  action: pod-kill
  mode: one
  selector:
    labelSelectors:
      app: storage
  scheduler:
    cron: '@every 5m'  # убивать рандомный storage pod каждые 5 минут
EOF
```

Что проверять:
- **Pod killed** → K8s рестартит pod → PV сохранён → данные на месте
- **Node drained** → pod перенесён на другую ноду → PV reattach (для EBS может занять ~1 мин)
- **Network partition** → Gateway failover на другую реплику → Placement запускает дорепликацию
- **Kafka broker killed** → остальные 2 broker работают → consumer переключается

---

## Стоимость

| Ресурс | Тип | Кол-во | Цена/мес (us-east-1) |
|---|---|---|---|
| App-ноды | t3.xlarge (4 vCPU, 16 GB) | 3 | 3 × $120 = **$360** |
| Infra-нода | t3.xlarge | 1 | **$120** |
| Monitoring-нода | t3.large (2 vCPU, 8 GB) | 1 | **$60** |
| EBS gp3 (Storage PV) | 100 GB × 3 | 3 | 3 × $8 = **$24** |
| EBS gp3 (PG, Neo4j, Kafka) | 50 GB × 10 | 10 | 10 × $4 = **$40** |
| ALB | managed | 1 | **$16** + traffic |
| **Итого** | | | **~$620/мес** |

**Как сэкономить:**
- **Spot instances** для app-нод (Gateway, Placement, Auth, Users) → -60-70%. Stateless сервисы переживают spot interruption
- **Reserved instances** для infra-нод (1 year) → -30-40%
- **Выключать на ночь** (если не продакшен) → `/2`
- **Yandex Cloud / VK Cloud** — дешевле для РФ

Реалистичная цена со spot + выключением на ночь: **~$150-200/мес**.

---

## Структура Helm-чарта

```
k8s/
├── helm/
│   └── opens3/
│       ├── Chart.yaml
│       ├── values.yaml              # Все настройки в одном месте
│       ├── values-dev.yaml          # Оверрайды для dev (1 реплика всего)
│       ├── values-prod.yaml         # Оверрайды для prod (полная топология)
│       └── templates/
│           ├── _helpers.tpl
│           ├── gateway-deployment.yaml
│           ├── gateway-service.yaml
│           ├── gateway-ingress.yaml
│           ├── placement-deployment.yaml
│           ├── placement-service.yaml
│           ├── storage-statefulset.yaml
│           ├── storage-service-headless.yaml
│           ├── authz-deployment.yaml
│           ├── metadata-deployment.yaml
│           ├── auth-deployment.yaml
│           ├── users-deployment.yaml
│           ├── postgres-users-statefulset.yaml
│           ├── postgres-metadata-statefulset.yaml
│           ├── neo4j-statefulset.yaml
│           ├── redis-statefulset.yaml
│           ├── kafka-statefulset.yaml
│           ├── prometheus-statefulset.yaml
│           ├── grafana-deployment.yaml
│           ├── jaeger-deployment.yaml
│           └── configmaps.yaml
```

Или вместо кастомного Helm — использовать готовые чарты для инфраструктуры:
- **Bitnami PostgreSQL HA** (Patroni из коробки)
- **Bitnami Kafka** (KRaft, без ZooKeeper)
- **Bitnami Redis** (Sentinel)
- **Neo4j Helm chart** (официальный)
- **kube-prometheus-stack** (Prometheus + Grafana + AlertManager)

Свои чарты — только для 7 наших сервисов.

---

## Поэтапная реализация

```
Фаза 6a: Контейнеризация
  │
  ├── Dockerfile для каждого сервиса (storage уже есть)
  ├── GitHub Actions: build + push images в GHCR
  └── Проверить что все образы запускаются standalone
  │
Фаза 6b: Kubernetes manifests
  │
  ├── Helm-чарт для наших 7 сервисов
  ├── Bitnami-чарты для PG, Kafka, Redis, Neo4j
  ├── values-dev.yaml (всё по 1 реплике, для быстрого тестирования)
  └── values-prod.yaml (полная топология из плана)
  │
Фаза 6c: Развёртывание кластера
  │
  ├── Terraform/Pulumi для создания кластера (EKS / GKE / YC Managed K8s)
  ├── helm install opens3 + инфраструктурные чарты
  ├── Ingress + TLS (cert-manager + Let's Encrypt)
  └── Smoke test: PutObject → GetObject через публичный endpoint
  │
Фаза 6d: CI/CD + chaos
  │
  ├── GitHub Actions: tag → build → push → helm upgrade
  ├── Chaos-mesh: рандомный pod-kill каждые 5 минут
  ├── Grafana dashboard: latency, error rate, replication lag
  └── Runbook: что делать при алертах
```

---

## Чему научит

| Навык | Где применяется |
|---|---|
| StatefulSet vs Deployment | Когда данные привязаны к pod |
| PersistentVolume lifecycle | EBS attach/detach при reschedule |
| Headless Service + gRPC LB | Service discovery для gRPC |
| Helm templating | Один чарт → dev/staging/prod |
| Horizontal Pod Autoscaler | Автоскейл Gateway по CPU/RPS |
| Pod Disruption Budget | Гарантия: не убивать >1 storage pod одновременно |
| Network Policy | Изоляция: Storage принимает только от Gateway |
| Resource limits/requests | Планирование размещения pod по нодам |
| Liveness/readiness probes | gRPC health check → K8s знает когда pod готов |
| Rolling update | Zero-downtime deploy |
| Chaos engineering | "Всё, что может сломаться — сломается" |
