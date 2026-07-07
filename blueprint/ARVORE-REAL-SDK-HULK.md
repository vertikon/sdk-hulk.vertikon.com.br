# ARVORE REAL — SDK-HULK

**Gerado em:** 2025-11-24 20:20:28

```
.
├── ai
│   ├── ai_test.go
│   ├── client.go
│   ├── llm.go
│   └── memory.go
├── blueprint
│   ├── AUDITORIA-DE-CONFORMIDADE-SDK-HULK.md
│   ├── DOCUMENTACAO-TECNICA-SDK-HULK.md
│   └── templates_sdk-hulk-v1.md
├── cmd
│   └── main.go
├── docs
│   ├── gaps
│   │   ├── .report-counter.json
│   │   ├── CLAUDE_CODE_GUIDE-v1.md
│   │   ├── CLAUDE_CODE_GUIDE-v10.md
│   │   ├── CLAUDE_CODE_GUIDE-v11.md
│   │   ├── CLAUDE_CODE_GUIDE-v12.md
│   │   ├── CLAUDE_CODE_GUIDE-v13.md
│   │   ├── CLAUDE_CODE_GUIDE-v14.md
│   │   ├── CLAUDE_CODE_GUIDE-v15.md
│   │   ├── CLAUDE_CODE_GUIDE-v16.md
│   │   ├── CLAUDE_CODE_GUIDE-v2.md
│   │   ├── CLAUDE_CODE_GUIDE-v3.md
│   │   ├── CLAUDE_CODE_GUIDE-v4.md
│   │   ├── CLAUDE_CODE_GUIDE-v5.md
│   │   ├── CLAUDE_CODE_GUIDE-v6.md
│   │   ├── CLAUDE_CODE_GUIDE-v7.md
│   │   ├── CLAUDE_CODE_GUIDE-v8.md
│   │   ├── CLAUDE_CODE_GUIDE-v9.md
│   │   ├── gaps-report-2025-11-24-v1.json
│   │   ├── gaps-report-2025-11-24-v10.json
│   │   ├── gaps-report-2025-11-24-v11.json
│   │   ├── gaps-report-2025-11-24-v12.json
│   │   ├── gaps-report-2025-11-24-v13.json
│   │   ├── gaps-report-2025-11-24-v14.json
│   │   ├── gaps-report-2025-11-24-v15.json
│   │   ├── gaps-report-2025-11-24-v16.json
│   │   ├── gaps-report-2025-11-24-v2.json
│   │   ├── gaps-report-2025-11-24-v3.json
│   │   ├── gaps-report-2025-11-24-v4.json
│   │   ├── gaps-report-2025-11-24-v5.json
│   │   ├── gaps-report-2025-11-24-v6.json
│   │   ├── gaps-report-2025-11-24-v7.json
│   │   ├── gaps-report-2025-11-24-v8.json
│   │   └── gaps-report-2025-11-24-v9.json
│   ├── melhorias
│   │   ├── .report-counter.json
│   │   ├── relatorio-validacao-2025-11-24-v1.md
│   │   ├── relatorio-validacao-2025-11-24-v10.md
│   │   ├── relatorio-validacao-2025-11-24-v11.md
│   │   ├── relatorio-validacao-2025-11-24-v12.md
│   │   ├── relatorio-validacao-2025-11-24-v13.md
│   │   ├── relatorio-validacao-2025-11-24-v14.md
│   │   ├── relatorio-validacao-2025-11-24-v15.md
│   │   ├── relatorio-validacao-2025-11-24-v16.md
│   │   ├── relatorio-validacao-2025-11-24-v2.md
│   │   ├── relatorio-validacao-2025-11-24-v3.md
│   │   ├── relatorio-validacao-2025-11-24-v4.md
│   │   ├── relatorio-validacao-2025-11-24-v5.md
│   │   ├── relatorio-validacao-2025-11-24-v6.md
│   │   ├── relatorio-validacao-2025-11-24-v7.md
│   │   ├── relatorio-validacao-2025-11-24-v8.md
│   │   └── relatorio-validacao-2025-11-24-v9.md
│   ├── validation
│   │   └── raw
│   │       ├── .linter-counter.json
│   │       ├── 2025-11-24-16-54-53-compilation.log
│   │       ├── 2025-11-24-16-55-00-tests.log
│   │       ├── 2025-11-24-17-01-38-compilation.log
│   │       ├── 2025-11-24-17-12-03-compilation.log
│   │       ├── 2025-11-24-17-42-51-compilation.log
│   │       ├── 2025-11-24-18-32-14-compilation.log
│   │       ├── 2025-11-24-18-43-24-compilation.log
│   │       ├── govet.txt
│   │       ├── lint-temp.json
│   │       ├── lint-v1.json
│   │       ├── lint-v1.sarif
│   │       ├── lint-v10.json
│   │       ├── lint-v10.sarif
│   │       ├── lint-v11.json
│   │       ├── lint-v11.sarif
│   │       ├── lint-v12.json
│   │       ├── lint-v12.sarif
│   │       ├── lint-v13.json
│   │       ├── lint-v13.sarif
│   │       ├── lint-v14.json
│   │       ├── lint-v14.sarif
│   │       ├── lint-v15.json
│   │       ├── lint-v15.sarif
│   │       ├── lint-v16.json
│   │       ├── lint-v16.sarif
│   │       ├── lint-v2.json
│   │       ├── lint-v2.sarif
│   │       ├── lint-v3.json
│   │       ├── lint-v3.sarif
│   │       ├── lint-v4.json
│   │       ├── lint-v4.sarif
│   │       ├── lint-v5.json
│   │       ├── lint-v5.sarif
│   │       ├── lint-v6.json
│   │       ├── lint-v6.sarif
│   │       ├── lint-v7.json
│   │       ├── lint-v7.sarif
│   │       ├── lint-v8.json
│   │       ├── lint-v8.sarif
│   │       ├── lint-v9.json
│   │       ├── lint-v9.sarif
│   │       └── staticcheck.txt
│   └── NATS_SUBJECTS.md
├── events
│   ├── bus.go
│   ├── events_test.go
│   ├── publisher.go
│   └── subscriber.go
├── examples
│   ├── docs
│   │   └── validation
│   │       └── raw
│   │           ├── govet.txt
│   │           ├── lint-temp.json
│   │           └── staticcheck.txt
│   ├── inventory_module
│   │   └── module.go
│   ├── simple_module
│   │   └── module.go
│   └── README.md
├── internal
│   └── health
│       ├── health.go
│       └── health_test.go
├── pkg
│   └── sdk-hulk
├── state
│   ├── cache.go
│   ├── repository.go
│   ├── state_test.go
│   └── store.go
├── telemetry
│   └── logger.go
├── .gitignore
├── context.go
├── coverage
├── DEPENDENCIAS-SDK-HULK.md
├── go.mod
├── go.sum
├── module.go
├── module_test.go
├── README.md
├── sdk-hulk.code-workspace
└── state_coverage
```
