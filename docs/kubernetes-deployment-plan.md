# Kubernetes deployment: проект

- Дата: 2026-04-06
- Актуализировано: 2026-09-11
Статус: **план, не готовый manifest**.

В репозитории пока нет Helm chart, Gateway, Placement и распределённого Storage.
Поэтому этот документ фиксирует порядок проектирования и критерии проверки, а не
copy/paste deployment.

## Перед началом

1. Реализовать/проверить контейнеры всех реально разворачиваемых сервисов.
2. Зафиксировать Storage replication protocol и failure domains.
3. Выбрать managed либо self-hosted PostgreSQL, Kafka, Redis и Neo4j.
4. Определить внешний authentication (SigV4 или явно собственный API), TLS и
   secret management.
5. Снять requests/limits из нагрузочных тестов, а не из предварительных оценок.

## Workload types

| Компонент | Kubernetes object | Условие |
|---|---|---|
| Gateway/Auth/AuthZ/Metadata/Users | Deployment | state находится во внешних зависимостях |
| Quota | Deployment, 1 active replica | до появления согласованного shared accounting |
| Storage | StatefulSet | stable identity + отдельный PV на replica |
| Placement | Deployment + durable control plane | после реализации epochs/repair state |
| stateful dependencies | managed service или официальный/проверенный chart | topology должна соответствовать quorum |

Для любого StatefulSet обязательны совпадающие `spec.selector.matchLabels` и
`spec.template.metadata.labels`. Storage также нужен headless Service. Реплики
распределяются через pod anti-affinity/topology spread по реальным failure zones.

Нельзя помещать majority одного quorum на одну node: два из трёх Kafka/Neo4j
members на общем failure domain теряют большинство при одном отказе. PostgreSQL
primary/replica и Redis master/replica/Sentinel также разносятся. Один Sentinel не
создаёт failover quorum.

## Варианты размещения и сети

- **Минимальный стенд:** один namespace и обычные Services; проще запустить, но
  он не проверяет multi-zone отказоустойчивость.
- **Headless Service + client-side gRPC balancing:** клиент видит pods и сам
  распределяет соединения; нужен корректный resolver и health handling.
- **Service mesh:** даёт traffic policy и mTLS, но добавляет эксплуатационную
  сложность; нужен только после появления конкретных требований.

Stateful dependencies можно поднять проверенными charts/operators либо вынести в
managed services. Выбор зависит от учебной цели: изучение эксплуатации кластера
или проверка поведения самих OpenS3 services.

## Probes

Liveness отвечает только на вопрос, жив ли process. Readiness должна проверять то,
без чего pod не способен обслужить заявленный traffic, но не быть настолько
тяжёлой, чтобы усугублять outage.

Standard gRPC Health в текущих сервисах часто выставляется статически при старте.
Для Storage filesystem есть отдельный custom RPC. Нельзя утверждать, что стандартный
probe автоматически проверяет PostgreSQL, Neo4j или disk space: это нужно
реализовать и протестировать для каждого сервиса.

## Observability

Текущий стек отправляет OTLP в collector, который экспортирует метрики для
Prometheus. ServiceMonitor должен scrape-ить реальный Prometheus endpoint
collector/сервиса; нельзя добавлять порт `/metrics`, которого приложение не
открывает.

В `infra/otel/grafana/dashboards` уже есть dashboards. Перед переносом проверяются
metric names, labels, datasource и retention. Provisioned dashboards из Git и
ручное состояние в Grafana volume имеют разный lifecycle.

## CI/CD prerequisites

Рабочий GitHub Actions job обязан содержать:

- `runs-on`;
- checkout;
- registry login и минимальные permissions;
- immutable image tag/digest;
- build context из корня для текущих Dockerfiles;
- получение cluster credentials без committed secrets;
- проверку/render Helm и rollout status;
- environment protection для production.

До появления этих частей YAML в документе считается псевдокодом. Pipeline не
должен перечислять отсутствующие Gateway/Placement images как уже собираемые.

## High availability

Для каждого stateful dependency фиксируются:

- число voting members и допустимые отказы;
- topology spread и storage class behavior;
- backup/restore с фактической проверкой восстановления;
- PodDisruptionBudget для добровольных выселений;
- поведение при involuntary node loss и network partition;
- порядок upgrade/schema migration.

PDB ограничивает voluntary disruptions, но не защищает от node crash, direct pod
deletion и всех одновременных отказов. Bitnami `postgresql-ha` использует
repmgr/pgpool, а не Patroni; если нужен Patroni, выбирается другой operator/chart.

## Chaos tests

`kubectl drain` тестирует controlled eviction с учётом PDB; это не имитация crash.
Node failure тестируется отдельным выключением/chaos action. Network partition
нужно создавать через Chaos Mesh/NetworkChaos либо подготовленный privileged
environment; случайная команда iptables внутри application container может не
иметь binary/capability и не задаёт точный набор peer IP.

Минимальная матрица:

| Сценарий | Что проверяется |
|---|---|
| pod deletion | restart, readiness, сохранность PV |
| voluntary drain | PDB и rescheduling |
| abrupt node loss | quorum/failover и время восстановления |
| one-way/two-way partition | fencing и отсутствие split brain |
| Kafka/DB outage | backpressure, outbox/retry, отсутствие ложного успеха |
| lost client response | idempotent retry и тот же terminal result |

## Стоимость

Цена зависит от provider, региона, control-plane fee, зон, storage IOPS/snapshots,
load balancer, egress, logs и времени работы. Старые суммы без этих параметров
удалены: перед deployment нужно сохранить датированный расчёт из официального
calculator вместе с выбранными instance/storage types. Spot discount и выключение
окружения рассматриваются как сценарии, а не гарантированная цена.

## Этапы

1. Local manifests/Helm для реализованных сервисов и одной development replica.
2. Managed dependencies или проверенные charts с restore test.
3. Gateway/Placement/replicated Storage после утверждения их контрактов.
4. Multi-zone placement, PDB, NetworkPolicy, secrets и TLS.
5. CI build/push/deploy с protected environment.
6. Load/chaos/restore tests и runbooks до заявления production readiness.

Полезные первичные источники:
[StatefulSet](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/),
[Pod disruptions](https://kubernetes.io/docs/concepts/workloads/pods/disruptions/),
[Bitnami postgresql-ha](https://github.com/bitnami/charts/tree/main/bitnami/postgresql-ha).

## Эскиз структуры Helm chart

Сохранённый вариант организации будущего deployment (каталог пока не создан):

```text
infra/helm/opens3/
  Chart.yaml
  values.yaml                    общие имена/порты, image digests
  values-dev.yaml                одна replica, локальные зависимости
  values-ha.yaml                 topology и измеренные requests/limits
  templates/
    deployments.yaml             stateless services; Quota с одним writer
    storage-statefulset.yaml     identity, selector/labels, PVC templates
    services.yaml                gRPC Services и headless Storage Service
    configmaps.yaml              несекретные env
    serviceaccounts.yaml         минимальные права
    networkpolicies.yaml         разрешённые межсервисные соединения
    pdb.yaml                     budgets для добровольных disruption
```

Это decomposition проекта, не обещание совместимого рабочего chart. Secrets
поступают через выбранный secret manager; пароли из локального `.env` не должны
становиться содержимым production values. Миграции Metadata/Users выполняются
отдельным контролируемым шагом до переключения приложений.

## Пример размещения по failure domains

Для исследования отказа одного домена можно взять три независимых домена A/B/C:

| Компонент | A | B | C |
|---|---|---|---|
| Storage при N=3 | replica 1 | replica 2 | replica 3 |
| Kafka при трёх members | member 1 | member 2 | member 3 |
| Neo4j при выбранной трёхголосной topology | voter 1 | voter 2 | voter 3 |
| PostgreSQL | primary | standby | место для восстановления/другой конфигурации |
| Redis при выбранном Sentinel deployment | primary + sentinel 1 | replica + sentinel 2 | sentinel 3 |

Таблица не задаёт число Kubernetes nodes или production sizing. Домен может быть
зоной с несколькими узлами; каждую зависимость нужно конфигурировать согласно её
реальному replication/failover протоколу. Наличие трёх pods само по себе не
включает репликацию, quorum или failover. Quota остаётся single writer даже при HA Redis.

## Последовательность delivery

```text
CI: checkout → tests → build из нужного context → registry push с immutable digest
Deploy: получить credentials → render/validate chart → migrations
        → rollout → dependency/readiness checks → smoke → зафиксировать результат
```

При неудачной migration откат image не гарантирует откат схемы. Rollback должен
быть описан отдельно для совместимых schema changes и для восстановления из backup.
В smoke входит внутренняя gRPC-проверка; внешний S3 smoke добавляется только после
появления Gateway. Сначала такой pipeline проверяется в dev namespace.
