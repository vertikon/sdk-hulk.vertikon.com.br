# Atualização do SDK e compatibilidade com MCP Hulk

Data: 2026-10-05. Base do SDK: `3ec809fb238d550aaab7ed648d6e857b08a6d265`.
Checkout inicialmente limpo e igual ao `origin/main` consultado nesta execução.

## Mudanças

- `github.com/klauspost/compress`: 1.18.5 → 1.18.7.
- `golang.org/x/crypto`: 0.55.0 → 0.56.0; exige Go 1.26.0, registrado no `go.mod`.
- CI usa Go 1.27.1, a mesma toolchain da validação local, e inclui `govulncheck` 1.8.0.
- APIs públicas e comportamento do runtime não foram alterados.

Versões corrigidas conferidas no proxy Go e nos avisos oficiais:
[compress GO-2026-5841](https://pkg.go.dev/vuln/GO-2026-5841),
[crypto GO-2026-6354](https://pkg.go.dev/vuln/GO-2026-6354) e
[crypto GO-2026-6355](https://pkg.go.dev/vuln/GO-2026-6355).

## Evidência local

Os registros JSON contêm comando, diretório, data UTC, exit real e hashes dos logs.
O executor usado é `B:\mcp-hulk.vertikon.com.br\scripts\audit_evidence.py`.
[Resumo](summary.json) identifica os hashes de go.mod, go.sum e workflow avaliados.

| Gate | Resultado | Registro |
|---|---|---|
| Build | Exit 0 | [build](build.json) |
| Vet | Exit 0 | [vet](vet.json) |
| Staticcheck 2026.2.1 | Exit 0 | [staticcheck](staticcheck.json) |
| Suíte SDK | 234 PASS, 0 FAIL, 1 SKIP; 16 pacotes testados | [unit](unit.json) |
| Discovery com NATS local isolado | TestClientPublishes: 1 PASS, 0 FAIL/SKIP | [discovery-nats](discovery-nats.json) |
| Serviço Go gerado | Geração por MCP, tidy/build e 9 PASS, 0 FAIL/SKIP | [consumer](consumer.json) |
| Segurança | 0 chamados/importados; 1 aviso de módulo residual | [updated-vuln](updated-vuln.json) |

O único SKIP da suíte é `TestClientPublishes`, porque a execução inicial não
configurava `INVENTORY_TEST_NATS_URL`. Ele foi executado separadamente contra um
broker novo em loopback e porta efêmera, encerrado ao final. Seis pacotes da suíte
e quatro do consumidor não têm testes; não são testes individuais pulados.
Contagens incluem subtestes. O CI padrão continua sem esse broker de integração.

O controle antes/depois é o scanner sobre o mesmo código: o baseline tinha os
quatro avisos de módulo `GO-2026-5841`, `GO-2026-5932`, `GO-2026-6354` e
`GO-2026-6355`. Após a atualização resta apenas `GO-2026-5932`, de OpenPGP,
sem pacote importado ou símbolo chamado. O código não tinha achado alcançável no
baseline; não se apresenta essa atualização como correção de exploração
demonstrada. Base de vulnerabilidades consultada: 2026-10-01T20:24:15Z.

## Consumo e publicação

O probe [consumer_probe.py](consumer_probe.py) usa o MCP instalado em
`B:\mcp-hulk.vertikon.com.br\bin\mcp-server.exe`, SHA-256
`1b8b976275ea10aec7736caea5bd08a61854b63972adc71320f57657f24b6d52`, release
`c79d8a1`. Gera um serviço em diretório temporário, aponta seu módulo SDK para
este checkout por `replace` e executa os testes de identidade com JWKS sintético:
audience, issuer, assinatura, expiração, tenant suspenso e tenant assinado.
Não acessa credenciais, IdP ou recursos de produção.

Este repositório é distribuído como módulo Go: atualizar seu código não exige
implantar um daemon. O template Go publicado pelo MCP continua fixado no SDK
`v0.0.0-20260924202050-512d0df79526`; o teste prova compatibilidade local com a
atualização, não adoção automática pelos consumidores. Atualizar esses pins e
recompilar consumidores é uma etapa distinta.
