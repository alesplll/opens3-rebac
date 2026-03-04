## 🎯 Назначение системы

**OpenS3-ReBAC** — распределённое объектное хранилище с S3 API и ReBAC авторизацией для хранения файлов в микросервисной архитектуре.

**Решает задачи:**
- Масштабируемое хранение файлов (бэкапы, логи, медиа, ML датасеты)
- S3-совместимое API для интеграции с boto3/aws-cli
- Гибкое управление доступом (владелец/команда/шеринг)

```
opens3-rebac/
│
├── proto/  
│   ├── authz/
│   │   └── v1/
│   │       └── authz.proto   
│   ├── metadata/
│   │   └── v1/
│   │       └── metadata.proto 
│   └── storage/
│       └── v1/
│           └── storage.proto  
│
├── services/
│   ├── authz/  
│   ├── metadata/ 
│   ├── storage/  
│   └── gateway/  
│
├── infra/      
├── .github/
├── docs/
├── .gitignore
└── README.md
```
